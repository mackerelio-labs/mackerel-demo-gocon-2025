package config

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config は各種設定をまとめたもの
type Config struct {
	Mode                  string
	Port                  int
	DatabaseDSN           string
	AccountAddr           string
	AccountECDSAPublicKey *ecdsa.PublicKey
	RendererAddr          string
	GracefulStopTimeout   time.Duration
	MackerelAPIKey        string
	TraceEndpoint         string
	MetricEndpoint        string
	ServiceName           string
	ServiceNameSpace      string
	ServiceVersion        string
}

// Load は環境変数から設定を読み込む
func Load() (*Config, error) {
	conf := &Config{
		Mode:                "development",
		Port:                8080,
		GracefulStopTimeout: 10 * time.Second,
		TraceEndpoint:       "otlp-vaxila.mackerelio.com",
		MetricEndpoint:      "otlp.mackerelio.com:4317",
		ServiceName:         "blog",
		ServiceNameSpace:    "mackerel-demo-gocon-2025",
		ServiceVersion:      "unknown",
	}

	// Mode
	mode := os.Getenv("MODE")
	if mode != "" {
		conf.Mode = mode
	}

	portStr := os.Getenv("PORT")
	if portStr != "" {
		port, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			return nil, fmt.Errorf("invalid PORT: %v", err)
		}
		conf.Port = port
	}

	// DatabaseDSN
	databaseDSN := os.Getenv("DATABASE_DSN")
	if databaseDSN == "" {
		return nil, errors.New("DATABASE_DSN is not set")
	}
	conf.DatabaseDSN = databaseDSN

	// AccountAddr
	accountAddr := os.Getenv("ACCOUNT_ADDR")
	if accountAddr == "" {
		return nil, errors.New("ACCOUNT_ADDR is not set")
	}
	conf.AccountAddr = accountAddr

	// AccountECDSAPublicKey
	accountECDSAPublicKeyFile := os.Getenv("ACCOUNT_ECDSA_PUBLIC_KEY_FILE")
	if accountECDSAPublicKeyFile == "" {
		return nil, errors.New("ECDSA_PRIVATE_KEY_FILE is not set")
	}
	accountECDSAPublicKey, err := loadECDSAPublicKey(accountECDSAPublicKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key from %s", accountECDSAPublicKeyFile)
	}
	conf.AccountECDSAPublicKey = accountECDSAPublicKey

	// RendererAddr
	rendererAddr := os.Getenv("RENDERER_ADDR")
	if rendererAddr == "" {
		return nil, errors.New("RENDERER_ADDR is not set")
	}
	conf.RendererAddr = rendererAddr

	// GracefulStopTimeout
	gracefulStopTimeout := os.Getenv("GRACEFUL_STOP_TIMEOUT")
	if gracefulStopTimeout != "" {
		d, err := time.ParseDuration(gracefulStopTimeout)
		if err != nil {
			return nil, fmt.Errorf("GRACEFUL_STOP_TIMEOUT is invalid: %v", err)
		}
		conf.GracefulStopTimeout = d
	}

	// MackerelAPIKey
	mackerelAPIKey := os.Getenv("MACKEREL_APIKEY")
	if mackerelAPIKey == "" {
		return nil, fmt.Errorf("MACKEREL_APIKEY is required")
	}
	conf.MackerelAPIKey = mackerelAPIKey

	// TraceEndpoint
	traceEndpoint := os.Getenv("TRACE_ENDPOINT")
	if traceEndpoint != "" {
		conf.TraceEndpoint = traceEndpoint
	}

	// MetricEndpoint
	metricEndpoint := os.Getenv("METRIC_ENDPOINT")
	if metricEndpoint != "" {
		conf.MetricEndpoint = metricEndpoint
	}

	// ServiceName
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName != "" {
		conf.ServiceName = serviceName
	}

	// ServiceNameSpace
	serviceNameSpace := os.Getenv("SERVICE_NAME_SPACE")
	if serviceNameSpace != "" {
		conf.ServiceNameSpace = serviceNameSpace
	}

	// ServiceVersion
	serviceVersion := os.Getenv("SERVICE_VERSION")
	if serviceVersion != "" {
		conf.ServiceVersion = serviceVersion
	}

	return conf, nil
}

func loadECDSAPublicKey(file string) (*ecdsa.PublicKey, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to load public key")
	}
	rawKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := rawKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("failed to load public key")
	}
	return key, nil
}
