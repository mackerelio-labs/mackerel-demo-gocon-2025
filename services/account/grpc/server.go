package grpc

import (
	"crypto/ecdsa"

	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/account/app"
	pb "github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/account/pb/account"
)

// Config はサーバーの設定
type Config struct {
	App             *app.App
	ECDSAPrivateKey *ecdsa.PrivateKey
}

// Server は pb.AccountServer に対する実装
type Server struct {
	pb.UnimplementedAccountServer

	app             *app.App
	ecdsaPrivateKey *ecdsa.PrivateKey
}

// NewServer は gRPC サーバーを作成する
func NewServer(conf *Config) *Server {
	return &Server{
		app:             conf.App,
		ecdsaPrivateKey: conf.ECDSAPrivateKey,
	}
}
