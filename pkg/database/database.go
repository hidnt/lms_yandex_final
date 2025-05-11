package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int64        `json:"id"`
	Username string       `json:"username"`
	Password string       `json:"password"`
	Expr     []Expression `json:"expressions"`
}

type Expression struct {
	ID      int64    `json:"id,omitempty"`
	UserID  int64    `json:"-"`
	Status  string   `json:"status,omitempty"`
	Result  float64  `json:"result,omitempty"`
	Actions []Action `json:"-"`
}

type Action struct {
	ID           int64   `json:"id"`
	ExpressionID int64   `json:"exprID"`
	UserID       int64   `json:"userID"`
	Arg1         float64 `json:"arg1"`
	Arg2         float64 `json:"arg2"`
	Result       float64 `json:"result"`
	Operation    string  `json:"operation"`
	IdDepends    []int64 `json:"idDepends"`
	Completed    bool    `json:"completed"`
	NowCalculate bool    `json:"nowCalculate"`
}

func CreateTables(ctx context.Context, db *sql.DB) error {
	const (
		usersTable = `
            CREATE TABLE IF NOT EXISTS users (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                username TEXT UNIQUE NOT NULL,
                password TEXT NOT NULL
            );`
		expressionsTable = `
            CREATE TABLE IF NOT EXISTS expressions (
    			id INTEGER NOT NULL,
    			user_id INTEGER NOT NULL,
    			status TEXT,
    			result REAL,
    			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    			UNIQUE (user_id, id)
			);`
		actionsTable = `
            CREATE TABLE IF NOT EXISTS actions (
    			id INTEGER,
    			expression_id INTEGER NOT NULL,
    			user_id INTEGER,
    			arg1 REAL,
    			arg2 REAL,
    			result REAL,
    			operation TEXT,
    			id_depends TEXT,
    			completed BOOLEAN,
    			now_calculate BOOLEAN,
    			FOREIGN KEY (expression_id, user_id) REFERENCES expressions(id, user_id) ON DELETE CASCADE ON UPDATE CASCADE
			);`
	)

	_, err := db.ExecContext(ctx, usersTable)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, expressionsTable)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, actionsTable)
	if err != nil {
		return err
	}

	return nil
}

func InsertUser(ctx context.Context, db *sql.DB, user *User) (int64, error) {
	query := `
        INSERT INTO users (username, password) 
        VALUES ($1, $2)
    `
	cryptedPassword, err := CryptPassword(user.Password)
	if err != nil {
		return 0, err
	}

	result, err := db.ExecContext(ctx, query, user.Username, cryptedPassword)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func InsertExpression(ctx context.Context, db *sql.DB, userID int64, expr *Expression) (int64, error) {
	query := "SELECT COALESCE(MAX(id), 0) FROM expressions WHERE user_id = $1"
	var maxExprId int64
	err := db.QueryRowContext(ctx, query, userID).Scan(&maxExprId)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	newExprId := maxExprId + 1

	queryInsert := `
        INSERT INTO expressions (id, user_id, status, result) 
        VALUES ($1, $2, $3, $4)
    `
	_, err = db.ExecContext(ctx, queryInsert, newExprId, userID, expr.Status, expr.Result)
	if err != nil {
		return 0, err
	}

	return newExprId, nil
}

func InsertActions(ctx context.Context, db *sql.DB, exprId int64, userID int64, action *Action) (int64, error) {
	query := "SELECT COALESCE(MAX(id), 0) FROM actions WHERE expression_id = $1 and user_id = $2"
	var maxActionId int64
	err := db.QueryRowContext(ctx, query, exprId, userID).Scan(&maxActionId)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	newActionId := maxActionId + 1

	idDependsJson, err := json.Marshal(action.IdDepends)
	if err != nil {
		return 0, err
	}

	queryInsert := `
        INSERT INTO actions (expression_id, user_id, id, arg1, arg2, result, operation, id_depends, completed, now_calculate)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `
	_, err = db.ExecContext(ctx, queryInsert, exprId, userID, newActionId, action.Arg1, action.Arg2, action.Result, action.Operation, string(idDependsJson), action.Completed, action.NowCalculate)

	if err != nil {
		return 0, err
	}

	return newActionId, nil
}

func SelectUsers(ctx context.Context, db *sql.DB) ([]User, error) {
	var users []User
	var q = "SELECT id, username, password FROM users"
	rows, err := db.QueryContext(ctx, q)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		u := User{}
		err := rows.Scan(&u.ID, &u.Username, &u.Password)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

func SelectUser(ctx context.Context, db *sql.DB, username string) (User, error) {
	var u User
	var q = "SELECT id, username, password FROM users WHERE username = $1"
	row, err := db.QueryContext(ctx, q, username)
	if err != nil {
		return User{}, err
	}

	for row.Next() {
		err = row.Scan(&u.ID, &u.Username, &u.Password)
		if err != nil {
			return User{}, err
		}
	}

	return u, nil
}

func SelectExpressions(ctx context.Context, db *sql.DB, userID int64) ([]Expression, error) {
	var exprs []Expression
	var q = "SELECT id, user_id, status, result FROM expressions WHERE user_id = $1"
	rows, err := db.QueryContext(ctx, q, userID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		e := Expression{}
		err := rows.Scan(&e.ID, &e.UserID, &e.Status, &e.Result)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, e)
	}

	return exprs, nil
}

func SelectExpression(ctx context.Context, db *sql.DB, userID int64, exprID int64) ([]Expression, error) {
	var exprs []Expression
	var q = "SELECT id, user_id, status, result FROM expressions WHERE user_id = $1 AND id = $2"
	rows, err := db.QueryContext(ctx, q, userID, exprID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		e := Expression{}
		err := rows.Scan(&e.ID, &e.UserID, &e.Status, &e.Result)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, e)
	}

	return exprs, nil
}

