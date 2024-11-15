package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

const (
	userTableQuery = `
		SELECT id, first_name, last_name, login, password, 
		date_created, worksite, role FROM users
	`
)

func InitDB() error {
	connStr := "user=postgres password=vladko123 dbname=tatraweb sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("error opening database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("error connecting to database: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return nil
}

func UserToHTML(user User) string {
	return fmt.Sprintf(`
        <tr>
            <td>%d</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>
                <button onclick="showEditModal(%d, '%s')" class="edit-btn">Upraviť</button>
                <button onclick="showDeleteModal(%d, '%s')" class="delete-btn">Odstrániť</button>
            </td>
        </tr>
    `, user.ID, user.FirstName, user.LastName, user.Login, user.Role,
		user.DateCreated.Format("2006-01-02 15:04:05"),
		user.Worksite.String, user.ID, user.Login, user.ID, user.Login)
}

func FetchAllUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(userTableQuery)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching users: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Login,
			&user.Password,
			&user.DateCreated,
			&user.Worksite,
			&user.Role,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading user data: %v", err), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating users: %v", err), http.StatusInternalServerError)
		return
	}

	var html string
	html += `<tbody hx-get="/fetch-all-users" hx-target="#usersTable tbody" hx-swap="outerHTML">`
	for _, user := range users {
		html += UserToHTML(user)
	}
	html += `</tbody>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func AddUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	user, err := validateAndCreateUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := RegisterUserInDB(user); err != nil {
		http.Error(w, "Error adding user to database", http.StatusInternalServerError)
		return
	}

	FetchAllUsers(w, r)
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	var user User
	err := db.QueryRow(`
        SELECT id, first_name, last_name, login, role, worksite 
        FROM users WHERE id = $1`, userID,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Login, &user.Role, &user.Worksite)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":        user.ID,
		"firstName": user.FirstName,
		"lastName":  user.LastName,
		"login":     user.Login,
		"role":      user.Role,
		"worksite":  user.Worksite.String,
	})
}

func EditUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	userID := r.FormValue("userId")
	firstName := r.FormValue("firstName")
	lastName := r.FormValue("lastName")
	login := r.FormValue("login")
	position := r.FormValue("position")
	worksite := r.FormValue("worksite")
	password := r.FormValue("password")

	var query string
	var args []interface{}

	if password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error processing the password", http.StatusInternalServerError)
			return
		}
		query = `
            UPDATE users 
            SET first_name = $1, last_name = $2, login = $3, 
                role = $4, worksite = $5, password = $6
            WHERE id = $7`
		args = []interface{}{firstName, lastName, login, position, worksite, hashedPassword, userID}
	} else {
		query = `
            UPDATE users 
            SET first_name = $1, last_name = $2, login = $3, 
                role = $4, worksite = $5
            WHERE id = $6`
		args = []interface{}{firstName, lastName, login, position, worksite, userID}
	}

	_, err := db.Exec(query, args...)
	if err != nil {
		http.Error(w, "Error updating user", http.StatusInternalServerError)
		return
	}

	FetchAllUsers(w, r)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	userID := r.FormValue("userId")
	confirmLogin := r.FormValue("confirmLogin")
	expectedLogin := r.FormValue("expectedLogin")

	if confirmLogin != expectedLogin {
		http.Error(w, "Login confirmation does not match", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}

	FetchAllUsers(w, r)
}

func validateAndCreateUser(r *http.Request) (*User, error) {
	firstName := r.FormValue("firstName")
	lastName := r.FormValue("lastName")
	login := r.FormValue("login")
	password := r.FormValue("password")
	position := r.FormValue("position")
	worksite := r.FormValue("worksite")

	if firstName == "" || lastName == "" || login == "" || password == "" || position == "" {
		return nil, fmt.Errorf("all fields are required")
	}

	if position == "worker" && worksite == "" {
		return nil, fmt.Errorf("workplace is required for worker")
	}

	user := &User{
		FirstName:   firstName,
		LastName:    lastName,
		Login:       login,
		Password:    password,
		DateCreated: time.Now(),
		Role:        position,
		Worksite: sql.NullString{
			String: worksite,
			Valid:  worksite != "",
		},
	}

	return user, nil
}

func ProductToHTML(product Product) string {

	return fmt.Sprintf(`
        <tr>
            <td>%s</td>
            <td>%s</td>
            <td>%d %s</td>
			<td>
                <button onclick="showEditModal(%d, '%s')" class="edit-btn">Upraviť</button>
                <button onclick="showDeleteModal(%d, '%s')" class="delete-btn">Odstrániť</button>
            </td>
        </tr>
    `, product.KC, product.Name, product.Weight, product.WeightType, product.ID,
	)
}

func FetchAllProducts(w http.ResponseWriter, r *http.Request) {

	rows, err := db.Query(`
        SELECT id, kc, name, weight, weight_type, created_by, created_at, updated_at 
        FROM products
        ORDER BY kc
    `)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching products: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var product Product
		err := rows.Scan(
			&product.ID,
			&product.KC,
			&product.Name,
			&product.Weight,
			&product.WeightType,
			&product.CreatedBy,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading product data: %v", err), http.StatusInternalServerError)
			return
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating products: %v", err), http.StatusInternalServerError)
		return
	}

	var html string
	html += `<tbody hx-get="/fetch-all-products" hx-target="#productsTable tbody" hx-swap="outerHTML">`
	for _, product := range products {
		html += ProductToHTML(product)
	}
	html += `</tbody>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func AddProduct(w http.ResponseWriter, r *http.Request) {
	log.Println("mnau")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	userID := GetUserIDFromSession(r)

	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	product, err := ValidateAndCreateProduct(r, userID)
	if err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := RegisterProductInDB(product); err != nil {
		http.Error(w, "Error adding product to database", http.StatusInternalServerError)
		return
	}

	FetchAllProducts(w, r)
}

