package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoConfig struct {
	URI      string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	Timeout  time.Duration
}

func NewMongoConnection(cfg MongoConfig) (*mongo.Client, error) {
	connectionTimeout := cfg.Timeout
	if connectionTimeout == 0 {
		connectionTimeout = 10 * time.Second
	}

	mongoURI := cfg.URI
	if mongoURI == "" {
		if cfg.User != "" && cfg.Password != "" {
			mongoURI = fmt.Sprintf("mongodb://%s:%s@%s:%s", cfg.User, cfg.Password, cfg.Host, cfg.Port)
		} else {
			mongoURI = fmt.Sprintf("mongodb://%s:%s", cfg.Host, cfg.Port)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	log.Printf("Connecting to MongoDB: %s/%s", mongoURI, cfg.DBName)

	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer pingCancel()
	err = client.Ping(pingCtx, readpref.Primary())
	if err != nil {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer disconnectCancel()
		_ = client.Disconnect(disconnectCtx)
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	log.Println("Successfully connected and pinged MongoDB!")
	return client, nil
}

func GetMongoDatabase(client *mongo.Client, dbName string) *mongo.Database {
	return client.Database(dbName)
}
