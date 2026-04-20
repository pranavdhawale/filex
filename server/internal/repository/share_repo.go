package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pranavdhawale/filex/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ShareRepository struct {
	collection *mongo.Collection
}

func NewShareRepository(db *mongo.Database) *ShareRepository {
	return &ShareRepository{collection: db.Collection("shares")}
}

func (r *ShareRepository) InitializeIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *ShareRepository) Insert(ctx context.Context, s *models.Share) error {
	_, err := r.collection.InsertOne(ctx, s)
	return err
}

func (r *ShareRepository) GetBySlug(ctx context.Context, slug string) (*models.Share, error) {
	var s models.Share
	err := r.collection.FindOne(ctx, bson.M{"slug": slug}).Decode(&s)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *ShareRepository) FindExpired(ctx context.Context, limit int) ([]*models.Share, error) {
	cursor, err := r.collection.Find(ctx,
		bson.M{"expires_at": bson.M{"$lte": time.Now()}},
		options.Find().SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, fmt.Errorf("find expired shares: %w", err)
	}
	defer cursor.Close(ctx)
	var shares []*models.Share
	if err := cursor.All(ctx, &shares); err != nil {
		return nil, fmt.Errorf("decode expired shares: %w", err)
	}
	return shares, nil
}

func (r *ShareRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *ShareRepository) BulkIncrementDownloadCounts(ctx context.Context, counts map[string]int64) error {
	writeModels := make([]mongo.WriteModel, 0, len(counts))
	for slug, count := range counts {
		writeModels = append(writeModels, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"slug": slug}).
			SetUpdate(bson.M{"$inc": bson.M{"download_count": count}}))
	}
	if len(writeModels) == 0 {
		return nil
	}
	_, err := r.collection.BulkWrite(ctx, writeModels)
	return err
}

func (r *ShareRepository) GetFilesByShareID(ctx context.Context, shareID bson.ObjectID) ([]*models.File, error) {
	cursor, err := r.collection.Database().Collection("files").Find(ctx,
		bson.M{"share_id": shareID},
	)
	if err != nil {
		return nil, fmt.Errorf("find files by share: %w", err)
	}
	defer cursor.Close(ctx)
	var files []*models.File
	if err := cursor.All(ctx, &files); err != nil {
		return nil, fmt.Errorf("decode files by share: %w", err)
	}
	return files, nil
}