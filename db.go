package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

const (
	userTableQuery = `
		SELECT id, first_name, last_name, login, password, 
		date_created, worksite, role FROM users
	`
)

func InitDB() error {
	connStr := "user=postgres password= dbname=tatraweb sslmode=disable"
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

func userToHTML(user User) string {
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
				<button hx-get="/edit-user/%d" hx-target="#userForm" hx-swap="outerHTML">Upraviť</button>
				<button hx-delete="/delete-user/%d" hx-target="#usersTable" hx-swap="outerHTML">Odstrániť</button>
			</td>
		</tr>
	`, user.ID, user.FirstName, user.LastName, user.Login, user.Role,
		user.DateCreated.Format("2006-01-02 15:04:05"),
		user.Worksite.String, user.ID, user.ID)
}

func fetchAllUsers(w http.ResponseWriter, r *http.Request) {
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
		html += userToHTML(user)
	}
	html += `</tbody>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func addUser(w http.ResponseWriter, r *http.Request) {
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

	fetchAllUsers(w, r)
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
