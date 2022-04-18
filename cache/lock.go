package cache

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	lockTime      = 180
	lockTicker    = 10
	lockAccountId = "lock:account_id:"
)

var ErrDistributedLockPreemption = errors.New("distributed lock preemption")

func (r *RedisCache) LockWithRedis(accountId string) error {
	ret := r.Red.SetNX(lockAccountId+accountId, accountId, time.Second*lockTime)
	if err := ret.Err(); err != nil {
		return fmt.Errorf("redis set order nx-->%s", err.Error())
	}
	if !ret.Val() {
		log.Info("LockWithRedis lock:", accountId)
		return ErrDistributedLockPreemption
	}
	log.Info("LockWithRedis:", accountId)
	return nil
}

func (r *RedisCache) UnLockWithRedis(accountId string) error {
	ret := r.Red.Del(lockAccountId + accountId)
	if err := ret.Err(); err != nil {
		return fmt.Errorf("redis del order nx-->%s", err.Error())
	}
	log.Info("UnLockWithRedis:", accountId)
	return nil
}

func (r *RedisCache) DoLockExpire(ctx context.Context, accountId string) {
	ticker := time.NewTicker(time.Second * lockTicker)
	count := 0
	go func() {
		for {
			select {
			case <-ticker.C:
				ok, err := r.Red.Expire(lockAccountId+accountId, time.Second*lockTime).Result()
				if err != nil {
					log.Error("DoLockExpire err: ", err.Error(), accountId)
				} else if ok {
					count++
				}
				log.Infof("DoLockExpire: %s %d %p", accountId, count, &count)
			case <-ctx.Done():
				log.Info("DoLockExpire done:", accountId)
				return
			}
		}
	}()

}
