package models

type OrderItem struct {
	ProductID    string  `json:"product_id" bson:"product_id" binding:"required"`
	Quantity     int     `json:"quantity" bson:"quantity" binding:"required,gt=0"`
	PriceAtOrder float64 `json:"price_at_order" bson:"price_at_order"`
}
