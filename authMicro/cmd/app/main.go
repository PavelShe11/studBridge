package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PavelShe11/studbridge/auth/internal/api/rest"
	"github.com/PavelShe11/studbridge/auth/internal/api/rest/handler"
	"github.com/PavelShe11/studbridge/auth/internal/config"
	"github.com/PavelShe11/studbridge/auth/internal/repository"
	"github.com/PavelShe11/studbridge/auth/internal/repository/database"
	"github.com/PavelShe11/studbridge/auth/internal/service"
	"github.com/PavelShe11/studbridge/auth/utlis/interceptor"
	"github.com/PavelShe11/studbridge/authMicro/grpcApi"
	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/PavelShe11/studbridge/common/translator"
	"github.com/PavelShe11/studbridge/common/validation"

	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/alts"
	"google.golang.org/grpc/credentials/insecure"
)

/**
TODO: транзакции
TODO: context
TODO: индексы
*/

type commonModule struct {
	logger     logger.Logger
	translator *translator.Translator
	config     *config.Config
	validator  *validation.Validator
}

type grpcServiceClientsModule struct {
	conn                 *grpc.ClientConn
	accountServiceClient grpcApi.AccountServiceClient
}

func (g *grpcServiceClientsModule) Close(l logger.Logger) {
	if err := g.conn.Close(); err != nil {
		l.Errorf("Failed to close gRPC connection: %v", err)
	} else {
		l.Info("gRPC connection closed")
	}
}

type repositoriesModule struct {
	db                            *sqlx.DB
	registrationSessionRepository *repository.RegistrationSessionRepository
	loginSessionRepository        *repository.LoginSessionRepository
	refreshTokenSessionRepository *repository.RefreshTokenSessionRepository
}

func (r *repositoriesModule) Close(l logger.Logger) {
	if err := r.db.Close(); err != nil {
		l.Errorf("Failed to close database connection: %v", err)
	} else {
		l.Info("Database connection closed")
	}
}

type servicesModule struct {
	registrationService *service.RegistrationService
	loginService        *service.LoginService
	tokenService        *service.TokenService
}

type app struct {
	common       *commonModule
	grpc         *grpcServiceClientsModule
	repositories *repositoriesModule
	services     *servicesModule
	router       *rest.Router
}

func newApp() *app {
	common := newCommonModule()
	grpcClient := newGrpcServiceClientModule(common)
	repositories := newRepositoriesModule(common)
	services := newServicesModule(common, repositories, grpcClient)

	router := rest.NewRouter(
		common.logger,
		common.translator,
		handler.NewRegisterHandler(common.logger, services.registrationService, common.translator),
		handler.NewLoginHandler(common.logger, services.loginService, services.tokenService),
		handler.NewRefreshTokenHandler(common.logger, services.tokenService),
	)

	return &app{
		common:       common,
		grpc:         grpcClient,
		repositories: repositories,
		services:     services,
		router:       router,
	}
}

func (a *app) start() {
	go func() {
		a.common.logger.Infof("Starting REST server on %s", a.common.config.HttpServerAddr)
		if err := a.router.Start(a.common.config.HttpServerAddr); err != nil {
			a.common.logger.Fatalf("Failed to start REST server: %v", err)
		}
	}()
}

func (a *app) shutdown(ctx context.Context) {
	a.common.logger.Info("Shutting down servers...")

	if err := a.router.Shutdown(ctx); err != nil {
		a.common.logger.Errorf("Error during server shutdown: %v", err)
	}

	a.grpc.Close(a.common.logger)
	a.repositories.Close(a.common.logger)

	a.common.logger.Info("Server exited properly")
}

func newCommonModule() *commonModule {
	l := logger.NewLogger()
	trans := translator.NewTranslator(l)
	cfg, errors := config.NewConfig()
	v := validation.NewValidator()
	if len(errors) > 0 {
		for _, err := range errors {
			l.Error(err.Error())
		}
		l.Fatal("Failed to initialize configuration")
	}

	return &commonModule{
		logger:     l,
		translator: trans,
		config:     cfg,
		validator:  v,
	}
}

func newGrpcServiceClientModule(commonModule *commonModule) *grpcServiceClientsModule {
	var transportOption grpc.DialOption
	if os.Getenv("USE_ALTS") == "true" {
		altsTC := alts.NewClientCreds(alts.DefaultClientOptions())
		transportOption = grpc.WithTransportCredentials(altsTC)
	} else {
		transportOption = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	authInterceptor := interceptor.UnaryClientInternalAuthInterceptor(
		commonModule.config.AccountServiceGrpc.InternalAPIKey,
		commonModule.logger,
	)

	conn, err := grpc.NewClient(
		commonModule.config.AccountServiceGrpc.Addr,
		transportOption,
		grpc.WithUnaryInterceptor(authInterceptor),
	)
	if err != nil {
		commonModule.logger.Fatalf("Failed to initialize account accountGrpcService: %v", err)
	}

	accountServiceClient := grpcApi.NewAccountServiceClient(conn)
	return &grpcServiceClientsModule{
		conn:                 conn,
		accountServiceClient: accountServiceClient,
	}
}

func newRepositoriesModule(commonModule *commonModule) *repositoriesModule {
	l := commonModule.logger
	db, err := database.NewPostgresDB(commonModule.config.DB)
	if err != nil {
		l.Fatalf("Failed to initialize database connection: %v", err)
	}
	l.Info("Database connection established")

	if err := database.InitSchema(db); err != nil {
		l.Fatalf("Failed to initialize database schema: %v", err)
	}

	return &repositoriesModule{
		db:                            db,
		registrationSessionRepository: repository.NewRegistrationSessionRepository(db),
		loginSessionRepository:        repository.NewLoginSessionRepository(db),
		refreshTokenSessionRepository: repository.NewRefreshTokenSessionRepository(db),
	}
}

func newServicesModule(
	commonModule *commonModule,
	repositoriesModule *repositoriesModule,
	grpcServiceClientModule *grpcServiceClientsModule,
) *servicesModule {
	l := commonModule.logger
	conf := commonModule.config
	validator := commonModule.validator
	grpcAccountServiceClient := grpcServiceClientModule.accountServiceClient

	return &servicesModule{
		registrationService: service.NewRegistrationService(
			repositoriesModule.registrationSessionRepository,
			grpcAccountServiceClient,
			l,
			conf.CodeGenConfig,
		),
		loginService: service.NewLoginService(
			repositoriesModule.loginSessionRepository,
			grpcAccountServiceClient,
			l,
			conf.CodeGenConfig,
			validator,
		),
		tokenService: service.NewTokenService(
			repositoriesModule.refreshTokenSessionRepository,
			grpcAccountServiceClient,
			commonModule.logger,
			conf.JWT,
		),
	}
}

func main() {
	app := newApp()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		app.shutdown(ctx)
	}()
	app.start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
}
