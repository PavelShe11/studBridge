package main

import (
	"authMicro/internal/api/grpc"
	"authMicro/internal/api/rest"
	"authMicro/internal/api/rest/handler"
	"authMicro/internal/config"
	"authMicro/internal/repository/db"
	"authMicro/internal/service"
	"authMicro/utlis/logger"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	g "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/alts"
)

func main() {
	l := logger.NewLogger()
	cfg, errors := config.NewConfig()
	if len(errors) > 0 {
		for _, err := range errors {
			l.Error(err.Error())
		}
		return
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

	altsTC := alts.NewClientCreds(alts.DefaultClientOptions())
	conn, err := g.NewClient(cfg.AccountServiceGrpcAddr, g.WithTransportCredentials(altsTC))
	if err != nil {
		l.Fatalf("Failed to initialize account service: %v", err)
	}

	accountServiceClient := grpc.NewAccountServiceClient(conn)

	loginService := service.NewLoginService(accountServiceClient)

	router := rest.NewRouter(
		handler.NewRegisterHandler(l),
		handler.NewLoginHandler(l, loginService),
		handler.NewRefreshTokenHandler(l),
	)

	go func() {
		l.Infof("Starting REST server on %s", cfg.HttpServerAddr)
		if err := router.Start(cfg.HttpServerAddr); err != nil {
			l.Fatalf("Failed to start REST server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	l.Info("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := router.Shutdown(ctx); err != nil {
		l.Errorf("Error during server shutdown: %v", err)
	}

	l.Info("Server exited properly")
}
