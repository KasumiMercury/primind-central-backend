package config

import (
	"fmt"

	"github.com/KasumiMercury/primind-central-backend/internal/config/persistence"
)

type Config struct {
	Persistence *persistence.Config
}

func Load() (*Config, error) {
	persistenceCfg, err := persistence.Load()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPersistenceLoad, err)
	}

	return &Config{Persistence: persistenceCfg}, nil
}
