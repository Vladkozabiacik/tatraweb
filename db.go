package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func InitDB() error {
	connStr := "user= password= host= port= dbname= sslmode="
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

	// if err := checkAndCreateTable("users", `
	//     CREATE TABLE users (
	//         id SERIAL PRIMARY KEY,
	//         name VARCHAR(100) NOT NULL,
	//         login VARCHAR(50) UNIQUE NOT NULL,
	//         password VARCHAR(255) NOT NULL,
	//         role VARCHAR(50) NOT NULL,
	//         worksite VARCHAR(50)
	//     )
	// `); err != nil {
	// 	return err
	// }

	// if err := checkAndCreateTable("customers", `
	//     CREATE TABLE customers (
	//         id SERIAL PRIMARY KEY,
	//         name VARCHAR(255) NOT NULL,
	//         created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	//     )
	// `); err != nil {
	// 	return err
	// }
	// if err := insertData("customers", `
	//     INSERT INTO customers (name) VALUES
	//     ('Firma ABC'),
	//     ('Obchodík s.r.o.'),
	//     ('Veľká spoločnosť a.s.'),
	//     ('Malý podnikateľ'),
	//     ('Zákazník XY'),
	//     ('Klient 123'),
	//     ('Nový zákazník'),
	//     ('Stály klient'),
	//     ('Firma XYZ'),
	//     ('Spoločnosť DEF')
	// `); err != nil {
	// 	return err
	// }

	// if err := checkAndCreateTable("worksites", `
	//     CREATE TABLE worksites (
	//         id SERIAL PRIMARY KEY,
	//         name VARCHAR(255) NOT NULL
	//     )
	// `); err != nil {
	// 	return err
	// }
	// if err := insertData("worksites", `
	//     INSERT INTO worksites (name) VALUES
	//     ('Sypke'),
	//     ('Pozivatiny'),
	//     ('Kozmetika'),
	//     ('Sklad')
	// `); err != nil {
	// 	return err
	// }

	// if err := checkAndCreateTable("orders", `
	//     CREATE TABLE orders (
	//         id SERIAL PRIMARY KEY,
	//         customer_id INTEGER REFERENCES customers(id) NOT NULL,
	//         created_by INTEGER REFERENCES users(id) NOT NULL,
	//         created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	//     )
	// `); err != nil {
	// 	return err
	// }

	// if err := checkAndCreateTable("products", `
	//     CREATE TABLE products (
	//         id SERIAL PRIMARY KEY,
	//         kc VARCHAR(50) NOT NULL,
	//         name TEXT NOT NULL
	//     )
	// `); err != nil {
	// 	return err
	// }
	// if err := insertData("products", `
	//     INSERT INTO products (kc, name) VALUES
	//     ('KC-TV-001', 'Televízor LED 55"'),
	//     ('KC-MOB-002', 'Mobilný telefón Smartphone'),
	//     ('KC-LAP-003', 'Notebook 15.6"'),
	//     ('KC-SLU-004', 'Slúchadlá Bluetooth'),
	//     ('KC-TAB-005', 'Tablet 10.1"'),
	//     ('KC-KAM-006', 'Digitálny fotoaparát'),
	//     ('KC-REP-007', 'Reproduktor Bluetooth'),
	//     ('KC-MON-008', 'Monitor 27"'),
	//     ('KC-KLA-009', 'Klávesnica mechanická'),
	//     ('KC-MYS-010', 'Myš bezdrôtová')
	// `); err != nil {
	// 	return err
	// }

	// if err := checkAndCreateTable("order_items", `
	//     CREATE TABLE order_items (
	//         id SERIAL PRIMARY KEY,
	//         order_id INTEGER REFERENCES orders(id) NOT NULL,
	//         product_id INTEGER REFERENCES products(id) NOT NULL,
	//         quantity INTEGER NOT NULL,
	//         delivery_date DATE NOT NULL,
	//         created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	//     )
	// `); err != nil {
	// 	return err
	// }

	// if err := checkAndCreateTable("production_orders", `
	//     CREATE TABLE production_orders (
	//         id SERIAL PRIMARY KEY,
	//         order_item_id INTEGER REFERENCES order_items(id) NOT NULL,
	//         worksite INTEGER REFERENCES worksites(id) NOT NULL,
	//         status VARCHAR(255) NOT NULL,
	//         produced_by INTEGER REFERENCES users(id),
	//         production_date TIMESTAMP WITH TIME ZONE,
	//         created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	//     )
	// `); err != nil {
	// 	return err
	// }

	// if err := checkAndCreateTable("history", `
	//     CREATE TABLE history (
	//         id SERIAL PRIMARY KEY,
	//         action VARCHAR(255) NOT NULL,
	//         performed_by INTEGER REFERENCES users(id) NOT NULL,
	//         details JSONB,
	//         created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	//     )
	// `); err != nil {
	// 	return err
	// }

	return nil
}

func checkAndCreateTable(tableName string, createTableSQL string) error {
	var exists bool
	err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", tableName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("error checking table existence: %v", err)
	}
	if !exists {
		_, err = db.Exec(createTableSQL)
		if err != nil {
			return fmt.Errorf("error creating table %s: %v", tableName, err)
		}
	}
	return nil
}

func insertData(tableName string, insertSQL string) error {
	_, err := db.Exec(insertSQL)
	if err != nil {
		return fmt.Errorf("error inserting data into table %s: %v", tableName, err)
	}
	return nil
}
