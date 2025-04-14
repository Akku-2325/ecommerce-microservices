package store

import (
	"context"
	"ecommerce-microservices/inventory-service/models"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const categoryCollectionName = "categories"

type MongoCategoryStore struct {
	collection *mongo.Collection
}

func NewMongoCategoryStore(db *mongo.Database) *MongoCategoryStore {
	collection := db.Collection(categoryCollectionName)
	return &MongoCategoryStore{collection: collection}
}

func (s *MongoCategoryStore) Create(ctx context.Context, category *models.Category) error {
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	result, err := s.collection.InsertOne(ctx, category)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("category with name '%s' already exists", category.Name)
		}
		return fmt.Errorf("failed to insert category: %w", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		category.ID = oid
	}
	log.Printf("Inserted category with ID: %v", result.InsertedID)
	return nil
}

func (s *MongoCategoryStore) GetByID(ctx context.Context, id string) (*models.Category, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id format: %w", err)
	}

	var category models.Category
	err = s.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&category)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("category not found") // Simple not found error
		}
		return nil, fmt.Errorf("failed to find category: %w", err)
	}
	return &category, nil
}

func (s *MongoCategoryStore) Update(ctx context.Context, id string, category *models.Category) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}

	filter := bson.M{"_id": objID}
	update := bson.M{
		"$set": bson.M{
			"name":       category.Name,
			"updated_at": time.Now(),
		},
	}

	result, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("category with name '%s' already exists", category.Name)
		}
		return fmt.Errorf("failed to update category: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("category not found to update")
	}
	log.Printf("Updated category ID: %s, Matched: %d, Modified: %d", id, result.MatchedCount, result.ModifiedCount)
	return nil
}

func (s *MongoCategoryStore) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id format: %w", err)
	}


	filter := bson.M{"_id": objID}
	result, err := s.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("category not found to delete")
	}
	log.Printf("Deleted category ID: %s, Count: %d", id, result.DeletedCount)
	return nil
}

func (s *MongoCategoryStore) List(ctx context.Context) ([]*models.Category, error) {
	cursor, err := s.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	defer cursor.Close(ctx)

	var categories []*models.Category
	if err = cursor.All(ctx, &categories); err != nil {
		return nil, fmt.Errorf("failed to decode categories: %w", err)
	}

	if categories == nil {
		return []*models.Category{}, nil // Return empty slice, not nil
	}

	return categories, nil
}

