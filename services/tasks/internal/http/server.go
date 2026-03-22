package http

import (
	"kate/services/tasks/internal/client"
	"kate/services/tasks/internal/http/handler"
	"kate/services/tasks/internal/repository"
	"kate/shared/middleware"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "modernc.org/sqlite"

	"go.uber.org/zap"
)

func StartServer(port string, authGrpcAddr string, logger *zap.Logger) {
	// БД
	repo, err := repository.NewSQLiteTaskRepository("./tasks.db")
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer repo.Close()

	if err := repo.InitDB(); err != nil {
		logger.Fatal("Failed to init database", zap.Error(err))
	}

	// gRPC клиент
	authGrpc, err := client.NewAuthGrpcClient(authGrpcAddr, logger)
	if err != nil {
		logger.Fatal("Failed to connect to Auth gRPC", zap.Error(err))
	}
	defer authGrpc.Close()

	// Хендлер (без taskSvc)
	taskHandler := handler.NewTaskHandler(repo, authGrpc, logger)

	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			taskHandler.GetAllTasks(w, r)
		case http.MethodPost:
			taskHandler.CreateTask(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			taskHandler.GetTaskByID(w, r)
		case http.MethodPatch:
			taskHandler.UpdateTask(w, r)
		case http.MethodDelete:
			taskHandler.DeleteTask(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Поиск
	mux.HandleFunc("/v1/tasks/search", taskHandler.SearchTasks)

	handlerWithMiddleware := middleware.RequestIDMiddleware(
		middleware.MetricsMiddleware(
			middleware.LoggingMiddleware(logger)(mux),
		),
	)

	logger.Info("Tasks HTTP server starting", zap.String("port", port))
	if err := http.ListenAndServe(":"+port, handlerWithMiddleware); err != nil {
		logger.Fatal("HTTP server failed", zap.Error(err))
	}
}
