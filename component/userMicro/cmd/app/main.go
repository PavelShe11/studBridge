package main

import (
	"os"
	"os/signal"
	"syscall"
	proto "userMicro/internal/api/grpc"
	"userMicro/internal/config"
	"userMicro/internal/repository/db"
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

	pg, err := db.NewPostgresDB(cfg.DB)
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
	if err := db.InitSchema(pg); err != nil {
		l.Fatalf("Failed to initialize database schema: %v", err)
	}

	grpcServer := proto.NewGRPCServer(cfg.Grpc, l)
	go func() {
		if err := grpcServer.Start(); err != nil {
			l.Fatalf("Failed to start grpc server: %v", err)
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
