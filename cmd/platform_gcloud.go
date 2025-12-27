//go:build gcloud

package main

import (
	"context"
	"os"

	"github.com/KasumiMercury/primind-central-backend/internal/observability"
	"github.com/KasumiMercury/primind-central-backend/internal/observability/logging"
)

func initObservability(ctx context.Context) (*observability.Resources, error) {
	serviceName := os.Getenv("K_SERVICE")
	if serviceName == "" {
		serviceName = "central-backend"
	}

	env := logging.EnvProd
	if e := os.Getenv("ENV"); e != "" {
		env = logging.Environment(e)
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GCLOUD_PROJECT_ID")
	}

	obs, err := observability.Init(ctx, observability.Config{
		ServiceInfo: logging.ServiceInfo{
			Name:     serviceName,
			Version:  Version,
			Revision: os.Getenv("K_REVISION"),
		},
		Environment:   env,
		GCPProjectID:  projectID,
		SamplingRate:  1.0,
		DefaultModule: logging.Module("central"),
	})
	if err != nil {
		return nil, err
	}

	return obs, nil
}