func SelectActions(ctx context.Context, db *sql.DB, userID int64, exprID int64) ([]Action, error) {
	var actions []Action
	var q = "SELECT id, expression_id, user_id, arg1, arg2, result, operation, id_depends, completed, now_calculate FROM actions WHERE user_id = $1 AND expression_id = $2"
	rows, err := db.QueryContext(ctx, q, userID, exprID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		a := Action{}
		s := ""
		err := rows.Scan(&a.ID, &a.ExpressionID, &a.UserID, &a.Arg1, &a.Arg2, &a.Result, &a.Operation, &s, &a.Completed, &a.NowCalculate)

		if err != nil {
			return nil, err
		}

		sReader := strings.NewReader(s)
		err = json.NewDecoder(sReader).Decode(&a.IdDepends)
		if err != nil {
			return nil, err
		}

		actions = append(actions, a)
	}

	return actions, nil
}

func UpdateUser(ctx context.Context, db *sql.DB, userID int64, user *User) error {
	var q = "UPDATE users SET username = $1, password = $2 WHERE id = $3"

	cryptedPassword, err := CryptPassword(user.Password)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, q, user.Username, cryptedPassword, userID)
	if err != nil {
		return err
	}
	return nil
}

func UpdateExpression(ctx context.Context, db *sql.DB, userID, exprID int64, expr *Expression) error {
	var q = "UPDATE expressions SET status = $1, result = $2 WHERE user_id = $3 AND id = $4"
	_, err := db.ExecContext(ctx, q, expr.Status, expr.Result, userID, exprID)
	if err != nil {
		return err
	}
	return nil
}

func UpdateAction(ctx context.Context, db *sql.DB, userID, exprID, actionID int64, action *Action) error {
	var q = "UPDATE actions SET arg1 = $1, arg2 = $2, result = $3, operation = $4, id_depends = $5, completed = $6, now_calculate = $7 WHERE user_id = $8 AND expression_id = $9 AND id = $10"

	idDependsJson, err := json.Marshal(action.IdDepends)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, q, action.Arg1, action.Arg2, action.Result, action.Operation, idDependsJson, action.Completed, action.NowCalculate, userID, exprID, actionID)
	if err != nil {
		return err
	}
	return nil
}

func UpdateActionStatus(ctx context.Context, db *sql.DB, userID, exprID, actionID int64, completed, now_calc bool) error {
	var q = "UPDATE actions SET completed = $1, now_calculate = $2 WHERE user_id = $3 AND expression_id = $4 AND id = $5"
	_, err := db.ExecContext(ctx, q, completed, now_calc, userID, exprID, actionID)
	return err
}

func UpdateActionResult(ctx context.Context, db *sql.DB, userID, exprID, actionID int64, result float64) error {
	var q = "UPDATE actions SET result = $1 WHERE user_id = $2 AND expression_id = $3 AND id = $4"
	_, err := db.ExecContext(ctx, q, result, userID, exprID, actionID)
	return err
}

func DeleteActions(ctx context.Context, db *sql.DB, userID, exprID int64) {
	var q = "DELETE FROM actions WHERE user_id = $1 AND expression_id = $2"
	db.ExecContext(ctx, q, userID, exprID)
}

func DeleteExpressions(ctx context.Context, db *sql.DB, userID int64) {
	var q = "DELETE FROM expressions WHERE user_id = $1"
	db.ExecContext(ctx, q, userID)
}

func DeleteUser(ctx context.Context, db *sql.DB, userID int64) {
	var q = "DELETE FROM users WHERE id = $1"
	db.ExecContext(ctx, q, userID)
}

func CryptPassword(s string) (string, error) {
	saltedBytes := []byte(s)
	hashedBytes, err := bcrypt.GenerateFromPassword(saltedBytes, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	hash := string(hashedBytes[:])
	return hash, nil
}

func ComparePassword(hash string, s string) error {
	incoming := []byte(s)
	existing := []byte(hash)
	return bcrypt.CompareHashAndPassword(existing, incoming)
}
