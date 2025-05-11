package database

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
)

func TestDB(t *testing.T) {
	db, err := sql.Open("sqlite3", "test.db")
	if err != nil {
		panic(err)
	}

	if _, err := db.ExecContext(context.TODO(), "PRAGMA foreign_keys = ON;"); err != nil {
		log.Fatal(err)
	}

	if err := db.PingContext(context.TODO()); err != nil {
		log.Fatal(err)
	}

	if err := CreateTables(context.TODO(), db); err != nil {
		log.Fatal(err)
	}

	userID, _ := InsertUser(context.TODO(), db, &User{Username: "abcd", Password: "1234"})
	if userID != 1 {
		t.Fatalf("incorrect user id want 1, have %d", userID)
	}

	user, _ := SelectUser(context.TODO(), db, "abcd")
	if err := ComparePassword(user.Password, "1234"); err != nil {
		t.Fatalf("incorrect password")
	}

	InsertExpression(context.TODO(), db, userID, &Expression{Status: "test", Result: 0})
	if exprs, _ := SelectExpressions(context.TODO(), db, userID); len(exprs) != 1 {
		t.Fatalf("incorrect expressions count")
	}

	DeleteUser(context.TODO(), db, userID)
	if users, _ := SelectUsers(context.TODO(), db); len(users) != 0 {
		t.Fatalf("incorrect users count")
	}
	if exprs, _ := SelectExpressions(context.TODO(), db, userID); len(exprs) != 0 {
		t.Fatalf("incorrect expressions count")
	}

	db.Close()
	os.Remove("test.db")
}
