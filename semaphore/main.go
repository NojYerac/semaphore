package main

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/nojyerac/go-lib/audit"
	"github.com/nojyerac/go-lib/auth"
	libdb "github.com/nojyerac/go-lib/db"
	"github.com/nojyerac/go-lib/health"
	"github.com/nojyerac/go-lib/log"
	"github.com/nojyerac/go-lib/metrics"
	"github.com/nojyerac/go-lib/tracing"
	"github.com/nojyerac/go-lib/transport"
	"github.com/nojyerac/go-lib/transport/grpc"
	libhttp "github.com/nojyerac/go-lib/transport/http"
	"github.com/nojyerac/go-lib/version"

	"github.com/nojyerac/semaphore/config"
	"github.com/nojyerac/semaphore/data"
	"github.com/nojyerac/semaphore/data/db"
	"github.com/nojyerac/semaphore/data/engine"
	"github.com/nojyerac/semaphore/security"
	"github.com/nojyerac/semaphore/transport/http"
	"github.com/nojyerac/semaphore/transport/rpc"
)

//nolint:funlen // main orchestrates process bootstrap and shutdown wiring in one place.
func main() {
	// Initialize configuration
	v := version.GetVersion()

	if err := config.InitAndValidate(); err != nil {
		panic(err)
	}

	// Setup shutdown signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize telemetry
	logger := log.NewLogger(config.LogConfig).WithField("service", v.Name)
	log.SetDefaultCtxLogger(logger)
	ctx = log.WithLogger(ctx, logger)

	tp := tracing.NewTracerProvider(config.TraceConfig)
	tracing.SetGlobal(tp)

	mp, metricHandler, err := metrics.NewMetricProvider()
	if err != nil {
		logger.WithError(err).Panic("failed to create meter provider")
	}
	metrics.SetGlobal(mp)

	checker := health.NewChecker(config.HealthConfig)

	// Initialize data sources
	database := libdb.NewDatabase(
		config.DBConfig,
		libdb.WithLogger(logger),
		libdb.WithHealthChecker(checker),
	)
	if err := database.Open(ctx); err != nil {
		logger.WithError(err).Panic("failed to connect to database")
	}
	auditLogger, err := audit.NewAuditLogger(config.AuditConfig)
	if err != nil {
		logger.WithError(err).Panic("failed to create audit logger")
	}

	source, err := db.NewDataSource(ctx, database, db.WithAuditLogger(auditLogger))
	if err != nil {
		logger.WithError(err).Panic("failed to create data source")
	}

	dataEngine := data.NewDataEngine(source, engine.NewEngine(source))
	validator := auth.NewValidator(config.AuthConfig)

	// Initialize transports
	httpServer := libhttp.NewServer(
		config.HTTPConfig,
		libhttp.WithLogger(logger),
		libhttp.WithHealthChecker(checker),
		libhttp.WithMetricsHandler(metricHandler),
		libhttp.WithAuthMiddleware(validator, security.HTTPPolicyMap()),
	)
	http.RegisterRoutes(dataEngine, httpServer)

	grpc.SetLogger(logger)
	grpcServer := grpc.NewServer(
		rpc.RegisterServices(dataEngine),
		grpc.AuthServerOptions(validator, security.GRPCPolicyMap())...,
	)

	trans, err := transport.NewServer(
		config.TransConfig,
		transport.WithHTTP(httpServer),
		transport.WithGRPC(grpcServer),
	)
	if err != nil {
		logger.WithError(err).Panic("failed to initialize transport")
	}

	// Start everything and wait for shutdown signal
	wg := new(sync.WaitGroup)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := checker.Start(ctx); err != nil && err != context.Canceled {
			logger.WithError(err).Panic("health checker failed")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := trans.Start(ctx); err != nil {
			logger.WithError(err).Panic("transport failed")
		}
	}()

	logger.Info("starting")
	<-sigChan
	cancel()
	logger.Info("stopping")
	wg.Wait()
	logger.Info("stopped")
}
