package main

import (
	"context"
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/http_server"
	"das_sub_account/http_server/handle"
	"das_sub_account/lb"
	"fmt"
	"github.com/scorpiotzh/mylog"
	"github.com/scorpiotzh/toolib"
	"github.com/urfave/cli/v2"
	"os"
	"sync"
	"time"
)

var (
	log               = mylog.NewLogger("main", mylog.LevelDebug)
	exit              = make(chan struct{})
	ctxServer, cancel = context.WithCancel(context.Background())
	wgServer          = sync.WaitGroup{}
)

func main() {
	log.Debugf("start：")
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Load configuration from `FILE`",
			},
		},
		Action: runServer,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runServer(ctx *cli.Context) error {
	// config file
	configFilePath := ctx.String("config")
	if err := config.InitCfg(configFilePath); err != nil {
		return err
	}

	// config file watcher
	watcher, err := config.AddCfgFileWatcher(configFilePath)
	if err != nil {
		return fmt.Errorf("AddCfgFileWatcher err: %s", err.Error())
	}
	// ============= service start =============

	// redis
	red, err := toolib.NewRedisClient(config.Cfg.Cache.Redis.Addr, config.Cfg.Cache.Redis.Password, config.Cfg.Cache.Redis.DbNum)
	if err != nil {
		return fmt.Errorf("NewRedisClient err:%s", err.Error())
	} else {
		log.Info("redis ok")
	}
	rc := &cache.RedisCache{
		Ctx: ctxServer,
		Red: red,
	}

	// lb
	if len(config.Cfg.Slb.Servers) == 0 {
		return fmt.Errorf("slb servers is nil")
	}
	slb := lb.NewLoadBalancing(config.Cfg.Slb.Servers)

	// http
	lbHS := http_server.LbHttpServer{
		Ctx:     ctxServer,
		Address: config.Cfg.Server.HttpServerAddr,
		H: &handle.LBHttpHandle{
			Ctx: ctxServer,
			RC:  rc,
			LB:  slb,
		},
	}
	lbHS.Run()

	log.Info("http server ok")
	// ============= service end =============
	toolib.ExitMonitoring(func(sig os.Signal) {
		log.Warn("ExitMonitoring:", sig.String())
		if watcher != nil {
			log.Warn("close watcher ... ")
			_ = watcher.Close()
		}
		cancel()
		wgServer.Wait()
		log.Warn("success exit server. bye bye!")
		time.Sleep(time.Second)
		exit <- struct{}{}
	})

	<-exit
	return nil
}
