package renderer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

// customHeadingRenderer は見出しノードを処理するカスタムレンダラー
type customHeadingRenderer struct{}

func (r customHeadingRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHeading, func(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		heading := node.(*ast.Heading)

		if entering {
			// FIXME: 重いレンダリングのシミュレーションとして見出しレベルに応じて待機時間を追加
			time.Sleep(time.Duration(heading.Level) * time.Second)

			fmt.Fprintf(w, "<h%d class=\"text-[%dpx]\">", heading.Level, (8-heading.Level)*8)
		} else {
			fmt.Fprintf(w, "</h%d>", heading.Level)
		}

		return ast.WalkContinue, nil
	})
}

// customImageRenderer は画像ノードを処理するカスタムレンダラー
type customImageRenderer struct{}

func (r customImageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, func(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		// FIXME: 画像ノードはサポートしない
		return ast.WalkStop, errors.New("image nodes are not supported")
	})
}

var tracer = otel.Tracer("renderer")

// Render は受け取った文書を HTML に変換する
func Render(ctx context.Context, src string) (string, error) {
	_, span := tracer.Start(ctx, "renderer.Render")
	defer span.End()

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(
				util.Prioritized(&customHeadingRenderer{}, 100),
				util.Prioritized(&customImageRenderer{}, 100),
			),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		span.SetStatus(codes.Error, "failed to convert markdown to HTML")
		span.RecordError(err)
		return "", err
	}

	return buf.String(), nil
}
