package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("your-secret-key")

func RegisterUserInDB(user *User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	query := `INSERT INTO users (first_name, last_name, login, password, worksite, role) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	err = db.QueryRow(query, user.FirstName, user.LastName, user.Login, user.Password, user.Worksite, user.Role).Scan(&user.ID)
	if err != nil {
		return err
	}

	return nil
}

func IsLogged(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("jwt_token")
	if err != nil || cookie == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tokenString := cookie.Value

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
		}
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"loggedIn": true,
		"role":     claims.Role,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func Login(w http.ResponseWriter, r *http.Request) {
	login := r.FormValue("login")
	password := r.FormValue("password")

	var user User
	query := `SELECT id, first_name, last_name, password, role, worksite FROM users WHERE login = $1`

	err := db.QueryRow(query, login).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Password, &user.Role, &user.Worksite)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/?error=User%20not%20found", http.StatusSeeOther)
		return
	} else if err != nil {
		http.Redirect(w, r, "/?error=Server%20error", http.StatusSeeOther)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		http.Redirect(w, r, "/?error=Invalid%20password", http.StatusSeeOther)
		return
	}

	expirationTime := time.Now().Add(4 * time.Hour)
	claims := &Claims{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
		Worksite:  user.Worksite.String,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "tatraweb",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Redirect(w, r, "/?error=Error%20generating%20token", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    tokenString,
		HttpOnly: true,
		Expires:  expirationTime,
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	var redirectURL string
	switch user.Role {
	case "manager":
		redirectURL = "/manager"
	case "admin":
		redirectURL = "/admin"
	case "salesman":
		redirectURL = "/salesman"
	case "worker":
		redirectURL = "/worker"
	default:
		redirectURL = "/"
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    "",
		HttpOnly: true,
		Expires:  time.Now().Add(-1 * time.Hour),
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	if r.Header.Get("HX-Request") == "true" {
		tmpl, err := template.ParseFiles("./templates/logout_message.html")
		if err != nil {
			http.Error(w, "Error loading logout message: "+err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, "Error rendering logout message: "+err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func RequireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("jwt_token")
		if err != nil || cookie == nil {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		tokenString := cookie.Value

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
			}
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		if claims.Role != role {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}
