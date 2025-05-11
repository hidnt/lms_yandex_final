package orchestrator

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hidnt/lms_yandex_final/pkg/calculation"
	"github.com/hidnt/lms_yandex_final/pkg/database"
	pb "github.com/hidnt/lms_yandex_final/proto"
	"github.com/joho/godotenv"

	"google.golang.org/grpc"
)

var (
	hasSession bool
	userID     int64
)

type RequestSignInOut struct {
	Username string `json:"login,omitempty"`
	Password string `json:"password,omitempty"`
	JWT      string `json:"jwt,omitempty"`
}

type RequestCalc struct {
	Expression string `json:"expression"`
}

type Server struct {
	pb.OrchestratorServer
	db *sql.DB
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) SetResult(ctx context.Context, in *pb.TaskResponse) (*pb.Empty, error) {
	log.Printf("Expression %d, task %d was completed", in.ExpressionId, in.ID)

	if in.Error {
		database.UpdateExpression(ctx, s.db, in.UserID, in.ExpressionId, &database.Expression{Status: "division by zero", Result: 0})
	} else {
		err := database.UpdateActionResult(context.TODO(), s.db, in.UserID, in.ExpressionId, in.ID, in.Res)
		if err != nil {
			return &pb.Empty{}, err
		}
		err = database.UpdateActionStatus(ctx, s.db, in.UserID, in.ExpressionId, in.ID, true, false)
		if err != nil {
			return &pb.Empty{}, err
		}

		actions, err := database.SelectActions(ctx, s.db, in.UserID, in.ExpressionId)
		if err != nil {
			return &pb.Empty{}, err
		}

		complete := 0
		for _, a := range actions {
			if a.Completed {
				complete++
			}
		}

		if complete == len(actions) {
			database.UpdateExpression(ctx, s.db, in.UserID, in.ExpressionId, &database.Expression{Status: "completed",
				Result: actions[len(actions)-1].Result})
		}
	}

	return &pb.Empty{}, nil
}

func (s *Server) GetTask(ctx context.Context, in *pb.Empty) (*pb.TaskRequest, error) {
	exprs, err := database.SelectExpressions(ctx, s.db, userID)
	if err != nil {
		return &pb.TaskRequest{}, nil
	}

	for _, expr := range exprs {
		actions, err := database.SelectActions(ctx, s.db, userID, expr.ID)
		if err != nil {
			return &pb.TaskRequest{}, fmt.Errorf("no available task")
		}
		for _, action := range actions {
			if !action.NowCalculate && !action.Completed && expr.Status == "under consideration" {
				if action.IdDepends[0] != -1 && !actions[action.IdDepends[0]-1].Completed {
					continue
				}
				if action.IdDepends[1] != -1 && !actions[action.IdDepends[1]-1].Completed {
					continue
				}
				task := pb.TaskRequest{ID: action.ID, ExpressionId: action.ExpressionID, UserID: action.UserID, Operation: action.Operation}

				switch task.Operation {
				case "*":
					n := os.Getenv("TIME_MULTIPLICATIONS_MS")
					t, err := strconv.ParseInt(n, 10, 32)
					if err != nil {
						t = 1000
					}
					task.OperationTime = int32(t)
				case "/":
					n := os.Getenv("TIME_DIVISIONS_MS")
					t, err := strconv.ParseInt(n, 10, 32)
					if err != nil {
						t = 1000
					}
					task.OperationTime = int32(t)
				case "+":
					n := os.Getenv("TIME_ADDITION_MS")
					t, err := strconv.ParseInt(n, 10, 32)
					if err != nil {
						t = 1000
					}
					task.OperationTime = int32(t)
				case "-":
					n := os.Getenv("TIME_SUBTRACTION_MS")
					t, err := strconv.ParseInt(n, 10, 32)
					if err != nil {
						t = 1000
					}
					task.OperationTime = int32(t)
				}

				if action.IdDepends[0] == -1 {
					task.Arg1 = action.Arg1
				} else {
					task.Arg1 = actions[action.IdDepends[0]-1].Result
				}

				if action.IdDepends[1] == -1 {
					task.Arg2 = action.Arg2
				} else {
					task.Arg2 = actions[action.IdDepends[1]-1].Result
				}

				database.UpdateActionStatus(ctx, s.db, userID, action.ExpressionID, action.ID, false, true)
				return &task, nil
			}
		}
	}

	return &pb.TaskRequest{}, fmt.Errorf("no available task")
}

func StartGRPC(port string, db *sql.DB) {
	addr := fmt.Sprintf("localhost:%s", port)
	lis, err := net.Listen("tcp", addr)

	if err != nil {
		log.Println("error starting tcp listener: ", err)
		os.Exit(2)
	}

	grpcServer := grpc.NewServer()
	server := NewServer()

	server.db = db

	pb.RegisterOrchestratorServer(grpcServer, server)

	if err := grpcServer.Serve(lis); err != nil {
		log.Println("error serving grpc: ", err)
		os.Exit(3)
	}
}

