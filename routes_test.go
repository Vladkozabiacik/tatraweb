package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginRoute(t *testing.T) {
	req, err := http.NewRequest("POST", "/login", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Login)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("Expected status 303 (See Other), got %v", status)
	}

	if location := rr.Header().Get("Location"); location != "/manager" {
		t.Errorf("Expected redirect to /manager, got %v", location)
	}
}

func TestLogoutRoute(t *testing.T) {
	req, err := http.NewRequest("POST", "/logout", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(Logout)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("Expected status 303 (See Other), got %v", status)
	}

	cookie := rr.Result().Cookies()
	if len(cookie) == 0 {
		t.Errorf("Expected a cookie to be set, got none")
	}
}

func TestRequireRoleMiddleware(t *testing.T) {
	tests := []struct {
		Role          string
		URL           string
		ExpectedCode  int
		ExpectedError string
	}{
		{"manager", "/manager", http.StatusOK, ""},
		{"admin", "/manager", http.StatusForbidden, "Forbidden"},
		{"salesman", "/manager", http.StatusForbidden, "Forbidden"},
		{"worker", "/manager", http.StatusForbidden, "Forbidden"},
	}

	for _, test := range tests {
		req, err := http.NewRequest("GET", test.URL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.AddCookie(&http.Cookie{
			Name:     "jwt_token",
			Value:    "mocked-jwt-token",
			HttpOnly: true,
		})

		rr := httptest.NewRecorder()

		handler := RequireRole("manager", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != test.ExpectedCode {
			t.Errorf("Expected status %v for role %s, got %v", test.ExpectedCode, test.Role, status)
		}

		if status := rr.Body.String(); status != test.ExpectedError {
			t.Errorf("Expected error message %v, got %v", test.ExpectedError, status)
		}
	}
}

func TestAccessDeniedForUnauthorizedRole(t *testing.T) {
	req, err := http.NewRequest("GET", "/manager", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()

	handler := RequireRole("manager", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("Expected status 403, got %v", status)
	}
}

func TestAccessDeniedWithoutJWT(t *testing.T) {
	req, err := http.NewRequest("GET", "/manager", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()

	handler := RequireRole("manager", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %v", status)
	}
}
