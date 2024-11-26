package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
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
			)) AS order_items,
			u.name AS created_by_name 
		FROM orders o
		JOIN customers c ON o.customer_id = c.id
		LEFT JOIN order_items oi ON o.id = oi.order_id
		LEFT JOIN products p ON oi.product_id = p.id
		LEFT JOIN users u ON o.created_by = u.id
		WHERE o.id NOT IN (SELECT oi2.order_id FROM order_items oi2 WHERE oi2.id IN (SELECT order_item_id FROM production_orders))
		GROUP BY o.id, c.name, u.name
		ORDER BY o.created_at DESC
    `)
	if err != nil {
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var orders []Order
	for rows.Next() {
		var order Order
		var orderItemsJSON []byte
		err := rows.Scan(&order.ID, &order.CustomerID, &order.CustomerName, &order.CreatedAt, &orderItemsJSON, &order.CreatedBy)
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

func AssignOrderToWorksite(w http.ResponseWriter, r *http.Request) {
	// 1. Získanie dát z požiadavky
	var requestData struct {
		OrderID     int    `json:"order_id"`
		WorkplaceID string `json:"workplace_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Neplatná požiadavka", http.StatusBadRequest)
		return
	}

	// 2. Overenie údajov
	var orderExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM orders WHERE id = $1)", requestData.OrderID).Scan(&orderExists)
	if err != nil {
		http.Error(w, "Chyba pri overovaní objednávky", http.StatusInternalServerError)
		return
	}
	if !orderExists {
		http.Error(w, "Objednávka neexistuje", http.StatusBadRequest)
		return
	}

	// 3. Aktualizácia objednávky - v tomto prípade nie je potrebné aktualizovať objednávku
	//    samotnú, ale vytvoríme nový záznam v tabuľke production_orders

	// 4. Vytvorenie výrobného príkazu
	var orderItemID int
	err = db.QueryRow("SELECT id FROM order_items WHERE order_id = $1", requestData.OrderID).Scan(&orderItemID)
	if err != nil {
		http.Error(w, "Chyba pri získavaní order_item_id", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO production_orders (order_item_id, worksite, status, created_at) VALUES ($1, $2, $3, $4)",
		orderItemID, requestData.WorkplaceID, "priradené", time.Now())
	if err != nil {
		http.Error(w, "Chyba pri vytváraní výrobného príkazu", http.StatusInternalServerError)
		return
	}

	// 5. Odpoveď
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Objednávka bola úspešne priradená"})
}

func FetchProductionOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
			SELECT po.id, p.name, oi.quantity, oi.delivery_date, w.name, po.produced_by, po.status
			FROM production_orders po
			JOIN order_items oi ON po.order_item_id = oi.id
			JOIN products p ON oi.product_id = p.id
			JOIN worksites w ON po.worksite = w.id
	`)
	if err != nil {
		http.Error(w, "Chyba pri načítaní dát z databázy", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var productionOrders []ProductionOrder
	for rows.Next() {
		var po ProductionOrder
		var productName string
		var quantity int
		var deliveryDate time.Time
		var worksiteName string
		err := rows.Scan(&po.ID, &productName, &quantity, &deliveryDate, &worksiteName, &po.ProducedBy, &po.Status)
		if err != nil {
			log.Println(err)
			http.Error(w, "Chyba pri spracovaní dát z databázy", http.StatusInternalServerError)
			return
		}
		formattedDeliveryDate := deliveryDate.Format("2006-01-02")

		po.OrderItem = OrderItem{
			ProductName:  productName,
			Quantity:     quantity,
			DeliveryDate: formattedDeliveryDate, // Použitie formátovaného dátumu
		}
		po.Worksite = worksiteName
		productionOrders = append(productionOrders, po)
	}

	// Serializácia dát do JSON
	jsonData, err := json.Marshal(productionOrders)
	if err != nil {
		http.Error(w, "Chyba pri serializácii dát do JSON", http.StatusInternalServerError)
		return
	}

	// Odosielanie JSON odpovede
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonData))
}
