package main

import (
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
)

const (
	PathStatic    = "/static/"
	PathAPI       = "/api"
	PathAuth      = "/auth"
	PathDashboard = "/dashboard"
)

const (
	RoleAdmin    = "admin"
	RoleManager  = "manager"
	RoleSalesman = "salesman"
	RoleWorker   = "worker"
)

type templateHandler struct {
	templatePath string
}

func (th *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(th.templatePath)
	if err != nil {
		http.Error(w, "Error loading template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

type Router struct {
	*mux.Router
}

func NewRouter() *Router {
	return &Router{mux.NewRouter()}
}

func RegisterRoutes() {
	r := NewRouter()

	r.registerStaticRoutes()
	r.registerAuthRoutes()
	r.registerDashboardRoutes()
	r.registerAPIRoutes()

	http.Handle("/", r.Router)
}

func (r *Router) registerStaticRoutes() {
	fileServer := http.FileServer(http.Dir("static"))
	r.PathPrefix(PathStatic).Handler(http.StripPrefix(PathStatic, fileServer))
}

func (r *Router) registerAuthRoutes() {
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		th := &templateHandler{templatePath: "./templates/auth.html"}
		th.ServeHTTP(w, r)
	}).Methods(http.MethodGet)

	// auth := r.PathPrefix(PathAuth).Subrouter()
	r.HandleFunc("/login", Login).Methods(http.MethodPost)
	r.HandleFunc("/logout", Logout).Methods(http.MethodPost)
	r.HandleFunc("/status", IsLogged).Methods(http.MethodPost)
}

func (r *Router) registerDashboardRoutes() {
	// dash := r.PathPrefix(PathDashboard).Subrouter()

	dashboards := map[string]string{
		RoleAdmin:    "./templates/admin_dashboard.html",
		RoleManager:  "./templates/manager_dashboard.html",
		RoleSalesman: "./templates/salesman_dashboard.html",
		RoleWorker:   "./templates/worker_dashboard.html",
	}

	for role, templatePath := range dashboards {
		handler := &templateHandler{templatePath: templatePath}
		r.Handle("/"+role, RequireRole([]string{role}, handler.ServeHTTP)).Methods(http.MethodGet)
	}
}

func (r *Router) registerAPIRoutes() {
	// api := r.PathPrefix(PathAPI).Subrouter()

	// User management endpoints (require manager role)
	// users := api.PathPrefix("/users").Subrouter()
	r.Handle("/add-user", RequireRole([]string{RoleManager}, AddUser)).Methods(http.MethodPost)
	r.Handle("/get-user?{id}", RequireRole([]string{RoleManager}, GetUser)).Methods(http.MethodGet)
	r.Handle("/fetch-all-users", RequireRole([]string{RoleManager, RoleAdmin}, FetchAllUsers)).Methods(http.MethodGet)
	r.Handle("/edit-user", RequireRole([]string{RoleManager}, EditUser)).Methods(http.MethodPut)
	r.Handle("/delete-user", RequireRole([]string{RoleManager}, DeleteUser)).Methods(http.MethodPost)

	r.Handle("/add-product", RequireRole([]string{RoleSalesman}, AddProduct)).Methods(http.MethodPost)
	r.Handle("/fetch-all-products", RequireRole([]string{RoleSalesman}, FetchAllProducts)).Methods(http.MethodGet)

	r.Handle("/add-customer", RequireRole([]string{RoleSalesman}, AddCustomer)).Methods(http.MethodPost)
	r.Handle("/fetch-all-customers", RequireRole([]string{RoleSalesman}, FetchAllCustomers)).Methods(http.MethodGet)
	// r.Handle("/get-customer?{id}", RequireRole([]string{RoleManager}, GetCustomer)).Methods(http.MethodGet)
	// r.Handle("/edit-customer", RequireRole([]string{RoleManager}, EditCustomer)).Methods(http.MethodPut)

	r.Handle("/add-order", RequireRole([]string{RoleSalesman}, AddOrder)).Methods(http.MethodPost)
	r.Handle("/assign-order-to-workplace", RequireRole([]string{RoleAdmin}, AssignOrderToWorksite)).Methods(http.MethodPost)
	r.Handle("/fetch-production-orders", RequireRole([]string{RoleAdmin, RoleManager}, FetchProductionOrders)).Methods(http.MethodGet)

	r.Handle("/fetch-all-orders", RequireRole([]string{RoleAdmin, RoleManager}, FetchAllOrders)).Methods(http.MethodGet)
}
