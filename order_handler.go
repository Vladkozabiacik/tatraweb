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

func AddOrder(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	customerID, err := strconv.Atoi(r.FormValue("customer_id"))
	if err != nil {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}

	products := r.Form["products[]"]
	quantities := r.Form["quantities[]"]
	if len(products) != len(quantities) {
		http.Error(w, "Invalid product data", http.StatusBadRequest)
		return
	}

	shippingDate := r.FormValue("shipping_date")

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

		orderID, err := createOrder(r, customerID, shippingDate)
		if err != nil {
			http.Error(w, "Failed to create order", http.StatusInternalServerError)
			return
		}

		err = createOrderItem(orderID, productID, quantity, shippingDate)
		if err != nil {
			http.Error(w, "Failed to create order item", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
}

func createOrder(r *http.Request, customerID int, shippingDate string) (int, error) {
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

func createOrderItem(orderID, productID, quantity int, deliveryDate string) error {
	_, err := db.Exec(`
        INSERT INTO order_items (order_id, product_id, quantity, delivery_date, created_at) 
        VALUES ($1, $2, $3, $4, NOW())
    `, orderID, productID, quantity, deliveryDate)
	return err
}

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
	GetUserIDFromSession(r)
	defer rows.Close()
	var orders []Order
	for rows.Next() {
		var order Order
		var orderItemsJSON []byte
		err := rows.Scan(&order.ID, &order.CustomerID, &order.CustomerName, &order.CreatedAt, &orderItemsJSON, &order.CreatedBy)
		if err != nil {
			http.Error(w, "Failed to scan order", http.StatusInternalServerError)
			return
		}

		err = json.Unmarshal(orderItemsJSON, &order.OrderItems)
		if err != nil {
			http.Error(w, "Failed to unmarshal order items", http.StatusInternalServerError)
			return
		}
		orders = append(orders, order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func FetchAllOrdersForSalesman(w http.ResponseWriter, r *http.Request) {
	userId := GetUserIDFromSession(r)

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
        AND o.created_by = $1 -- Pridaná podmienka
        GROUP BY o.id, c.name, u.name
        ORDER BY o.created_at DESC
    `, userId)
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
			http.Error(w, "Failed to scan order", http.StatusInternalServerError)
			return
		}

		err = json.Unmarshal(orderItemsJSON, &order.OrderItems)
		if err != nil {
			http.Error(w, "Failed to unmarshal order items", http.StatusInternalServerError)
			return
		}
		orders = append(orders, order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func AssignOrderToWorksite(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		OrderID     int    `json:"order_id"`
		WorkplaceID string `json:"workplace_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Neplatná požiadavka", http.StatusBadRequest)
		return
	}

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
		WHERE po.status != 'dokončená'
	`)
	if err != nil {
		log.Println(err)
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
			http.Error(w, "Chyba pri spracovaní dát z databázy", http.StatusInternalServerError)
			return
		}
		formattedDeliveryDate := deliveryDate.Format("2006-01-02")

		po.OrderItem = OrderItem{
			ProductName:  productName,
			Quantity:     quantity,
			DeliveryDate: formattedDeliveryDate,
		}
		po.Worksite = worksiteName
		productionOrders = append(productionOrders, po)
	}

	jsonData, err := json.Marshal(productionOrders)
	if err != nil {
		http.Error(w, "Chyba pri serializácii dát do JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonData))
}
func FetchCompletedOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT po.id, p.name, oi.quantity, oi.delivery_date, w.name, u.name, po.produced_by, po.status
		FROM production_orders po
		JOIN order_items oi ON po.order_item_id = oi.id
		JOIN products p ON oi.product_id = p.id
		JOIN worksites w ON po.worksite = w.id
		JOIN users u ON po.produced_by = u.id
		WHERE po.status = 'dokončená'
	`)
	if err != nil {
		log.Println(err)
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
		err := rows.Scan(&po.ID, &productName, &quantity, &deliveryDate, &worksiteName, &po.ProducedByName, &po.ProducedBy, &po.Status)
		if err != nil {
			http.Error(w, "Chyba pri spracovaní dát z databázy", http.StatusInternalServerError)
			return
		}
		formattedDeliveryDate := deliveryDate.Format("2006-01-02")

		po.OrderItem = OrderItem{
			ProductName:  productName,
			Quantity:     quantity,
			DeliveryDate: formattedDeliveryDate,
		}
		po.Worksite = worksiteName
		productionOrders = append(productionOrders, po)
	}

	jsonData, err := json.Marshal(productionOrders)
	if err != nil {
		http.Error(w, "Chyba pri serializácii dát do JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonData))
}
func EditOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var order struct {
		OrderID      int    `json:"order_id"`
		ProductName  string `json:"product_name"`
		Quantity     string `json:"quantity"`
		DeliveryDate string `json:"delivery_date"`
		CreatedBy    string `json:"created_by"`
	}

	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	quantity, err := strconv.Atoi(order.Quantity)
	if err != nil {
		http.Error(w, "Quantity musí byť číslo", http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE order_items SET product_id=(SELECT id FROM products WHERE name=$1 LIMIT 1), quantity=$2, delivery_date=$3 WHERE order_id=$4")
	if err != nil {
		http.Error(w, "Chyba pri príprave SQL dotazu", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(order.ProductName, quantity, order.DeliveryDate, order.OrderID)
	if err != nil {

		http.Error(w, "Chyba pri aktualizácii order_items", http.StatusInternalServerError)
		return
	}

	stmt, err = db.Prepare("UPDATE orders SET created_by=(SELECT id from users where name=$1 LIMIT 1) WHERE id=$2")
	if err != nil {

		http.Error(w, "Chyba pri príprave SQL dotazu", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(order.CreatedBy, order.OrderID)
	if err != nil {
		http.Error(w, "Chyba pri aktualizácii orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{})
}

func FetchProductionOrdersForWorksite(w http.ResponseWriter, r *http.Request) {
	userId := GetUserIDFromSession(r)
	var worksiteId string
	err := db.QueryRow("SELECT worksite FROM users WHERE id = $1", userId).Scan(&worksiteId)
	if err != nil {
		http.Error(w, "Chyba pri načítaní pracoviska používateľa", http.StatusInternalServerError)
		return
	}
	var worksiteIdNum int
	switch worksiteId {
	case "sypke":
		worksiteIdNum = 1
	case "pozivatiny":
		worksiteIdNum = 2
	case "kozmetika":
		worksiteIdNum = 3
	case "sklad":
		worksiteIdNum = 4
	default:
		http.Error(w, "Neplatná hodnota pracoviska", http.StatusBadRequest)
		return
	}
	rows, err := db.Query(`
	SELECT po.id, p.name, oi.quantity, oi.delivery_date, w.name, po.produced_by, po.status
	FROM production_orders po
	JOIN order_items oi ON po.order_item_id = oi.id
	JOIN products p ON oi.product_id = p.id
	JOIN worksites w ON po.worksite = w.id
	WHERE w.id = $1
	AND (po.status != 'dokončená' OR po.produced_by IS NULL)  -- Pridaná podmienka
`, worksiteIdNum)
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
			http.Error(w, "Chyba pri spracovaní dát z databázy", http.StatusInternalServerError)
			return
		}
		formattedDeliveryDate := deliveryDate.Format("2006-01-02")

		po.OrderItem = OrderItem{
			ProductName:  productName,
			Quantity:     quantity,
			DeliveryDate: formattedDeliveryDate,
		}
		po.Worksite = worksiteName
		productionOrders = append(productionOrders, po)
	}

	jsonData, err := json.Marshal(productionOrders)
	if err != nil {
		http.Error(w, "Chyba pri serializácii dát do JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonData))
}
func FetchProductionOrdersCompleted(w http.ResponseWriter, r *http.Request) {
	userId := GetUserIDFromSession(r)
	var worksiteId string
	err := db.QueryRow("SELECT worksite FROM users WHERE id = $1", userId).Scan(&worksiteId)
	if err != nil {
		http.Error(w, "Chyba pri načítaní pracoviska používateľa", http.StatusInternalServerError)
		return
	}
	var worksiteIdNum int
	switch worksiteId {
	case "sypke":
		worksiteIdNum = 1
	case "pozivatiny":
		worksiteIdNum = 2
	case "kozmetika":
		worksiteIdNum = 3
	case "sklad":
		worksiteIdNum = 4
	default:
		http.Error(w, "Neplatná hodnota pracoviska", http.StatusBadRequest)
		return
	}
	rows, err := db.Query(`
	SELECT po.id, p.name, oi.quantity, oi.delivery_date, w.name, po.produced_by, po.status, po.production_date
	FROM production_orders po
	JOIN order_items oi ON po.order_item_id = oi.id
	JOIN products p ON oi.product_id = p.id
	JOIN worksites w ON po.worksite = w.id
	WHERE w.id = $1
	AND po.status = 'dokončená'  -- Zmenená podmienka
`, worksiteIdNum)
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
		err := rows.Scan(&po.ID, &productName, &quantity, &deliveryDate, &worksiteName, &po.ProducedBy, &po.Status, &po.ProductionDate)
		if err != nil {
			http.Error(w, "Chyba pri spracovaní dát z databázy", http.StatusInternalServerError)
			return
		}
		formattedDeliveryDate := deliveryDate.Format("2006-01-02")

		po.OrderItem = OrderItem{
			ProductName:  productName,
			Quantity:     quantity,
			DeliveryDate: formattedDeliveryDate,
		}
		po.Worksite = worksiteName
		productionOrders = append(productionOrders, po)
	}

	jsonData, err := json.Marshal(productionOrders)
	if err != nil {
		http.Error(w, "Chyba pri serializácii dát do JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(jsonData))
}
func MarkOrderAsCompleted(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	prikazId, err := strconv.Atoi(vars["prikazId"])
	if err != nil {
		http.Error(w, "Neplatné ID objednávky", http.StatusBadRequest)
		return
	}

	var productionOrder ProductionOrder
	row := db.QueryRow("SELECT id, worksite, status, produced_by, production_date, created_at FROM production_orders WHERE id = $1", prikazId)
	err = row.Scan(&productionOrder.ID, &productionOrder.Worksite, &productionOrder.Status, &productionOrder.ProducedBy, &productionOrder.ProductionDate, &productionOrder.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Objednávka nenájdená", http.StatusNotFound)
		} else {
			http.Error(w, "Chyba pri hľadaní objednávky", http.StatusInternalServerError)
		}
		return
	}

	productionOrder.Status = "dokončená"
	productionOrder.ProductionDate.Time = time.Now()
	productionOrder.ProducedBy.Int64 = int64(GetUserIDFromSession(r))
	productionOrder.ProducedBy.Valid = true

	_, err = db.Exec("UPDATE production_orders SET status = $1, production_date = $2, produced_by = $3 WHERE id = $4", productionOrder.Status, productionOrder.ProductionDate.Time, productionOrder.ProducedBy, productionOrder.ID)
	if err != nil {
		http.Error(w, "Chyba pri aktualizácii objednávky", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
