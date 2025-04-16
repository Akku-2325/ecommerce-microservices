package repository

import (
	"context"
	"ecommerce-microservices/order-service/internal/domain"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const orderCollectionName = "orders"

type MongoOrderStore struct {
	collection *mongo.Collection
}

func NewMongoOrderStore(db *mongo.Database) *MongoOrderStore {
	collection := db.Collection(orderCollectionName)
	return &MongoOrderStore{collection: collection}
}

func (s *MongoOrderStore) Create(ctx context.Context, order *domain.Order) error {
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	result, err := s.collection.InsertOne(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		order.ID = oid
	}
	log.Printf("Inserted order with ID: %v for user %s", result.InsertedID, order.UserID)
	return nil
}

func (s *MongoOrderStore) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id format: %w", err)
	}

	var order domain.Order
	err = s.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&order)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to find order: %w", err)
	}
	return &order, nil
}

func (s *MongoOrderStore) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	filter := bson.M{"_id": objID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("order not found to update status")
	}
	log.Printf("Updated order status ID: %s to %s, Matched: %d, Modified: %d", id, status, result.MatchedCount, result.ModifiedCount)
	return nil
}

func (s *MongoOrderStore) ListByUserID(ctx context.Context, userID string, limit, offset int64) ([]*domain.Order, int64, error) {
	filter := bson.M{"user_id": userID}

	findOptions := options.Find()
	if limit > 0 {
		findOptions.SetLimit(limit)
	}
	if offset > 0 {
		findOptions.SetSkip(offset)
	}
	findOptions.SetSort(bson.D{{"created_at", -1}})

	totalCount, err := s.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count orders for user %s: %w", userID, err)
	}

	cursor, err := s.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders for user %s: %w", userID, err)
	}
	defer cursor.Close(ctx)

	var orders []*domain.Order
	if err = cursor.All(ctx, &orders); err != nil {
		return nil, 0, fmt.Errorf("failed to decode orders for user %s: %w", userID, err)
	}

	if orders == nil {
		orders = []*domain.Order{}
	}

	return orders, totalCount, nil
}
