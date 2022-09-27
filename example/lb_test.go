package example

import (
	"context"
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/http_server"
	"das_sub_account/http_server/handle"
	"das_sub_account/lb"
	"fmt"
	"testing"
)

func TestLB(t *testing.T) {
	var list []config.Server
	list = append(list, config.Server{
		Name:   "svr-1",
		Url:    "http://127.0.0.1:8888",
		Weight: 1,
	})
	list = append(list, config.Server{
		Name:   "svr-2",
		Url:    "http://127.0.0.1:8889",
		Weight: 1,
	})
	list = append(list, config.Server{
		Name:   "svr-3",
		Url:    "http://127.0.0.1:8887",
		Weight: 1,
	})
	list = append(list, config.Server{
		Name:   "svr-4",
		Url:    "http://127.0.0.1:8887",
		Weight: 0,
	})
	slb := lb.NewLoadBalancing(list)

	accList := []string{"aaa.bit", "bbb.bit", "ccc.bit"}
	for _, v := range accList {
		fmt.Println(slb.GetServer(v), v)
	}

	s := slb.GetServers()
	for i, v := range s {
		fmt.Println(i, v.Name, v.Url)
	}
}

func TestLBHttp(t *testing.T) {
	config.Cfg.Server.RunMode = "normal"
	ctxServer := context.Background()
	hs1 := http_server.HttpServer{
		Ctx:             ctxServer,
		Address:         ":8125",
		InternalAddress: ":8126",
		H: &handle.HttpHandle{
			Ctx: ctxServer,
			RC:  &cache.RedisCache{},
		},
	}
	hs1.Run()

	hs2 := http_server.HttpServer{
		Ctx:             ctxServer,
		Address:         ":8127",
		InternalAddress: ":8128",
		H: &handle.HttpHandle{
			Ctx: ctxServer,
			RC:  &cache.RedisCache{},
		},
	}
	hs2.Run()

	//
	var list []config.Server
	list = append(list, config.Server{
		Name:   "svr-1",
		Url:    "http://127.0.0.1:8125",
		Weight: 1,
	})
	list = append(list, config.Server{
		Name:   "svr-2",
		Url:    "http://127.0.0.1:8127",
		Weight: 1,
	})
	slb := lb.NewLoadBalancing(list)
	lbhs := http_server.LbHttpServer{
		Ctx:     ctxServer,
		Address: ":8129",
		H: &handle.LBHttpHandle{
			Ctx: ctxServer,
			RC:  &cache.RedisCache{},
			LB:  slb,
		},
	}
	lbhs.Run()

	select {}
}
