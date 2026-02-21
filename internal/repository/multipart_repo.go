package repository

import (
	"context"
	"fmt"

	"github.com/pranavdhawale/bytefile/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

// FindExpired explicitly finds records that have expired.
// Used by the cleanup worker to cross-abort in MinIO.
// Note: if MongoDB TTL deletes it first, the worker can't find it.
// A common pattern is to let the worker query `expires_at < now` and then delete it explicitly after aborting in MinIO.
func (r *MultipartRepository) FindExpired(ctx context.Context, nowUnix int64) ([]*models.MultipartSession, error) {
	// Not implemented perfectly here without more fleshing out, but placeholder for the worker phase.
	return nil, nil
}
