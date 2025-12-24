package main

import (
	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/PavelShe11/studbridge/common/translator"
	"github.com/PavelShe11/studbridge/user/internal/api/grpc"
	"github.com/PavelShe11/studbridge/user/internal/api/grpc/accountGrpcService"
	"github.com/PavelShe11/studbridge/user/internal/config"
	"github.com/PavelShe11/studbridge/user/internal/repository"
	"github.com/PavelShe11/studbridge/user/internal/repository/database"
	"github.com/PavelShe11/studbridge/user/internal/service"
	"github.com/PavelShe11/studbridge/user/utlis/validation"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmoiron/sqlx"
)

func main() {
	l := logger.NewLogger()
	v := validation.NewValidator()
	trans := translator.NewTranslator(l)

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

	accountService := service.NewAccountService(accountRepository, l, v)

	grpcServer := grpc.NewGRPCServer(cfg.Grpc, l)

	// Register service before starting server
	accountGrpcService.Register(grpcServer.Server, *accountService, trans)

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