func StartOrchestrator() {
	err := godotenv.Load()
	if err != nil {
		log.Println("failed to open .env")
		os.Exit(1)
	}
	port := os.Getenv("PORT")

	db, err := sql.Open("sqlite3", "./database/store.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if _, err := db.ExecContext(context.TODO(), "PRAGMA foreign_keys = ON;"); err != nil {
		log.Fatal(err)
	}

	if err := db.PingContext(context.TODO()); err != nil {
		log.Fatal(err)
	}

	if err := database.CreateTables(context.TODO(), db); err != nil {
		log.Fatal(err)
	}

	go StartGRPC(port, db)

	signUpHandler := &SignUpHandler{db: db}
	signInHandler := &SignInHandler{db: db}
	calcHandler := &CalcHandler{db: db}
	expressionsHandler := &ExpressionsHandler{db: db}
	expressionsIdHandler := &ExpressionsIdHandler{db: db}

	http.Handle("/api/v1/register", signUpHandler)
	http.Handle("/api/v1/login", signInHandler)
	http.Handle("/api/v1/calculate", AuthMiddleware(calcHandler.ServeHTTP))
	http.Handle("/api/v1/expressions", AuthMiddleware(expressionsHandler.ServeHTTP))
	http.Handle("/api/v1/expressions/", AuthMiddleware(expressionsIdHandler.ServeHTTP))

	log.Printf("Запущен сервер на :%s", port)
	http.ListenAndServe(":"+port, nil)
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if hasSession {
			next(w, r)
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}
}

type SignUpHandler struct {
	db *sql.DB
}

func (h *SignUpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := new(RequestSignInOut)
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	u := database.User{Username: request.Username, Password: request.Password}

	if _, err := database.InsertUser(context.TODO(), h.db, &u); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type SignInHandler struct {
	db *sql.DB
}

func (h *SignInHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := new(RequestSignInOut)
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	username, password := "", ""

	if request.JWT != "" {
		m, err := DecodeToken(request.JWT)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if _, ok := m["password"]; !ok {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		if _, ok := m["username"]; !ok {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		username = fmt.Sprint(m["username"])
		password = fmt.Sprint(m["password"])
	} else {
		username = request.Username
		password = request.Password
	}
	u, err := database.SelectUser(context.TODO(), h.db, username)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if err = database.ComparePassword(u.Password, password); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	userID = u.ID
	hasSession = true

	jwt, err := CreateToken(username, password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(RequestSignInOut{JWT: jwt})
	w.WriteHeader(http.StatusOK)
}

func CreateToken(username string, password string) (string, error) {
	secret := "a8ff05239d9490298270511a34921076e092ec70b5c562c25f013122a03b185f1eef174ad38d2b20d48e18a315f40cb0c3b29717efb59f597759eaaec722e6757dd5eb412ea204c0859f95004b024bc88656886d1a116064c1f07988105c2b3c83211ecf7dc662007802f45b7a54f25ab9cebb51a29ca9a1e70a7782a78aa07a87d257632537c40e6a9afd46b4a7889aaf6d0eb0bb81cc89d6e74b8442c1f197383017a6b73004cdbba93057586b3e44110706eedef53392e7e7973c2bfb329dff65a5d22045265b80ec08d5a6b7efc95b42310962f6abbc17bcb1d3eb63c6c6d6f89a163ef174985b44718314b659d5f4dac9a1b24555c1f1ee5ef6af53e731"
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = username
	claims["password"] = password

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func DecodeToken(tokenString string) (map[string]interface{}, error) {
	secret := "a8ff05239d9490298270511a34921076e092ec70b5c562c25f013122a03b185f1eef174ad38d2b20d48e18a315f40cb0c3b29717efb59f597759eaaec722e6757dd5eb412ea204c0859f95004b024bc88656886d1a116064c1f07988105c2b3c83211ecf7dc662007802f45b7a54f25ab9cebb51a29ca9a1e70a7782a78aa07a87d257632537c40e6a9afd46b4a7889aaf6d0eb0bb81cc89d6e74b8442c1f197383017a6b73004cdbba93057586b3e44110706eedef53392e7e7973c2bfb329dff65a5d22045265b80ec08d5a6b7efc95b42310962f6abbc17bcb1d3eb63c6c6d6f89a163ef174985b44718314b659d5f4dac9a1b24555c1f1ee5ef6af53e731"
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, err
	}

	return claims, nil
}

type CalcHandler struct {
	db *sql.DB
}

func (h *CalcHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := new(RequestCalc)
	defer r.Body.Close()

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	expr, err := calculation.Calc(request.Expression)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		expr.Status = fmt.Sprint(err)
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	exprID, _ := database.InsertExpression(context.TODO(), h.db, userID, &expr)
	for _, a := range expr.Actions {
		database.InsertActions(context.TODO(), h.db, exprID, userID, &a)
	}
	json.NewEncoder(w).Encode(database.Expression{ID: exprID})
}

type ExpressionsHandler struct {
	db *sql.DB
}

func (h *ExpressionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	exprs, err := database.SelectExpressions(context.TODO(), h.db, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(exprs)
	w.WriteHeader(http.StatusOK)
}

type ExpressionsIdHandler struct {
	db *sql.DB
}

func (h *ExpressionsIdHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	exprs, _ := database.SelectExpressions(context.TODO(), h.db, userID)
	id := r.URL.Path[len("/api/v1/expressions/"):]
	if id == "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(exprs)
		return
	}
	n, err := strconv.ParseInt(id[1:], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(exprs)
		return
	}
	expr, err := database.SelectExpression(context.TODO(), h.db, userID, n)
	if err != nil || len(expr) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(exprs)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(expr)
}
