package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config は各種設定をまとめたもの
type Config struct {
	Mode                string
	GRPCPort            int
	GracefulStopTimeout time.Duration
	MackerelAPIKey      string
	TraceEndpoint       string
	MetricEndpoint      string
	ServiceName         string
	ServiceNameSpace    string
	ServiceVersion      string
}

// Load は環境変数から設定を読み込む
func Load() (*Config, error) {
	conf := &Config{
		Mode:                "development",
		GRPCPort:            50051,
		GracefulStopTimeout: 10 * time.Second,
		TraceEndpoint:       "otlp-vaxila.mackerelio.com",
		MetricEndpoint:      "otlp.mackerelio.com:4317",
		ServiceName:         "renderer",
		ServiceNameSpace:    "mackerel-demo-gocon-2025",
		ServiceVersion:      "unknown",
	}

	// Mode
	mode := os.Getenv("MODE")
	if mode != "" {
		conf.Mode = mode
	}

	// GRPCPort
	grpcPortStr := os.Getenv("GRPC_PORT")
	if grpcPortStr != "" {
		grpcPort, err := strconv.Atoi(os.Getenv("GRPC_PORT"))
		if err != nil {
			return nil, fmt.Errorf("GRPC_PORT is invalid: %v", err)
		}
		conf.GRPCPort = grpcPort
	}

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
