package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestFetchAllUsers(t *testing.T) {
	setup()
	defer teardown()

	rows := sqlmock.NewRows([]string{"id", "first_name", "last_name", "login", "password", "date_created", "worksite", "role"}).
		AddRow(1, "John", "Doe", "johndoe", "password123", time.Now(), "HQ", "manager").
		AddRow(2, "Jane", "Smith", "janesmith", "password456", time.Now(), "SiteA", "worker")

	mock.ExpectQuery(`SELECT id, first_name, last_name, login, password, date_created, worksite, role FROM users`).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/fetch-all-users", nil)
	rr := httptest.NewRecorder()

	fetchAllUsers(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "John")
	assert.Contains(t, rr.Body.String(), "Doe")
	assert.Contains(t, rr.Body.String(), "johndoe")
	assert.Contains(t, rr.Body.String(), "manager")
	assert.Contains(t, rr.Body.String(), "HQ")

	err := mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAddUser_Success(t *testing.T) {
	setup()
	defer teardown()

	user := User{
		FirstName:   "Alice",
		LastName:    "Wonder",
		Login:       "alicewonder",
		Password:    "password789",
		DateCreated: time.Now(),
		Worksite:    sql.NullString{String: "HQ", Valid: true},
		Role:        "worker",
	}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO users (first_name, last_name, login, password, date_created, worksite, role) VALUES`).
		WithArgs(user.FirstName, user.LastName, user.Login, user.Password, user.DateCreated, user.Worksite, user.Role).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPost, "/add-user", nil)
	req.PostForm = map[string][]string{
		"firstName": {"Alice"},
		"lastName":  {"Wonder"},
		"login":     {"alicewonder"},
		"password":  {"password789"},
		"position":  {"worker"},
		"worksite":  {"HQ"},
	}
	rr := httptest.NewRecorder()

	addUser(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response User
	err := json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", response.FirstName)
	assert.Equal(t, "Wonder", response.LastName)
	assert.Equal(t, "alicewonder", response.Login)
	assert.Equal(t, "worker", response.Role)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAddUser_MissingFields(t *testing.T) {
	setup()
	defer teardown()

	req := httptest.NewRequest(http.MethodPost, "/add-user", nil)
	req.PostForm = map[string][]string{
		"firstName": {"Alice"},
		"lastName":  {"Wonder"},
		"login":     {"alicewonder"},
		"password":  {"password789"},
		"position":  {"worker"},
	}
	rr := httptest.NewRecorder()

	addUser(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "All fields are required")
}

func TestAddUser_WorksiteRequiredForWorker(t *testing.T) {
	setup()
	defer teardown()

	req := httptest.NewRequest(http.MethodPost, "/add-user", nil)
	req.PostForm = map[string][]string{
		"firstName": {"Alice"},
		"lastName":  {"Wonder"},
		"login":     {"alicewonder"},
		"password":  {"password789"},
		"position":  {"worker"},
	}
	rr := httptest.NewRecorder()

	addUser(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Workplace is required for worker")
}
