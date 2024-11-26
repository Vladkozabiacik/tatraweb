package main

import (
	"log"
	"net/http"
)

func main() {
	if err := InitDB(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	RegisterRoutes()
	// newUser := User{
	// 	Name:     "Spravca",
	// 	Login:    "spravca",
	// 	Password: "spravca",
	// 	Worksite: sql.NullString{String: "", Valid: false},
	// 	Role:     "manager",
	// }

	// if err := RegisterUserInDB(&newUser); err != nil {
	// 	log.Fatalf("Error registering user: %v", err)
	// } else {
	// 	log.Printf("User registered successfully: %+v\n", newUser)
	// }
	log.Println("Starting server on 127.0.0.1:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
