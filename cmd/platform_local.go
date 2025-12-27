//go:build !gcloud

package main

import (
	"context"
	"os"

	"github.com/KasumiMercury/primind-central-backend/internal/observability"
	"github.com/KasumiMercury/primind-central-backend/internal/observability/logging"
)

func initObservability(ctx context.Context) (*observability.Resources, error) {
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "central-backend"
	}

	env := logging.EnvDev
	if e := os.Getenv("ENV"); e != "" {
		env = logging.Environment(e)
	}

	obs, err := observability.Init(ctx, observability.Config{
		ServiceInfo: logging.ServiceInfo{
			Name:     serviceName,
			Version:  Version,
			Revision: "",
		},
		Environment:   env,
		GCPProjectID:  "",
		SamplingRate:  1.0,
		DefaultModule: logging.Module("central"),
	})
	if err != nil {
		return nil, err
	}

	return obs, nil
}
