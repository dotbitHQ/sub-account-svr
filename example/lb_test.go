package example

import (
	"das_sub_account/config"
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
