package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// AddOrder pridá novú objednávku
func AddOrder(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Získame ID zákazníka
	customerID, err := strconv.Atoi(r.FormValue("customer_id"))
	if err != nil {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}

	// Získame zoznam produktov a ich množstvo
	products := r.Form["products[]"]
	quantities := r.Form["quantities[]"]
	if len(products) != len(quantities) {
		http.Error(w, "Invalid product data", http.StatusBadRequest)
		return
	}

	// Získame dátum expedície
	shippingDate := r.FormValue("shipping_date")

	// Pre každý produkt vytvoríme samostatnú objednávku
	for i := range products {
		productID, err := strconv.Atoi(products[i])
		if err != nil {
			http.Error(w, "Invalid product ID", http.StatusBadRequest)
			return
		}
		quantity, err := strconv.Atoi(quantities[i])
		if err != nil {
			http.Error(w, "Invalid quantity", http.StatusBadRequest)
			return
		}

		// Vytvoríme novú objednávku v databáze
		orderID, err := createOrder(r, customerID, shippingDate)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to create order", http.StatusInternalServerError)
			return
		}

		// Vytvoríme položku objednávky v databáze
		err = createOrderItem(orderID, productID, quantity, shippingDate)
		if err != nil {
			http.Error(w, "Failed to create order item", http.StatusInternalServerError)
			return
		}
	}

	// Vrátime úspešnú odpoveď
	w.WriteHeader(http.StatusCreated)
}

// createOrder vytvorí novú objednávku v databáze
func createOrder(r *http.Request, customerID int, shippingDate string) (int, error) {
	// Získame ID prihláseného používateľa
	userID := GetUserIDFromSession(r)

	var orderID int
	err := db.QueryRow(`
        INSERT INTO orders (customer_id, created_by, created_at) 
        VALUES ($1, $2, NOW()) RETURNING id
    `, customerID, userID).Scan(&orderID)
	if err != nil {
		return 0, err
	}

	return orderID, nil
}

// createOrderItem vytvorí novú položku objednávky v databáze
func createOrderItem(orderID, productID, quantity int, deliveryDate string) error {
	_, err := db.Exec(`
        INSERT INTO order_items (order_id, product_id, quantity, delivery_date, created_at) 
        VALUES ($1, $2, $3, $4, NOW())
    `, orderID, productID, quantity, deliveryDate)
	return err
}

// FetchAllOrders načíta všetky objednávky z databázy
func FetchAllOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
        SELECT 
            o.id, 
            o.customer_id, 
            c.name, 
            o.created_at,
            json_agg(json_build_object(
                'id', oi.id,
                'product_name', p.name,
                'quantity', oi.quantity,
                'delivery_date', oi.delivery_date
            )) AS order_items
        FROM orders o
        JOIN customers c ON o.customer_id = c.id
        LEFT JOIN order_items oi ON o.id = oi.order_id
        LEFT JOIN products p ON oi.product_id = p.id  -- Spojenie s tabuľkou products
        GROUP BY o.id, c.name
        ORDER BY o.created_at DESC
    `)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var orders []Order
	for rows.Next() {
		var order Order
		var orderItemsJSON []byte
		err := rows.Scan(&order.ID, &order.CustomerID, &order.CustomerName, &order.CreatedAt, &orderItemsJSON)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to scan order", http.StatusInternalServerError)
			return
		}

		// Dekódujeme JSON s položkami objednávky
		err = json.Unmarshal(orderItemsJSON, &order.OrderItems)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to unmarshal order items", http.StatusInternalServerError)
			return
		}

		orders = append(orders, order)
	}

	// Vrátime objednávky ako JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}
