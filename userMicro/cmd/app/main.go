package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/PavelShe11/studbridge/common/translator"
	"github.com/PavelShe11/studbridge/common/validation"
	"github.com/PavelShe11/studbridge/user/internal/api/grpc"
	"github.com/PavelShe11/studbridge/user/internal/api/grpc/accountGrpcService"
	"github.com/PavelShe11/studbridge/user/internal/config"
	"github.com/PavelShe11/studbridge/user/internal/repository"
	"github.com/PavelShe11/studbridge/user/internal/repository/database"
	"github.com/PavelShe11/studbridge/user/internal/service"
	"github.com/jmoiron/sqlx"
)

type commonModule struct {
	logger     logger.Logger
	translator *translator.Translator
	config     *config.Config
	validator  *validation.Validator
}

type repositoriesModule struct {
	db                *sqlx.DB
	accountRepository *repository.AccountRepository
}

func (r *repositoriesModule) Close(l logger.Logger) {
	if err := r.db.Close(); err != nil {
		l.Errorf("Failed to close database connection: %v", err)
	} else {
		l.Info("Database connection closed")
	}
}

type servicesModule struct {
	accountService *service.AccountService
}

type grpcServerModule struct {
	server *grpc.Server
}

func (g *grpcServerModule) Close(l logger.Logger) {
	g.server.Stop()
	l.Info("gRPC server stopped")
}

type app struct {
	common       *commonModule
	repositories *repositoriesModule
	services     *servicesModule
	grpcServer   *grpcServerModule
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

func newRepositoriesModule(common *commonModule) *repositoriesModule {
	l := common.logger
	db, err := database.NewPostgresDB(common.config.DB)
	if err != nil {
		l.Fatalf("Failed to initialize database connection: %v", err)
	}
	l.Info("Database connection established")

	if err := database.InitSchema(db); err != nil {
		l.Fatalf("Failed to initialize database schema: %v", err)
	}

	return &repositoriesModule{
		db:                db,
		accountRepository: repository.NewAccountRepository(db),
	}
}

func newServicesModule(common *commonModule, repositories *repositoriesModule) *servicesModule {
	return &servicesModule{
		accountService: service.NewAccountService(
			repositories.accountRepository,
			common.logger,
			common.validator,
		),
	}
}

func newGrpcServerModule(common *commonModule, services *servicesModule) *grpcServerModule {
	grpcServer := grpc.NewGRPCServer(common.config.Grpc, common.logger)
	accountGrpcService.Register(grpcServer.Server, *services.accountService, common.translator)

	return &grpcServerModule{
		server: grpcServer,
	}
}

func newApp() *app {
	common := newCommonModule()
	repositories := newRepositoriesModule(common)
	services := newServicesModule(common, repositories)
	grpcServer := newGrpcServerModule(common, services)

	return &app{
		common:       common,
		repositories: repositories,
		services:     services,
		grpcServer:   grpcServer,
	}
}

func (a *app) start() {
	go func() {
		if err := a.grpcServer.server.Start(); err != nil {
			a.common.logger.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()
}

func (a *app) shutdown() {
	a.common.logger.Info("Shutting down server...")

	a.grpcServer.Close(a.common.logger)
	a.repositories.Close(a.common.logger)

	a.common.logger.Info("Server exited properly")
}

func main() {
	app := newApp()
	defer app.shutdown()
	app.start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
