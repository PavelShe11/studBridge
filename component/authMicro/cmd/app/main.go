package main

import (
	"context"
	"github.com/PavelShe11/studbridge/auth/internal/api/grpcService"
	"github.com/PavelShe11/studbridge/auth/internal/api/rest"
	"github.com/PavelShe11/studbridge/auth/internal/api/rest/handler"
	"github.com/PavelShe11/studbridge/auth/internal/config"
	"github.com/PavelShe11/studbridge/auth/internal/repository"
	"github.com/PavelShe11/studbridge/auth/internal/repository/database"
	"github.com/PavelShe11/studbridge/auth/internal/service"
	"github.com/PavelShe11/studbridge/auth/utlis/interceptor"
	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/PavelShe11/studbridge/common/translator"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/alts"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	l := logger.NewLogger()
	trans := translator.NewTranslator(l)
	cfg, errors := config.NewConfig()
	if len(errors) > 0 {
		for _, err := range errors {
			l.Error(err.Error())
		}
		return
	}

	// grpcService
	var transportOption grpc.DialOption
	if os.Getenv("USE_ALTS") == "true" {
		altsTC := alts.NewClientCreds(alts.DefaultClientOptions())
		transportOption = grpc.WithTransportCredentials(altsTC)
	} else {
		transportOption = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	// Create internal auth interceptor for microservice-to-microservice authentication
	authInterceptor := interceptor.UnaryClientInternalAuthInterceptor(cfg.AccountServiceGrpc.InternalAPIKey, l)

	conn, err := grpc.NewClient(
		cfg.AccountServiceGrpc.Addr,
		transportOption,
		grpc.WithUnaryInterceptor(authInterceptor),
	)
	if err != nil {
		l.Fatalf("Failed to initialize account accountGrpcService: %v", err)
	}

	accountServiceClient := grpcService.NewAccountServiceClient(conn)

	// database
	db, err := database.NewPostgresDB(cfg.DB)
	if err != nil {
		l.Fatalf("Failed to initialize database connection: %v", err)
	}
	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {
			l.Fatalf("Failed to close database connection: %v", err)
		}
	}(db)
	l.Info("Database connection established")

	if err := database.InitSchema(db); err != nil {
		l.Fatalf("Failed to initialize database schema: %v", err)
	}

	registrationSessionRepository := repository.NewRegistrationSessionRepository(db)

	// services
	registrationService := service.NewRegistrationService(*registrationSessionRepository, accountServiceClient, l, &cfg.CodeGenConfig)
	loginService := service.NewLoginService(accountServiceClient)

	// REST server
	router := rest.NewRouter(
		l,
		trans,
		handler.NewRegisterHandler(l, registrationService, trans),
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
		l.Errorf("NameField during server shutdown: %v", err)
	}

	l.Info("Server exited properly")
}
