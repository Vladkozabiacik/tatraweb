package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func InitDB() error {
	connStr := "user=postgres password=vladko123 dbname=tatraweb sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	return db.Ping()
}

func fetchAllUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
	SELECT id, first_name, last_name, login, password, date_created, worksite, role FROM users;
	`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching users: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var userHTML string
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Login, &user.Password,
			&user.DateCreated, &user.Worksite, &user.Role)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading user data: %v", err), http.StatusInternalServerError)
			return
		}

		userHTML += fmt.Sprintf(`
            <tbody hx-get="/fetch-all-users" hx-target="#usersTable tbody" hx-swap="outerHTML">
				<tr>
					<td>%d</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>%s</td>
					<td>
						<button hx-get="/edit-user/%d" hx-target="#userForm" hx-swap="outerHTML">Upraviť</button>
						<button hx-delete="/delete-user/%d" hx-target="#usersTable" hx-swap="outerHTML">Odstrániť</button>
					</td>
				</tr>
			</tbody>
		`, user.ID, user.FirstName, user.LastName, user.Login, user.Role, user.DateCreated.Format("2006-01-02 15:04:05"),
			user.Worksite.String, user.ID, user.ID)
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(userHTML))
}

func addUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	firstName := r.FormValue("firstName")
	lastName := r.FormValue("lastName")
	login := r.FormValue("login")
	password := r.FormValue("password")
	position := r.FormValue("position")
	worksite := r.FormValue("worksite")

	if firstName == "" || lastName == "" || login == "" || password == "" || position == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	user := User{
		FirstName:   firstName,
		LastName:    lastName,
		Login:       login,
		Password:    password,
		DateCreated: time.Now(),
		Worksite:    sql.NullString{String: worksite, Valid: worksite != ""},
		Role:        position,
	}

	if position == "worker" {
		if worksite == "" {
			http.Error(w, "Workplace is required for worker", http.StatusBadRequest)
			return
		}
		user.Worksite = sql.NullString{String: worksite, Valid: true}
	}

	err = RegisterUserInDB(&user)
	if err != nil {
		http.Error(w, "Error adding user to database", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
