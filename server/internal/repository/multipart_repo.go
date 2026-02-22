package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pranavdhawale/bytefile/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MultipartRepository struct {
	collection *mongo.Collection
}

// NewMultipartRepository creates a repository for MultipartSession entities.
func NewMultipartRepository(db *mongo.Database) *MultipartRepository {
	return &MultipartRepository{
		collection: db.Collection("multipart_sessions"),
	}
}

// InitializeIndexes creates the TTL index on the expires_at field.
func (r *MultipartRepository) InitializeIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}

	name, err := r.collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create TTL index for multipart_sessions: %w", err)
	}

	fmt.Printf("MultipartSession index initialized: %s\n", name)
	return nil
}

// FindExpired returns a list of multipart sessions whose expires_at time has passed.
func (r *MultipartRepository) FindExpired(ctx context.Context, limit int) ([]*models.MultipartSession, error) {
	now := time.Now()
	filter := bson.M{"expires_at": bson.M{"$lte": now}}
	opts := options.Find().SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to cursor expired sessions: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []*models.MultipartSession
	if err := cursor.All(ctx, &sessions); err != nil {
		return nil, fmt.Errorf("failed to decode expired sessions: %w", err)
	}

	return sessions, nil
}

// ExistsByUploadID checks if a multipart session exists by its S3 upload_id.
func (r *MultipartRepository) ExistsByUploadID(ctx context.Context, uploadID string) (bool, error) {
	filter := bson.M{"upload_id": uploadID}
	err := r.collection.FindOne(ctx, filter, options.FindOne().SetProjection(bson.M{"_id": 1})).Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}
	return true, nil
}

// Insert creates a new MultipartSession document.
func (r *MultipartRepository) Insert(ctx context.Context, session *models.MultipartSession) error {
	_, err := r.collection.InsertOne(ctx, session)
	return err
}

// GetByID gets the session.
func (r *MultipartRepository) GetByID(ctx context.Context, id string) (*models.MultipartSession, error) {
	var s models.MultipartSession
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&s)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Found nothing
		}
		return nil, err
	}
	return &s, nil
}

// Delete removes the multipart session, typically when upload completes.
func (r *MultipartRepository) Delete(ctx context.Context, id string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
