package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusCompleted OrderStatus = "completed"
	StatusCancelled OrderStatus = "cancelled"
	StatusFailed    OrderStatus = "failed"
)

type Order struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      string             `json:"user_id" bson:"user_id" binding:"required"`
	Items       []OrderItem        `json:"items" bson:"items" binding:"required,dive"`
	TotalAmount float64            `json:"total_amount" bson:"total_amount"`
	Status      OrderStatus        `json:"status" bson:"status"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

type CreateOrderInput struct {
	UserID string           `json:"user_id" binding:"required"`
	Items  []OrderItemInput `json:"items" binding:"required,min=1,dive"`
}

type OrderItemInput struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,gt=0"`
}

type UpdateOrderStatusInput struct {
	Status OrderStatus `json:"status" binding:"required"`
}
