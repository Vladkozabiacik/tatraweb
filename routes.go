package main

import (
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
)

func RegisterRoutes() {
	r := mux.NewRouter()
	// todo.. sort this mess ;(
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r) // impl not found page + redirect to main auth
			return
		}
		tmpl, err := template.ParseFiles("./templates/auth.html")
		if err != nil {
			http.Error(w, "Error loading template: "+err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}).Methods("GET")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.HandleFunc("/login", Login).Methods("POST")
	r.HandleFunc("/logout", Logout).Methods("POST")
	r.HandleFunc("/isLogged", IsLogged).Methods("POST")
	r.HandleFunc("/manager", RequireRole("manager", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("./templates/manager_dashboard.html")
		if err != nil {
			http.Error(w, "Error loading template: "+err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})).Methods("GET")

	r.HandleFunc("/admin", RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("./templates/admin_dashboard.html")
		if err != nil {
			http.Error(w, "Error loading template: "+err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})).Methods("GET")

	r.HandleFunc("/salesman", RequireRole("salesman", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("./templates/salesman_dashboard.html")
		if err != nil {
			http.Error(w, "Error loading template: "+err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})).Methods("GET")

	r.HandleFunc("/worker", RequireRole("worker", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("./templates/worker_dashboard.html")
		if err != nil {
			http.Error(w, "Error loading template: "+err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})).Methods("GET")

	r.HandleFunc("/fetch-all-users", RequireRole("manager", fetchAllUsers)).Methods("GET")
	r.HandleFunc("/add-user", RequireRole("manager", addUser)).Methods("POST")

	http.Handle("/", r)
}
