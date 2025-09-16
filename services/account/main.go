package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL ドライバを使うために必要
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/account/app"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/account/config"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/account/db"
	server "github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/account/grpc"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/account/log"
	pb "github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/account/pb/account"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	// 設定をロード
	conf, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %+v", err)
	}

	// OpenTelemetry を初期化
	ctx := context.Background()
	res, err := newResource(ctx, *conf)
	if err != nil {
		return fmt.Errorf("failed to create resource: %+v", err)
	}
	tp, err := newTraceProvider(ctx, *conf, res)
	if err != nil {
		return fmt.Errorf("failed to create trace provider: %+v", err)
	}
	mp, err := newMeterProvider(ctx, *conf, res)
	if err != nil {
		return fmt.Errorf("failed to create meter provider: %+v", err)
	}

	// データベースに接続
	db, err := db.Connect(conf.DatabaseDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %+v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			panic(err)
		}
	}()

	// アプリケーションを初期化
	app := app.NewApp(db)

	// ロガーを初期化
	logger, err := log.NewLogger(log.Config{Mode: conf.Mode})
	if err != nil {
		return fmt.Errorf("failed to create logger: %+v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	// サーバーを起動
	logger.Info(fmt.Sprintf("starting gRPC server (port = %v)", conf.GRPCPort))
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(conf.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %+v", err)
	}
	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(logger),
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)
	reflection.Register(s)
	svr := server.NewServer(&server.Config{
		App:             app,
		ECDSAPrivateKey: conf.ECDSAPrivateKey,
	})
	pb.RegisterAccountServer(s, svr)
	go stop(s, conf.GracefulStopTimeout, logger)
	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %+v", err)
	}

	if err := mp.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown meter provider: %+v", err)
	}

	if err := tp.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %+v", err)
	}

	return nil
}

func stop(s *grpc.Server, timeout time.Duration, logger *zap.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	logger.Info(fmt.Sprintf("gracefully stopping server (sig = %v)", sig))
	t := time.NewTimer(timeout)
	defer t.Stop()
	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()
	select {
	case <-t.C:
		logger.Warn(fmt.Sprintf("stopping server (not stopped in %s)", timeout.String()))
		s.Stop()
	case <-stopped:
	}
}

func newResource(ctx context.Context, conf config.Config) (*resource.Resource, error) {
	return resource.New(
		ctx,
		resource.WithProcessPID(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceName(conf.ServiceName),
			semconv.ServiceNamespace(conf.ServiceNameSpace),
			semconv.ServiceVersion(conf.ServiceVersion),
			semconv.DeploymentEnvironment(conf.Mode),
		),
	)
}

func newTraceProvider(ctx context.Context, conf config.Config, res *resource.Resource) (*trace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(conf.TraceEndpoint),
		otlptracehttp.WithHeaders(map[string]string{
			"Accept":           "*/*",
			"Mackerel-Api-Key": conf.MackerelAPIKey,
		}),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
	)
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}

func newMeterProvider(ctx context.Context, conf config.Config, res *resource.Resource) (*metric.MeterProvider, error) {
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(conf.MetricEndpoint),
		otlpmetricgrpc.WithHeaders(map[string]string{
			"Mackerel-Api-Key": conf.MackerelAPIKey,
		}),
	)
	if err != nil {
		return nil, err
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter, metric.WithInterval(15*time.Second))),
		metric.WithResource(res),
	)

	otel.SetMeterProvider(mp)

	return mp, nil
}
