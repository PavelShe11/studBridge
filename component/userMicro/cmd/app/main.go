package main

import (
	"os"
	"os/signal"
	"syscall"
	"userMicro/internal/api/grpc"
	"userMicro/internal/api/grpc/accountGrpcService"
	"userMicro/internal/config"
	"userMicro/internal/repository"
	"userMicro/internal/repository/database"
	"userMicro/internal/service"
	"userMicro/utlis/logger"

	"github.com/jmoiron/sqlx"
)

func main() {
	l := logger.NewLogger()

	cfg, errors := config.NewConfig()
	if len(errors) > 0 {
		for _, e := range errors {
			l.Errorf(e.Error())
		}
	}

	pg, err := database.NewPostgresDB(cfg.DB)
	if err != nil {
		l.Fatalf("Failed to initialize database connection: %v", err)
	}
	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {
			l.Fatalf("Failed to close database connection: %v", err)
		}
	}(pg)
	l.Info("Database connection established")
	if err := database.InitSchema(pg); err != nil {
		l.Fatalf("Failed to initialize database schema: %v", err)
	}

	accountRepository := repository.NewAccountRepository(pg)

	accountService := service.NewAccountService(accountRepository, l)

	grpcServer := grpc.NewGRPCServer(cfg.Grpc, l)

	// Register service before starting server
	accountGrpcService.Register(grpcServer.Server, *accountService)

	go func() {
		if err := grpcServer.Start(); err != nil {
			l.Fatalf("Failed to start grpcService server: %v", err)
		}
	}()
	defer func() {
		grpcServer.Stop()
		l.Info("Server gracefully stopped")
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	l.Info("Shutting down server...")
}
