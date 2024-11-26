package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func FetchAllCustomers(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("term")

	query := `
        SELECT id, name
        FROM customers
    `

	if searchTerm != "" {
		query += `
            WHERE name LIKE $1
            ORDER BY 
                CASE
                    WHEN name LIKE $2 THEN 1
                    WHEN name LIKE $3 THEN 2
                    WHEN name LIKE $4 THEN 3
                    ELSE 4
                END, name
        `
	}

	stmt, err := db.Prepare(query)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to prepare query", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	var rows *sql.Rows
	if searchTerm != "" {
		rows, err = stmt.Query("%"+searchTerm+"%", searchTerm+"%", "%"+searchTerm, "%"+searchTerm+"%")
	} else {
		rows, err = stmt.Query()
	}
	if err != nil {
		http.Error(w, "Failed to fetch customers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var customer Customer
		err := rows.Scan(&customer.CustomerID, &customer.Name)
		if err != nil {
			http.Error(w, "Failed to scan customer", http.StatusInternalServerError)
			return
		}
		customers = append(customers, customer)
	}
	if err = rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating customers: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"results": customers,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON: %v", err), http.StatusInternalServerError)
	}
}

func CustomerToHTML(customer Customer) string {
	return fmt.Sprintf(`<option value="%d">%s</option>`,
		customer.CustomerID,
		customer.Name,
	)
}

func AddCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var customer Customer
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&customer); err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	if err := RegisterCustomerInDB(&customer); err != nil {
		http.Error(w, "Error adding customer to database", http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(customer)
	if err != nil {
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func RegisterCustomerInDB(customer *Customer) error {

	const query = `
		INSERT INTO customers (name) 
		VALUES ($1) 
		RETURNING id`

	err := db.QueryRow(
		query,
		customer.Name,
	).Scan(&customer.CustomerID)

	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

func GetCustomer(w http.ResponseWriter, r *http.Request) {
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

func validateAndCreateCustomer(r *http.Request) (*Customer, error) {
	name := r.FormValue("customer-name")

	customer := &Customer{
		Name: name,
	}

	return customer, nil
}
