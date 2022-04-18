package cache

import (
	"fmt"
	"time"
)

func (r *RedisCache) getSignTxCacheKey(key string) string {
	return "sign:tx:" + key
}

func (r *RedisCache) GetSignTxCache(key string) (string, error) {
	if r.Red == nil {
		return "", fmt.Errorf("redis is nil")
	}
	key = r.getSignTxCacheKey(key)
	if txStr, err := r.Red.Get(key).Result(); err != nil {
		return "", err
	} else {
		return txStr, nil
	}
}

func (r *RedisCache) SetSignTxCache(key, txStr string) error {
	if r.Red == nil {
		return fmt.Errorf("redis is nil")
	}
	key = r.getSignTxCacheKey(key)
	if err := r.Red.Set(key, txStr, time.Minute*10).Err(); err != nil {
		return err
	}
	return nil
}
