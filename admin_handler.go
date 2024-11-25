package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func UserToHTML(user User) string {
	return fmt.Sprintf(`
        <tr id="user-%d">
            <td>%d</td>
            <td class="editable-cell" data-field="name">%s</td>
            <td class="editable-cell" data-field="login">%s</td>
            <td class="editable-cell" data-field="password"></td>
            <td class="editable-cell" data-field="role">%s</td>
            <td class="editable-cell" data-field="worksite">%s</td>
            <td>
                <button onclick="enableRowEdit(this.closest('tr'), event)" class="edit-btn">Upraviť</button>
                <button onclick="deleteUser(%d)" class="delete-btn">Odstrániť</button>
            </td>
        </tr>
    `, user.ID, user.ID, user.Name, user.Login, user.Role,
		user.Worksite.String, user.ID)
}

func FetchAllUsers(w http.ResponseWriter, r *http.Request) {

	rows, err := db.Query(
		`SELECT id, name, login, password, 
		 worksite, role FROM users`,
	)
	if err != nil {
		log.Println(err)
		http.Error(w, fmt.Sprintf("Error fetching users: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Login,
			&user.Password,
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
	).Scan(&user.ID, &user.Name, &user.Login, &user.Role, &user.Worksite)

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
		"id":       user.ID,
		"name":     user.Name,
		"login":    user.Login,
		"role":     user.Role,
		"worksite": user.Worksite.String,
	})
}

func EditUser(w http.ResponseWriter, r *http.Request) {
	userID := r.FormValue("userId")
	name := r.FormValue("name")
	login := r.FormValue("login")
	role := r.FormValue("role")
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
            SET name = $1, login = $2, 
                role = $3, worksite = $4, password = $5
            WHERE id = $6`
		args = []interface{}{name, login, role, worksite, hashedPassword, userID}
	} else {
		query = `
            UPDATE users 
            SET name = $1, login = $2, 
                role = $3, worksite = $4
            WHERE id = $5`
		args = []interface{}{name, login, role, worksite, userID}
	}

	_, err := db.Exec(query, args...)
	if err != nil {
		http.Error(w, "Error updating user", http.StatusInternalServerError)
		return
	}

	FetchAllUsers(w, r)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	userID := strings.TrimSpace(string(body))

	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}

	FetchAllUsers(w, r)
}

func validateAndCreateUser(r *http.Request) (*User, error) {
	name := r.FormValue("name")
	login := r.FormValue("login")
	password := r.FormValue("password")
	role := r.FormValue("role")
	worksite := r.FormValue("worksite")

	if role == "worker" && worksite == "" {
		return nil, fmt.Errorf("workplace is required for worker")
	}
	if role != "worker" {
		worksite = ""
	}
	user := &User{
		Name:     name,
		Login:    login,
		Password: password,
		Role:     role,
		Worksite: sql.NullString{
			String: worksite,
			Valid:  worksite != "",
		},
	}

	return user, nil
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
