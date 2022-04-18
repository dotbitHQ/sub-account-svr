package example

import (
	"context"
	"das_sub_account/cache"
	"fmt"
	"github.com/scorpiotzh/toolib"
	"testing"
	"time"
)

func TestLock(t *testing.T) {
	addr, password := "127.0.0.1:6379", ""
	red, err := toolib.NewRedisClient(addr, password, 0)
	if err != nil {
		t.Fatal(err)
	}

	rc := &cache.RedisCache{
		Ctx: context.Background(),
		Red: red,
	}

	doLock := func(accountId string) error {
		if err := rc.LockWithRedis(accountId); err != nil {
			return fmt.Errorf("rc.LockWithRedis err: %s", err.Error())
		}
		ctx, cancel := context.WithCancel(context.Background())

		defer func() {
			if err := rc.UnLockWithRedis(accountId); err != nil {
				fmt.Println("UnLockWithRedis:", err.Error())
			}
			cancel()
		}()

		rc.DoLockExpire(ctx, accountId)

		time.Sleep(time.Second * 10)
		return nil
	}

	for {
		go func() {
			accountId := "0x111"
			if err := doLock(accountId); err != nil {
				fmt.Println("doLock err:", err.Error())
			}
		}()
		time.Sleep(time.Second * 2)
	}
}
