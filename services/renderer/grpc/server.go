package grpc

import (
	"context"

	pb "github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/renderer/pb/renderer"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/renderer/renderer"
)

// Server は pb.RendererServer に対する実装
type Server struct {
	pb.UnimplementedRendererServer
}

// NewServer は gRPC サーバーを作成する
func NewServer() *Server {
	return &Server{}
}

// Render は受け取った文書を HTML に変換する
func (s *Server) Render(ctx context.Context, in *pb.RenderRequest) (*pb.RenderReply, error) {
	html, err := renderer.Render(ctx, in.Src)
	if err != nil {
		return nil, err
	}
	return &pb.RenderReply{Html: html}, nil
}
