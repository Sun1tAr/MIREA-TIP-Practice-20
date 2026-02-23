package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sun1tar/MIREA-TIP-Practice-20/tech-ip-sem2/shared/logger"
	"github.com/sun1tar/MIREA-TIP-Practice-20/tech-ip-sem2/shared/middleware" // requestid и логирование
	"github.com/sun1tar/MIREA-TIP-Practice-20/tech-ip-sem2/tasks/internal/client/authclient"
	handlers "github.com/sun1tar/MIREA-TIP-Practice-20/tech-ip-sem2/tasks/internal/http"
	metricsMiddleware "github.com/sun1tar/MIREA-TIP-Practice-20/tech-ip-sem2/tasks/internal/middleware" // metrics
	"github.com/sun1tar/MIREA-TIP-Practice-20/tech-ip-sem2/tasks/internal/service"
)

func main() {
	// Инициализация структурированного логгера
	logrusLogger := logger.Init("tasks")

	tasksPort := os.Getenv("TASKS_PORT")
	if tasksPort == "" {
		tasksPort = "8082"
	}
	authGrpcAddr := os.Getenv("AUTH_GRPC_ADDR")
	if authGrpcAddr == "" {
		authGrpcAddr = "localhost:50051"
	}

	authClient, err := authclient.NewClient(authGrpcAddr, 2*time.Second, logrusLogger)
	if err != nil {
		logrusLogger.WithError(err).Fatal("Failed to create auth client")
	}
	defer authClient.Close()

	taskService := service.NewTaskService()
	taskHandler := handlers.NewTaskHandler(taskService, authClient, logrusLogger)

	mux := http.NewServeMux()

	// API эндпоинты
	mux.HandleFunc("POST /v1/tasks", taskHandler.CreateTask)
	mux.HandleFunc("GET /v1/tasks", taskHandler.ListTasks)
	mux.HandleFunc("GET /v1/tasks/{id}", taskHandler.GetTask)
	mux.HandleFunc("PATCH /v1/tasks/{id}", taskHandler.UpdateTask)
	mux.HandleFunc("DELETE /v1/tasks/{id}", taskHandler.DeleteTask)

	// Метрики Prometheus (без авторизации)
	mux.Handle("GET /metrics", metricsMiddleware.MetricsHandler())

	// Цепочка middleware
	handler := middleware.RequestIDMiddleware(middleware.LoggingMiddleware(mux)) // сначала request-id
	handler = metricsMiddleware.MetricsMiddleware(handler)                       // потом метрики
	handler = middleware.LoggingMiddleware(handler)                              // потом логирование

	addr := fmt.Sprintf(":%s", tasksPort)
	logrusLogger.WithField("port", tasksPort).Info("Tasks service starting")
	if err := http.ListenAndServe(addr, handler); err != nil {
		logrusLogger.WithError(err).Fatal("server failed")
	}
}
