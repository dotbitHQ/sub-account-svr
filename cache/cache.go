package cache

import (
	"context"
	"github.com/go-redis/redis"
	"github.com/scorpiotzh/mylog"
)

type RedisCache struct {
	Ctx context.Context
	Red *redis.Client
}

var (
	log = mylog.NewLogger("cache", mylog.LevelDebug)
)
