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

type FileRepository struct {
	collection *mongo.Collection
}

func NewFileRepository(db *mongo.Database) *FileRepository {
	return &FileRepository{collection: db.Collection("files")}
}

func (r *FileRepository) InitializeIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "object_key", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *FileRepository) Insert(ctx context.Context, f *models.File) error {
	_, err := r.collection.InsertOne(ctx, f)
	return err
}

func (r *FileRepository) GetBySlug(ctx context.Context, slug string) (*models.File, error) {
	var f models.File
	err := r.collection.FindOne(ctx, bson.M{"slug": slug}).Decode(&f)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *FileRepository) GetByID(ctx context.Context, id bson.ObjectID) (*models.File, error) {
	var f models.File
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&f)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *FileRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	err := r.collection.FindOne(ctx, bson.M{"slug": slug}, options.FindOne().SetProjection(bson.M{"_id": 1})).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *FileRepository) FindExpired(ctx context.Context, limit int) ([]*models.File, error) {
	cursor, err := r.collection.Find(ctx,
		bson.M{"expires_at": bson.M{"$lte": time.Now()}},
		options.Find().SetLimit(int64(limit)),
	)
	if err != nil {
		return nil, fmt.Errorf("find expired files: %w", err)
	}
	defer cursor.Close(ctx)
	var files []*models.File
	if err := cursor.All(ctx, &files); err != nil {
		return nil, fmt.Errorf("decode expired files: %w", err)
	}
	return files, nil
}

func (r *FileRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *FileRepository) ExistsByObjectKey(ctx context.Context, objectKey string) (bool, error) {
	err := r.collection.FindOne(ctx, bson.M{"object_key": objectKey}, options.FindOne().SetProjection(bson.M{"_id": 1})).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *FileRepository) UpdateByID(ctx context.Context, id bson.ObjectID, update bson.M) error {
	_, err := r.collection.UpdateByID(ctx, id, update)
	return err
}

func (r *FileRepository) SetShareID(ctx context.Context, fileID bson.ObjectID, shareID bson.ObjectID) error {
	_, err := r.collection.UpdateByID(ctx, fileID, bson.M{"$set": bson.M{"share_id": shareID}})
	return err
}

func (r *FileRepository) BulkIncrementDownloadCounts(ctx context.Context, counts map[string]int64) error {
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