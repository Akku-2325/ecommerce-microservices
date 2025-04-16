package repository

import (
	"context"
	"ecommerce-microservices/inventory-service/internal/domain"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const productCollectionName = "products"

type MongoProductStore struct {
	collection *mongo.Collection
}

func NewMongoProductStore(db *mongo.Database) *MongoProductStore {
	collection := db.Collection(productCollectionName)
	return &MongoProductStore{collection: collection}
}

func (s *MongoProductStore) Create(ctx context.Context, product *domain.Product) error {
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	result, err := s.collection.InsertOne(ctx, product)
	if err != nil {
		return fmt.Errorf("failed to insert product: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		product.ID = oid
	}
	log.Printf("Inserted product with ID: %v", result.InsertedID)
	return nil
}

func (s *MongoProductStore) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id format: %w", err)
	}

	var product domain.Product
	err = s.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&product)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to find product: %w", err)
	}
	return &product, nil
}

func (s *MongoProductStore) Update(ctx context.Context, id string, product *domain.Product) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	filter := bson.M{"_id": objID}
	update := bson.M{
		"$set": bson.M{
			"name":        product.Name,
			"description": product.Description,
			"price":       product.Price,
			"stock":       product.Stock,
			"category_id": product.CategoryID,
			"updated_at":  time.Now(),
		},
	}

	result, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("product not found to update")
	}
	log.Printf("Updated product ID: %s, Matched: %d, Modified: %d", id, result.MatchedCount, result.ModifiedCount)
	return nil
}

func (s *MongoProductStore) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	filter := bson.M{"_id": objID}
	result, err := s.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("product not found to delete")
	}
	log.Printf("Deleted product ID: %s, Count: %d", id, result.DeletedCount)
	return nil
}

func (s *MongoProductStore) List(ctx context.Context, filter bson.M, limit, offset int64) ([]*domain.Product, int64, error) {
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
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	cursor, err := s.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}
	defer cursor.Close(ctx)

	var products []*domain.Product
	if err = cursor.All(ctx, &products); err != nil {
		return nil, 0, fmt.Errorf("failed to decode products: %w", err)
	}

	if products == nil {
		products = []*domain.Product{}
	}

	return products, totalCount, nil
}
