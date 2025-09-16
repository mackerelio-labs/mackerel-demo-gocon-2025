# Mackerel Demo Go Conference 2025

「Mackerel Demo Go Conference 2025」は, [Hatena SUMMER INTERNSHIP 2025](https://github.com/hatena/Hatena-Intern-2025-Public) のブログシステムをベースとして, 各サービスを OpenTelemetry で計装し, テレメトリーを Mackerel に送信します. ブログシステムはマイクロサービスを意識しており, メインであるブログサービスに加えて, アカウントサービスと, Markdown などの記法を変換するサービスが用意されています. それぞれのサービス間は gRPC を使ってやりとりしています.

### アプリケーションの起動

```console
export MACKEREL_APIKEY="[YOUR API KEY HERE]"
docker compose up
```

しばらく待つとアプリケーションが起動したログが出力されます

```
renderer-1    | 2025-09-11T03:25:03.092Z	INFO	app/main.go:49	starting gRPC server (port = 50051)
account-1     | 2025-09-11T03:25:03.148Z	INFO	app/main.go:66	starting gRPC server (port = 50051)
blog-1        | 2025-09-11T03:25:03.228Z	INFO	app/main.go:98	starting web server (port = 8080)
```

## サービス

アプリケーションには以下の 3 つのサービスが存在します.

- 認証基盤 (Account) サービス
  - ユーザーアカウントの登録や認証を管轄します
- ブログ (Blog) サービス
  - ユーザーに対して, ブログを作成したり記事を書いたりする機能を提供します
- 記法変換 (Renderer) サービス
  - ブログの記事を記述するための「記法」から HTML への変換を担います

このうちブログサービスが Web サーバーとして動作し, ユーザーに対してアプリケーションを操作するためのインターフェースを提供します.
認証基盤サービスと記法変換サービスは gRPC サービスとして動作し, ブログサービスから使用されます.
また, 各サービスはそれぞれ Maclerel にテレメトリーを送信します.

## ディレクトリ構成

- `config/`: 各サービスの設定
- `pb/`: gRPC サービスの定義
- `services/`: 各サービスの実装
  - `account/`: 認証基盤サービス
  - `blog/`: ブログサービス
  - `renderer/`: 記法変換サービス

## クレジット

- 株式会社はてな
  - [Hatena SUMMER INTERNSHIP 2025](https://github.com/hatena/Hatena-Intern-2025-Public)
    - [@akiym](https://github.com/akiym)
    - [@cockscomb](https://github.com/cockscomb)
    - [@itchyny](https://github.com/itchyny)
    - [@susisu](https://github.com/susisu)
    - [@astj](https://github.com/astj)
    - [@tkzwtks](https://github.com/tkzwtks)
    - [@SlashNephy](https://github.com/SlashNephy)
  - Mackerel Demo Go Conference 2025
    - [@yohfee](https://github.com/yohfee)
    - [@lufia](https://github.com/lufia)

(順不同)

このリポジトリの内容は MIT ライセンスで提供されます. 詳しくは `LICENSE` をご確認ください.
