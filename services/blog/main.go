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

	"google.golang.org/grpc/resolver"

	_ "github.com/go-sql-driver/mysql" // MySQL ドライバを使うために必要
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/app"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/config"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/db"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/log"
	pb_account "github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/pb/account"
	pb_renderer "github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/pb/renderer"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/blog/web"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	accountConn, err := grpc.NewClient(conf.AccountAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	rendererConn, err := grpc.NewClient(conf.RendererAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	server, err := web.NewServer(app)
	if err != nil {
		return fmt.Errorf("failed to create server: %+v", err)
	}
	logger.Info(fmt.Sprintf("starting web server (port = %v)", conf.Port))
	go stop(server, conf.GracefulStopTimeout, logger)
	if err := server.Start(":" + strconv.Itoa(conf.Port)); !errors.Is(err, http.ErrServerClosed) {
		return err
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