func ValidateAndCreateProduct(r *http.Request, userID int) (*Product, error) {
	kc := r.FormValue("kc")
	name := r.FormValue("name")
	weightVolumeStr := r.FormValue("weightVolume")
	weightType := r.FormValue("weightType")

	if kc == "" || name == "" || weightVolumeStr == "" || weightType == "" {
		return nil, fmt.Errorf("all fields are required")
	}

	weight, err := strconv.Atoi(weightVolumeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid weight/volume value")
	}

	if weight <= 0 {
		return nil, fmt.Errorf("weight/volume must be greater than 0")
	}

	product := &Product{
		KC:         kc,
		Name:       name,
		Weight:     weight,
		WeightType: weightType,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	return product, nil
}

func RegisterProductInDB(product *Product) error {
	query := `
        INSERT INTO products (
            kc, name, weight, weight_type, created_by, created_at, updated_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

	_, err := db.Exec(query,
		product.KC,
		product.Name,
		product.Weight,
		product.WeightType,
		product.CreatedBy,
		product.CreatedAt,
		product.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("error inserting product: %v", err)
	}

	return nil
}

func GetUserIDFromSession(r *http.Request) int {
	cookie, err := r.Cookie("jwt_token")
	if err != nil {
		return 0
	}
	tokenString := strings.TrimPrefix(cookie.Value, "Bearer ")
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})

	if err != nil {
		return 0
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {

		if !claims.ExpiresAt.After(time.Now()) {
			return 0
		}
		return claims.UserID
	}

	return 0
}

func EditProduct(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	userID := r.FormValue("userId")

	FetchAllProducts(w, r)
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	userID := r.FormValue("userId")
	confirmLogin := r.FormValue("confirmLogin")
	expectedLogin := r.FormValue("expectedLogin")

	if confirmLogin != expectedLogin {
		http.Error(w, "Login confirmation does not match", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}

	FetchAllUsers(w, r)
}
