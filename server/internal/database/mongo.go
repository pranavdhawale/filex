package database

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Connect connects to the MongoDB server at the given URI.
func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
	slog.Debug("Connecting to MongoDB", "uri", uri)

	clientOptions := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping the primary to verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	slog.Info("Successfully connected to MongoDB")
	return client, nil
}

// Close gracefully closes the MongoDB connection.
func Close(ctx context.Context, client *mongo.Client) error {
	slog.Info("Closing MongoDB connection")
	return client.Disconnect(ctx)
}

// GetDatabase returns the bytefile database instance.
func GetDatabase(client *mongo.Client) *mongo.Database {
	return client.Database("bytefile")
}

// ContextWithTimeout is a helper for repos.
func ContextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
