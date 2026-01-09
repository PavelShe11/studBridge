package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	repository2 "github.com/PavelShe11/studbridge/authMicro/internal/infrastructure/adapter/repository"
	"github.com/PavelShe11/studbridge/authMicro/internal/infrastructure/adapter/repository/database"
	"github.com/PavelShe11/studbridge/authMicro/internal/usecase"
	trmsql "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	trmcontext "github.com/avito-tech/go-transaction-manager/trm/v2/context"

	"github.com/PavelShe11/studbridge/authMicro/grpcApi"
	"github.com/PavelShe11/studbridge/authMicro/internal/api/rest"
	"github.com/PavelShe11/studbridge/authMicro/internal/api/rest/handler"
	"github.com/PavelShe11/studbridge/authMicro/internal/config"
	grpcAdapter "github.com/PavelShe11/studbridge/authMicro/internal/infrastructure/adapter/grpc"
	"github.com/PavelShe11/studbridge/authMicro/internal/port"
	"github.com/PavelShe11/studbridge/authMicro/internal/service"
	"github.com/PavelShe11/studbridge/authMicro/utlis/interceptor"
	jwtAdapter "github.com/PavelShe11/studbridge/authMicro/utlis/tokenGenerator"
	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/PavelShe11/studbridge/common/translator"
	"github.com/PavelShe11/studbridge/common/validation"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/alts"
	"google.golang.org/grpc/credentials/insecure"
)

/**
TODO: Подумать над заведением отдельным моделей для хранилища.
TODO: Добавить ошибку 403 при обновлении refreshToken с незарегистрированным refreshToken
TODO: Добавить тесты
*/

type commonModule struct {
	logger     logger.Logger
	translator *translator.Translator
	config     *config.Config
	validator  *validation.Validator
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

type grpcServiceClientsModule struct {
	conn                 *grpc.ClientConn
	accountServiceClient grpcApi.AccountServiceClient
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

func (g *grpcServiceClientsModule) Close(l logger.Logger) {
	if err := g.conn.Close(); err != nil {
		l.Errorf("Failed to close gRPC connection: %v", err)
	} else {
		l.Info("gRPC connection closed")
	}
}

type repositoriesModule struct {
	db                            *sqlx.DB
	registrationSessionRepository port.RegistrationSessionRepository
	loginSessionRepository        port.LoginSessionRepository
	refreshTokenSessionRepository port.RefreshTokenSessionRepository
	trManager                     *manager.Manager
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

	trManager := manager.Must(
		trmsql.NewDefaultFactory(db),
		manager.WithCtxManager(trmcontext.DefaultManager),
	)

	return &repositoriesModule{
		db:                            db,
		registrationSessionRepository: repository2.NewRegistrationSessionRepository(db, trmsql.DefaultCtxGetter),
		loginSessionRepository:        repository2.NewLoginSessionRepository(db, trmsql.DefaultCtxGetter),
		refreshTokenSessionRepository: repository2.NewRefreshTokenSessionRepository(db, trmsql.DefaultCtxGetter),
		trManager:                     trManager,
	}
}

func (r *repositoriesModule) Close(l logger.Logger) {
	if err := r.db.Close(); err != nil {
		l.Errorf("Failed to close database connection: %v", err)
	} else {
		l.Info("Database connection closed")
	}
}

type infrastructureModule struct {
	accountProvider port.AccountProvider
	tokenGenerator  jwtAdapter.TokenGenerator
}

func newInfrastructureModule(
	commonModule *commonModule,
	grpcServiceClientModule *grpcServiceClientsModule,
) *infrastructureModule {
	accountProvider := grpcAdapter.NewAccountGrpcAdapter(
		grpcServiceClientModule.accountServiceClient,
		commonModule.logger,
	)

	tokenGenerator := jwtAdapter.NewJwtTokenAdapter(commonModule.config.JWT)

	return &infrastructureModule{
		accountProvider: accountProvider,
		tokenGenerator:  tokenGenerator,
	}
}

type servicesModule struct {
	registrationService *service.RegistrationService
	loginService        *service.LoginService
	tokenService        *service.TokenService
}

func newServicesModule(
	commonModule *commonModule,
	repositoriesModule *repositoriesModule,
	infrastructureModule *infrastructureModule,
) *servicesModule {
	l := commonModule.logger
	conf := commonModule.config
	validator := commonModule.validator
	accountProvider := infrastructureModule.accountProvider

	return &servicesModule{
		registrationService: service.NewRegistrationService(
			repositoriesModule.registrationSessionRepository,
			accountProvider,
			l,
			conf.CodeGenConfig,
		),
		loginService: service.NewLoginService(
			repositoriesModule.loginSessionRepository,
			accountProvider,
			l,
			conf.CodeGenConfig,
			validator,
		),
		tokenService: service.NewTokenService(
			repositoriesModule.refreshTokenSessionRepository,
			infrastructureModule.accountProvider,
			infrastructureModule.tokenGenerator,
			commonModule.logger,
			conf.JWT.AccessTokenExpiration,
			conf.JWT.RefreshTokenExpiration,
		),
	}
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
	infrastructure := newInfrastructureModule(common, grpcClient)
	services := newServicesModule(common, repositories, infrastructure)

	authenticateUserUsecase := usecase.NewAuthenticateUser(
		services.loginService,
		services.tokenService,
		repositories.trManager,
	)

	router := rest.NewRouter(
		common.logger,
		common.translator,
		handler.NewRegisterHandler(common.logger, services.registrationService, common.translator),
		handler.NewLoginHandler(common.logger, services.loginService, authenticateUserUsecase),
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
