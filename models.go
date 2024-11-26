package main

import (
	"database/sql"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Worksite struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
type User struct {
	ID       int
	Name     string
	Login    string
	Password string
	Worksite sql.NullString // Sypké, Poživatiny, Kozmetika, Sklad,
	Role     string         // Manager, Admin, Salesman, Worker
}

type JWTClaims struct {
	UserID int    `json:"id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type Claims struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Worksite string `json:"worksite,omitempty"`
	jwt.RegisteredClaims
}

type Product struct {
	ID   int    `json:"id"`
	KC   string `json:"kc"`
	Name string `json:"name"`
}

type Customer struct {
	CustomerID int    `json:"id"`
	Name       string `json:"name"`
}

type Order struct {
	ID           int         `json:"id"`
	CustomerID   int         `json:"customer_id"`
	CustomerName string      `json:"customer_name"`
	CreatedAt    string      `json:"created_at"`
	CreatedBy    string      `json:"created_by"`
	OrderItems   []OrderItem `json:"order_items"`
}

type OrderItem struct {
	ID           int    `json:"id"`
	ProductName  string `json:"product_name"`
	Quantity     int    `json:"quantity"`
	DeliveryDate string `json:"delivery_date"`
}

type ProductionOrder struct {
	ID             int           `json:"id"`
	OrderItem      OrderItem     `json:"order_item"`
	Worksite       string        `json:"worksite"`
	Status         string        `json:"status"`
	ProducedBy     sql.NullInt64 `json:"produced_by"`
	ProductionDate time.Time     `json:"production_date"`
	CreatedAt      time.Time     `json:"created_at"`
}
