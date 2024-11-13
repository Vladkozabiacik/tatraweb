package main

import (
	"database/sql"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID          int
	FirstName   string
	LastName    string
	Login       string
	Password    string
	DateCreated time.Time
	Worksite    sql.NullString // Sypké, Poživatiny, Kozmetika, Sklad,
	Role        string         // Manager, Admin, Salesman, Worker
}

type Claims struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	Worksite  string `json:"worksite,omitempty"`
	jwt.RegisteredClaims
}