package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	pb "github.com/hidnt/lms_yandex_final/proto"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func StartAgents() {
	err := godotenv.Load()
	if err != nil {
		log.Println("failed to open .env")
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	n := os.Getenv("COMPUTING_POWER")
	computing_power, err := strconv.Atoi(n)
	if err != nil {
		computing_power = 1
	}

	for range computing_power {
		go func() {
			addr := fmt.Sprintf("localhost:%s", port)
			conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Println("could not connect to grpc server: ", err)
				os.Exit(1)
			}
			defer conn.Close()

			grpcClient := pb.NewOrchestratorClient(conn)

			for {
				task, err := grpcClient.GetTask(context.TODO(), &pb.Empty{})
				if err != nil {
					time.Sleep(time.Second)
					continue
				}

				timer := time.NewTimer(time.Duration(task.OperationTime) * time.Millisecond)
				<-timer.C

				resp := pb.TaskResponse{ID: task.ID, ExpressionId: task.ExpressionId, UserID: task.UserID}

				switch task.Operation {
				case "+":
					resp.Res = task.Arg1 + task.Arg2
					resp.Error = false
				case "-":
					resp.Res = task.Arg1 - task.Arg2
					resp.Error = false
				case "*":
					resp.Res = task.Arg1 * task.Arg2
					resp.Error = false
				case "/":
					if task.Arg2 == 0 {
						resp.Res = 0
						resp.Error = true
					} else {
						resp.Res = task.Arg1 / task.Arg2
						resp.Error = false
					}
				default:
					resp.Res = 0
					resp.Error = true
				}

				grpcClient.SetResult(context.TODO(), &resp)
			}
		}()
	}
	select {}
}
