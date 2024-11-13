package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

var mock sqlmock.Sqlmock

func setup() {
	var err error
	db, mock, err = sqlmock.New()
	if err != nil {
		panic(err)
	}
}

func teardown() {
	db.Close()
}

func TestRegisterUserInDB(t *testing.T) {
	setup()
	defer teardown()

	user := &User{
		FirstName: "John",
		LastName:  "Doe",
		Login:     "johndoe",
		Password:  "password123",
		Role:      "manager",
		Worksite:  sql.NullString{String: "HQ", Valid: "HQ" != ""},
	}

	mock.ExpectQuery(`INSERT INTO users (first_name, last_name, login, password, worksite, role) VALUES`).
		WithArgs(user.FirstName, user.LastName, user.Login, user.Password, user.Worksite, user.Role).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err := RegisterUserInDB(user)
	assert.NoError(t, err)
	assert.Equal(t, 1, user.ID)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestLogin_Success(t *testing.T) {
	setup()
	defer teardown()

	login := "johndoe"
	password := "password123"

	user := &User{
		ID:        1,
		FirstName: "John",
		LastName:  "Doe",
		Password:  "$2a$10$2pO.C8w09fEnMP9cyIDShsqAwOHbPqlkR6csUJm2QhKzU8LsugO2K",
		Role:      "manager",
		Worksite:  sql.NullString{String: "HQ", Valid: "HQ" != ""},
	}

	mock.ExpectQuery(`SELECT id, first_name, last_name, password, role, worksite FROM users WHERE login = \$1`).
		WithArgs(login).
		WillReturnRows(sqlmock.NewRows([]string{"id", "first_name", "last_name", "password", "role", "worksite"}).
			AddRow(user.ID, user.FirstName, user.LastName, user.Password, user.Role, user.Worksite))

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.Form = map[string][]string{
		"login":    {login},
		"password": {password},
	}

	rr := httptest.NewRecorder()

	Login(rr, req)

	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Contains(t, rr.Header().Get("Location"), "/manager")

	err := mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestIsLogged_InvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/is-logged", nil)
	req.AddCookie(&http.Cookie{Name: "jwt_token", Value: "invalid.token"})

	rr := httptest.NewRecorder()

	IsLogged(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestIsLogged_ValidToken(t *testing.T) {
	claims := &Claims{
		ID:        1,
		FirstName: "John",
		LastName:  "Doe",
		Role:      "manager",
		Worksite:  "HQ",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Issuer:    "tatraweb",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/is-logged", nil)
	req.AddCookie(&http.Cookie{Name: "jwt_token", Value: tokenString})

	rr := httptest.NewRecorder()

	IsLogged(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["loggedIn"])
	assert.Equal(t, "manager", response["role"])
}

func TestRequireRole_Success(t *testing.T) {
	setup()
	defer teardown()

	claims := &Claims{
		ID:        1,
		FirstName: "John",
		LastName:  "Doe",
		Role:      "admin",
		Worksite:  "HQ",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Issuer:    "tatraweb",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "jwt_token", Value: tokenString})

	rr := httptest.NewRecorder()

	handler := RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireRole_Forbidden(t *testing.T) {
	setup()
	defer teardown()

	claims := &Claims{
		ID:        1,
		FirstName: "John",
		LastName:  "Doe",
		Role:      "worker",
		Worksite:  "HQ",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Issuer:    "tatraweb",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "jwt_token", Value: tokenString})

	rr := httptest.NewRecorder()

	handler := RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
