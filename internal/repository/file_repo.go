package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pranavdhawale/bytefile/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FileRepository struct {
	collection *mongo.Collection
}

// NewFileRepository creates a repository for File entities.
func NewFileRepository(db *mongo.Database) *FileRepository {
	return &FileRepository{
		collection: db.Collection("files"),
	}
}

// InitializeIndexes creates the TTL index on the expires_at field.
func (r *FileRepository) InitializeIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0), // Delete immediately when expires_at passes
	}

	name, err := r.collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create TTL index for files: %w", err)
	}

	// Add an index on object_key for fast lookups if needed by gc, though ID works mostly.
	_, _ = r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "object_key", Value: 1}},
	})

	fmt.Printf("File index initialized: %s\n", name)
	return nil
}

// Insert creates a new File document.
func (r *FileRepository) Insert(ctx context.Context, f *models.File) error {
	_, err := r.collection.InsertOne(ctx, f)
	return err
}

// GetByID explicitly retrieves the file by its UUID.
func (r *FileRepository) GetByID(ctx context.Context, id string) (*models.File, error) {
	var f models.File
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&f)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Found nothing
		}
		return nil, err
	}
	return &f, nil
}

// ExtendTTL atomically sets expires_at to max(currentExpiry, newExpiry) and updates last_accessed.
func (r *FileRepository) ExtendTTL(ctx context.Context, id string, newExpiry time.Time) error {
	update := bson.M{
		"$max": bson.M{
			"expires_at": newExpiry,
		},
		"$set": bson.M{
			"last_accessed": time.Now().UTC(),
		},
	}

	_, err := r.collection.UpdateByID(ctx, id, update)
	return err
}

// FindExpired returns theoretically expired files that the worker might need to delete from MinIO (as a fallback, MongoDB deletes them natively via TTL, but we might want pre-empts).
// But standard design just relies on Mongo TTL streams or polling. If we rely on TTL directly, we need Change Streams. Or poll for things about to expire.
