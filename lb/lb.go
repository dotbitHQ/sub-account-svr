package lb

import (
	"das_sub_account/config"
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
)

type LoadBalancing struct {
	servers []Server
}

func NewLoadBalancing(list []config.Server) *LoadBalancing {
	var lb LoadBalancing
	for _, v := range list {
		totalNum := defaultNum * v.Weight
		for i := 0; i < totalNum; i++ {
			tmpStr := fmt.Sprintf("%s:%s:%s", v.Name, v.Url, strconv.Itoa(i))
			spotVal := getUint32Val(tmpStr)
			lb.servers = append(lb.servers, Server{
				Name:    v.Name,
				Url:     v.Url,
				spotVal: spotVal,
			})
		}
	}
	lb.Sort()
	return &lb
}

func (l *LoadBalancing) GetServer(key string) Server {
	uint32Val := getUint32Val(key)
	i := sort.Search(l.Len(), func(i int) bool { return l.servers[i].spotVal >= uint32Val })
	return l.servers[i]
}
func (l *LoadBalancing) GetServers() []Server {
	return l.servers
}

type Server struct {
	Name    string
	Url     string
	spotVal uint32
}

const defaultNum = 100

func (l *LoadBalancing) Len() int { return len(l.servers) }
func (l *LoadBalancing) Less(i, j int) bool {
	return l.servers[i].spotVal < l.servers[j].spotVal
}
func (l *LoadBalancing) Swap(i, j int) {
	l.servers[i], l.servers[j] = l.servers[j], l.servers[i]
}
func (l *LoadBalancing) Sort() { sort.Sort(l) }

func getUint32Val(s string) (v uint32) {
	return crc32.ChecksumIEEE([]byte(s))
	//h := sha1.New()
	//defer h.Reset()
	//h.Write([]byte(s))
	//hashBytes := h.Sum(nil)
	//if len(hashBytes[4:8]) == 4 {
	//	v = (uint32(hashBytes[3]) << 24) | (uint32(hashBytes[2]) << 12) | (uint32(hashBytes[1]) << 6) | (uint32(hashBytes[0]) << 3)
	//}
	//
	//return
}
