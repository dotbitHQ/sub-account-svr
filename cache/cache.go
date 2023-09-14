package cache

import (
	"context"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/go-redis/redis"
)

type RedisCache struct {
	Ctx context.Context
	Red *redis.Client
}

var (
	log = logger.NewLogger("cache", logger.LevelDebug)
)
