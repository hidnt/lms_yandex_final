package orchestrator

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hidnt/lms_yandex_final/pkg/database"
)

func initDB(t *testing.T) (*sql.DB, func()) {
	tmpDB, err := os.CreateTemp("", "test.db")
	if err != nil {
		t.Fatalf("Cannot create temp file: %v", err)
	}
	defer func() {
		tmpDB.Close()
		os.Remove(tmpDB.Name())
	}()

	os.Setenv("DATABASE_NAME", tmpDB.Name())

	db, err := sql.Open("sqlite3", tmpDB.Name())
	if err != nil {
		t.Fatalf("Cannot open db: %v", err)
	}

	if err := database.CreateTables(context.Background(), db); err != nil {
		t.Fatalf("Cannot create tables: %v", err)
	}

	return db, func() {
		db.Close()
	}
}

func TestUserFlow(t *testing.T) {
	// Инициализация базы данных
	db, cleanup := initDB(t)
	defer cleanup()

	// Создание тестового сервера
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/register":
			signUpHandler := &SignUpHandler{db: db}
			signUpHandler.ServeHTTP(w, r)
		case "/api/v1/login":
			signInHandler := &SignInHandler{db: db}
			signInHandler.ServeHTTP(w, r)
		case "/api/v1/calculate":
			calcHandler := &CalcHandler{db: db}
			handler := AuthMiddleware(calcHandler.ServeHTTP)
			handler.ServeHTTP(w, r)
		}
	}))
	defer ts.Close()

	// 1. Регистрация пользователя

	username := "testuser"
	password := "testpass"

	loginData := RequestSignInOut{
		Username: username,
		Password: password,
	}

	loginBytes, err := json.Marshal(loginData)
	if err != nil {
		t.Fatalf("Cannot marshal login data: %v", err)
	}

	resp, err := http.Post(ts.URL+"/api/v1/register", "application/json", bytes.NewBuffer(loginBytes))
	if err != nil {
		t.Fatalf("Cannot make POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status code after registration: %v", resp.StatusCode)
	}

	// 2. Попытка добавить выражение без авторизации

	expr := "2+2*2"
	calcData := RequestCalc{
		Expression: expr,
	}

	calcBytes, err := json.Marshal(calcData)
	if err != nil {
		t.Fatalf("Cannot marshal calc data: %v", err)
	}

	resp, err = http.Post(ts.URL+"/api/v1/calculate", "application/json", bytes.NewBuffer(calcBytes))
	if err != nil {
		t.Fatalf("Cannot make POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Unexpected status code after unauthorized attempt: %v", resp.StatusCode)
	}

	// 3. Вход в аккаунт

	loginData.Password = password
	loginBytes, err = json.Marshal(loginData)
	if err != nil {
		t.Fatalf("Cannot marshal login data: %v", err)
	}

	resp, err = http.Post(ts.URL+"/api/v1/login", "application/json", bytes.NewBuffer(loginBytes))
	if err != nil {
		t.Fatalf("Cannot make POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status code after login: %v", resp.StatusCode)
	}

	var result RequestSignInOut
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Cannot decode response: %v", err)
	}
	if result.JWT == "" {
		t.Error("Empty JWT token received")
	}

	// 4. Добавление корректного выражения

	client := &http.Client{}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/calculate", bytes.NewBuffer(calcBytes))
	if err != nil {
		t.Fatalf("Cannot create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+result.JWT)

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Cannot make POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Unexpected status code after correct expression: %v", resp.StatusCode)
	}

	// 5. Добавление некорректного выражения

	invalidExpr := "123-"
	calcData.Expression = invalidExpr
	calcBytes, err = json.Marshal(calcData)
	if err != nil {
		t.Fatalf("Cannot marshal calc data: %v", err)
	}

	req, err = http.NewRequest("POST", ts.URL+"/api/v1/calculate", bytes.NewBuffer(calcBytes))
	if err != nil {
		t.Fatalf("Cannot create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+result.JWT)

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Cannot make POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("Unexpected status code after invalid expression: %v", resp.StatusCode)
	}
}
