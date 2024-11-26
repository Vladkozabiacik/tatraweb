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
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "http://localhost:8080/"
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
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

	r.HandleFunc("/login", Login).Methods(http.MethodPost)
	r.HandleFunc("/logout", Logout).Methods(http.MethodPost)
	r.HandleFunc("/status", IsLogged).Methods(http.MethodPost)
}

func (r *Router) registerDashboardRoutes() {

	dashboards := map[string]string{
		RoleAdmin:    "./templates/admin_dashboard.html",
		RoleManager:  "./templates/manager_dashboard.html",
		RoleSalesman: "./templates/salesman_dashboard.html",
		RoleWorker:   "./templates/worker_dashboard.html",
	}

	for role, templatePath := range dashboards {
		handler := &templateHandler{templatePath: templatePath}
		r.Handle("/"+role, CORSMiddleware(RequireRole([]string{role}, handler.ServeHTTP))).Methods(http.MethodGet)
	}
}

func (r *Router) registerAPIRoutes() {

	r.Handle("/add-user", CORSMiddleware(RequireRole([]string{RoleManager}, AddUser))).Methods(http.MethodPost)
	r.Handle("/get-user?{id}", CORSMiddleware(RequireRole([]string{RoleManager}, GetUser))).Methods(http.MethodGet)
	r.Handle("/fetch-all-users", CORSMiddleware(RequireRole([]string{RoleManager, RoleAdmin}, FetchAllUsers))).Methods(http.MethodGet)
	r.Handle("/edit-user", CORSMiddleware(RequireRole([]string{RoleManager}, EditUser))).Methods(http.MethodPut)
	r.Handle("/delete-user", CORSMiddleware(RequireRole([]string{RoleManager}, DeleteUser))).Methods(http.MethodPost)

	r.Handle("/add-product", CORSMiddleware(RequireRole([]string{RoleSalesman, RoleAdmin}, AddProduct))).Methods(http.MethodPost)
	r.Handle("/fetch-all-products", CORSMiddleware(RequireRole([]string{RoleSalesman, RoleAdmin}, FetchAllProducts))).Methods(http.MethodGet)

	r.Handle("/add-customer", CORSMiddleware(RequireRole([]string{RoleSalesman, RoleAdmin}, AddCustomer))).Methods(http.MethodPost)
	r.Handle("/fetch-all-customers", CORSMiddleware(RequireRole([]string{RoleSalesman, RoleAdmin}, FetchAllCustomers))).Methods(http.MethodGet)

	r.Handle("/add-order", CORSMiddleware(RequireRole([]string{RoleSalesman, RoleAdmin}, AddOrder))).Methods(http.MethodPost)
	r.Handle("/edit-order", CORSMiddleware(RequireRole([]string{RoleSalesman, RoleAdmin}, EditOrder))).Methods(http.MethodPut)
	r.Handle("/assign-order-to-workplace", CORSMiddleware(RequireRole([]string{RoleAdmin}, AssignOrderToWorksite))).Methods(http.MethodPost)
	r.Handle("/fetch-production-orders", CORSMiddleware(RequireRole([]string{RoleAdmin, RoleSalesman}, FetchProductionOrders))).Methods(http.MethodGet)

	r.Handle("/fetch-completed-orders", CORSMiddleware(RequireRole([]string{RoleAdmin, RoleSalesman}, FetchCompletedOrders))).Methods(http.MethodGet)

	r.Handle("/fetch-all-orders", CORSMiddleware(RequireRole([]string{RoleAdmin, RoleSalesman}, FetchAllOrders))).Methods(http.MethodGet)
	r.Handle("/fetch-all-orders-for-salesman", CORSMiddleware(RequireRole([]string{RoleAdmin, RoleSalesman}, FetchAllOrdersForSalesman))).Methods(http.MethodGet)

	r.Handle("/fetch-production-orders-for-worksite", CORSMiddleware(RequireRole([]string{RoleWorker}, FetchProductionOrdersForWorksite))).Methods(http.MethodGet)
	r.Handle("/fetch-completed-orders-for-worksite", CORSMiddleware(RequireRole([]string{RoleWorker}, FetchProductionOrdersCompleted))).Methods(http.MethodGet)

	r.Handle("/mark-order-as-completed/{prikazId}", CORSMiddleware(RequireRole([]string{RoleWorker}, MarkOrderAsCompleted))).Methods(http.MethodGet)
}
