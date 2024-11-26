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

const (
	tokenDuration   = 4 * time.Hour
	tokenCookieName = "jwt_token"
	tokenIssuer     = "tatraweb"
)

var (
	jwtKey          = []byte("your-secret-key")
	roleRedirectMap = map[string]string{
		"manager":  "/manager",
		"admin":    "/admin",
		"salesman": "/salesman",
		"worker":   "/worker",
	}
)

type AuthError struct {
	Message string
	Code    int
}

func NewAuthError(message string, code int) *AuthError {
	return &AuthError{
		Message: message,
		Code:    code,
	}
}

func RegisterUserInDB(user *User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	const query = `
		INSERT INTO users (name, login, password, worksite, role) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id`

	err = db.QueryRow(
		query,
		user.Name,
		user.Login,
		string(hashedPassword),
		user.Worksite,
		user.Role,
	).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

func IsLogged(w http.ResponseWriter, r *http.Request) {
	claims, err := validateToken(r)
	if err != nil {
		http.Error(w, err.Message, err.Code)
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
	user, err := authenticateUser(r.FormValue("login"), r.FormValue("password"))
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/?error=%s", err.Message), http.StatusSeeOther)
		return
	}
	token, err := generateToken(user)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/?error=%s", err.Message), http.StatusSeeOther)
		return
	}

	setAuthCookie(w, token)
	redirectToUserDashboard(w, r, user.Role)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	clearAuthCookie(w)

	if r.Header.Get("HX-Request") == "true" {
		if err := renderLogoutMessage(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func RequireRole(roles []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := validateToken(r)
		if err != nil {
			http.Error(w, err.Message, err.Code)
			return
		}

		roleAllowed := false
		for _, role := range roles {
			if claims.Role == role {
				roleAllowed = true
				break
			}
		}

		if !roleAllowed {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

func authenticateUser(login, password string) (*User, *AuthError) {

	var user User
	const query = `
	SELECT id, name, password, role, worksite 
	FROM users 
	WHERE login = $1`

	err := db.QueryRow(query, login).Scan(
		&user.ID,
		&user.Name,
		&user.Password,
		&user.Role,
		&user.Worksite,
	)
	if err == sql.ErrNoRows {
		return nil, NewAuthError("User not found", http.StatusUnauthorized)
	} else if err != nil {
		return nil, NewAuthError("Server error", http.StatusInternalServerError)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, NewAuthError("Invalid password", http.StatusUnauthorized)
	}

	return &user, nil
}

func generateToken(user *User) (string, *AuthError) {
	claims := &Claims{
		ID:       user.ID,
		Name:     user.Name,
		Role:     user.Role,
		Worksite: user.Worksite.String,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenDuration)),
			Issuer:    tokenIssuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", NewAuthError("Error generating token", http.StatusInternalServerError)
	}
	return tokenString, nil
}

func validateToken(r *http.Request) (*Claims, *AuthError) {
	cookie, err := r.Cookie(tokenCookieName)
	if err != nil || cookie == nil {
		return nil, NewAuthError("Missing token", http.StatusUnauthorized)
	}

	token, err := jwt.ParseWithClaims(cookie.Value, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
		}
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		return nil, NewAuthError("Invalid or expired token", http.StatusUnauthorized)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, NewAuthError("Invalid token claims", http.StatusUnauthorized)
	}

	return claims, nil
}

func setAuthCookie(w http.ResponseWriter, tokenString string) {
	http.SetCookie(w, &http.Cookie{
		Name:     tokenCookieName,
		Value:    tokenString,
		HttpOnly: true,
		Expires:  time.Now().Add(tokenDuration),
		Path:     "/",
		// Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     tokenCookieName,
		Value:    "",
		HttpOnly: true,
		Expires:  time.Now().Add(-1 * time.Hour),
		Path:     "/",
		// Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func renderLogoutMessage(w http.ResponseWriter) error {
	tmpl, err := template.ParseFiles("./templates/logout_message.html")
	if err != nil {
		return fmt.Errorf("error loading logout message template: %w", err)
	}

	if err := tmpl.Execute(w, nil); err != nil {
		return fmt.Errorf("error rendering logout message: %w", err)
	}

	return nil
}

func redirectToUserDashboard(w http.ResponseWriter, r *http.Request, role string) {
	redirectURL, exists := roleRedirectMap[role]
	if !exists {
		redirectURL = "/"
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
