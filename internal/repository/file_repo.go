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

// FindExpired returns a list of files whose expires_at time has passed.
func (r *FileRepository) FindExpired(ctx context.Context, limit int) ([]*models.File, error) {
	now := time.Now()
	filter := bson.M{"expires_at": bson.M{"$lte": now}}
	opts := options.Find().SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to cursor expired files: %w", err)
	}
	defer cursor.Close(ctx)

	var files []*models.File
	if err := cursor.All(ctx, &files); err != nil {
		return nil, fmt.Errorf("failed to decode expired files: %w", err)
	}

	return files, nil
}

// Delete removes a file document from the database by ID.
func (r *FileRepository) Delete(ctx context.Context, id string) error {
	filter := bson.M{"_id": id}
	_, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete file document: %w", err)
	}
	return nil
}

// ExistsByObjectKey checks if a specific object key exists in the files collection.
func (r *FileRepository) ExistsByObjectKey(ctx context.Context, objectKey string) (bool, error) {
	filter := bson.M{"object_key": objectKey}
	err := r.collection.FindOne(ctx, filter, options.FindOne().SetProjection(bson.M{"_id": 1})).Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence by object_key: %w", err)
	}
	return true, nil
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
