package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

//AddProduct
//GetProduct
//EditProduct
//DeleteProduct
//FetchAllProducts

func AddProduct(w http.ResponseWriter, r *http.Request) {
	var product Product
	err := json.NewDecoder(r.Body).Decode(&product)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if product.KC == "" || product.Name == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	userID := GetUserIDFromSession(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	const query = `
        INSERT INTO products (kc, name) 
        VALUES ($1, $2) 
        RETURNING id`

	err = db.QueryRow(
		query,
		product.KC,
		product.Name,
	).Scan(&product.ID)

	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to insert product", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": product.ID, "kc": product.KC, "name": product.Name})
}

func GetProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var product Product
	const query = `
        SELECT id, kc, name
        FROM products 
        WHERE id = $1`

	err = db.QueryRow(query, productID).Scan(
		&product.ID,
		&product.KC,
		&product.Name,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to fetch product", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

func FetchAllProducts(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("term")
	// Základný SQL dotaz
	query := `
        SELECT id, kc, name
        FROM products
    `

	// Ak je zadaný vyhľadávací reťazec, pridáme WHERE klauzulu a ORDER BY
	if searchTerm != "" {
		query += `
            WHERE name LIKE $1 OR kc LIKE $2
			ORDER BY 
				CASE
					WHEN name LIKE $3 THEN 1
					WHEN name LIKE $4 THEN 2
					WHEN name LIKE $5 THEN 3
					ELSE 4
				END, name
        `
	}

	// Pripravíme SQL dotaz s wildcards pre vyhľadávanie
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Println(err)

		http.Error(w, "Failed to prepare query", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	var rows *sql.Rows
	if searchTerm != "" {
		// Vykonáme dotaz s vyhľadávacím reťazcom
		rows, err = stmt.Query("%"+searchTerm+"%", "%"+searchTerm+"%", searchTerm+"%", "%"+searchTerm, "%"+searchTerm+"%")
	} else {
		// Vykonáme dotaz bez vyhľadávacieho reťazca
		rows, err = stmt.Query()
	}
	if err != nil {
		http.Error(w, "Failed to fetch products", http.StatusInternalServerError)
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
		)
		if err != nil {
			http.Error(w, "Failed to scan product", http.StatusInternalServerError)
			return
		}
		products = append(products, product)
	}
	if err = rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating products: %v", err), http.StatusInternalServerError)
		return
	}

	// Set response header and encode products as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"results": products,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON: %v", err), http.StatusInternalServerError)
	}
}

func ProductToHTML(product Product) string {
	return fmt.Sprintf(`<option value="%d">%s - %s</option>`,
		product.ID,
		product.KC,
		product.Name,
	)
}

func EditProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var product Product
	err = json.NewDecoder(r.Body).Decode(&product)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if product.KC == "" || product.Name == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Aktualizuj produkt v databáze
	const query = `
        UPDATE products 
        SET kc = $1, name = $2, updated_at = $3
        WHERE id = $4`

	_, err = db.Exec(
		query,
		product.KC,
		product.Name,
		time.Now(),
		productID,
	)

	if err != nil {
		http.Error(w, "Failed to update product", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	const query = `DELETE FROM products WHERE id = $1`
	_, err = db.Exec(query, productID)
	if err != nil {
		http.Error(w, "Failed to delete product", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
