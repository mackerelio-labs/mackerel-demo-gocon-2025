package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/renderer/config"
	server "github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/renderer/grpc"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/renderer/log"
	pb "github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/renderer/pb/renderer"
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
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(logger),
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)
	reflection.Register(s)
	svr := server.NewServer()
	pb.RegisterRendererServer(s, svr)
	go stop(s, conf.GracefulStopTimeout, logger)
	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %+v", err)
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
