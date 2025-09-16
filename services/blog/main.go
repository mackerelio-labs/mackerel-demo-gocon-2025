package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL ドライバを使うために必要
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/app"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/config"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/db"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/log"
	pb_account "github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/pb/account"
	pb_renderer "github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/pb/renderer"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/web"
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
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
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

	// アカウントサービスに接続
	resolver.SetDefaultScheme("passthrough")
	accountConn, err := grpc.NewClient(conf.AccountAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to account service: %+v", err)
	}
	defer func() {
		if err := accountConn.Close(); err != nil {
			panic(err)
		}
	}()
	accountCli := pb_account.NewAccountClient(accountConn)

	// レンダラ (記法変換) サービスに接続
	resolver.SetDefaultScheme("passthrough")
	rendererConn, err := grpc.NewClient(conf.RendererAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to renderer service: %+v", err)
	}
	defer func() {
		if err := rendererConn.Close(); err != nil {
			panic(err)
		}
	}()
	rendererCli := pb_renderer.NewRendererClient(rendererConn)

	// アプリケーションを初期化
	app := app.NewApp(db, accountCli, conf.AccountECDSAPublicKey, rendererCli)

	// ロガーを初期化
	logger, err := log.NewLogger(log.Config{Mode: conf.Mode})
	if err != nil {
		return fmt.Errorf("failed to create logger: %+v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	// サーバーを起動
	// TODO: logger をサーバーでも使う
	server, err := web.NewServer(app, conf.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to create server: %+v", err)
	}
	logger.Info(fmt.Sprintf("starting web server (port = %v)", conf.Port))
	go stop(server, conf.GracefulStopTimeout, logger)
	if err := server.Start(":" + strconv.Itoa(conf.Port)); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	if err := mp.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown meter provider: %+v", err)
	}

	if err := tp.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %+v", err)
	}

	return nil
}

func stop(server *web.Server, timeout time.Duration, logger *zap.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	logger.Info(fmt.Sprintf("gracefully stopping server (sig = %v)", sig))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Warn(fmt.Sprintf("failed to stop server: %+v", err))
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
