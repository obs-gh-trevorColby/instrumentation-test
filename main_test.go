package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMain(m *testing.M) {
	// Initialize OpenTelemetry for tests
	shutdown := setupInstrumentation("instrumentation-test")
	defer shutdown()
	
	// Initialize metrics
	initMetrics()
	
	// Run tests
	m.Run()
}

func TestHomeHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(homeHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Message != "Welcome to the instrumentation test API" {
		t.Errorf("handler returned unexpected message: got %v want %v", response.Message, "Welcome to the instrumentation test API")
	}
}

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Message != "Service is healthy" {
		t.Errorf("handler returned unexpected message: got %v want %v", response.Message, "Service is healthy")
	}
}

func TestUsersHandlerGet(t *testing.T) {
	req, err := http.NewRequest("GET", "/users", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(usersHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Message != "Users retrieved successfully" {
		t.Errorf("handler returned unexpected message: got %v want %v", response.Message, "Users retrieved successfully")
	}
}

func TestUsersHandlerPost(t *testing.T) {
	newUser := User{
		Name:  "Test User",
		Email: "test@example.com",
	}
	
	userJSON, err := json.Marshal(newUser)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/users", bytes.NewBuffer(userJSON))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(usersHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Message != "User created successfully" {
		t.Errorf("handler returned unexpected message: got %v want %v", response.Message, "User created successfully")
	}
}

func TestUserHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/users/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(userHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Message != "User found" {
		t.Errorf("handler returned unexpected message: got %v want %v", response.Message, "User found")
	}
}

func TestUserHandlerNotFound(t *testing.T) {
	req, err := http.NewRequest("GET", "/users/999", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(userHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Message != "User not found" {
		t.Errorf("handler returned unexpected message: got %v want %v", response.Message, "User not found")
	}
}

func TestSlowHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/slow", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(slowHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Message != "Slow operation completed" {
		t.Errorf("handler returned unexpected message: got %v want %v", response.Message, "Slow operation completed")
	}
}
