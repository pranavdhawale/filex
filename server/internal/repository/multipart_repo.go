package repository

import (
	"context"
	"time"

	"github.com/pranavdhawale/filex/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MultipartRepository struct {
	collection *mongo.Collection
}

func NewMultipartRepository(db *mongo.Database) *MultipartRepository {
	return &MultipartRepository{collection: db.Collection("multipart_sessions")}
}

func (r *MultipartRepository) InitializeIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		{Keys: bson.D{{Key: "upload_id", Value: 1}}},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *MultipartRepository) Insert(ctx context.Context, s *models.MultipartSession) error {
	_, err := r.collection.InsertOne(ctx, s)
	return err
}

func (r *MultipartRepository) GetByID(ctx context.Context, id bson.ObjectID) (*models.MultipartSession, error) {
	var s models.MultipartSession
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&s)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *MultipartRepository) FindExpired(ctx context.Context, limit int) ([]*models.MultipartSession, error) {
	cursor, err := r.collection.Find(ctx,
		bson.M{"expires_at": bson.M{"$lte": time.Now()}},
		options.Find().SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var sessions []*models.MultipartSession
	if err := cursor.All(ctx, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *MultipartRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}