package api_code

import (
	"bytes"
	"das_sub_account/config"
	"das_sub_account/txtool"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/gin-gonic/gin"
	"github.com/parnurzeal/gorequest"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"time"
)

var (
	log        = logger.NewLogger("api_code", logger.LevelDebug)
	ApiSummary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "api",
	}, []string{"method", "ip", "http_status", "err_no", "err_msg"})
)

func init() {
	txtool.PromRegister.MustRegister(ApiSummary)
}

type ReqPushLog struct {
	Index   string        `json:"index"`
	Method  string        `json:"method"`
	Ip      string        `json:"ip"`
	Latency time.Duration `json:"latency"`
	ErrMsg  string        `json:"err_msg"`
	ErrNo   int           `json:"err_no"`
}

func PushLog(url string, req ReqPushLog) {
	if url == "" {
		return
	}
	go func() {
		resp, _, errs := gorequest.New().Post(url).SendStruct(&req).End()
		if len(errs) > 0 {
			log.Error("PushLog err:", errs)
		} else if resp.StatusCode != http.StatusOK {
			log.Error("PushLog StatusCode:", resp.StatusCode)
		}
	}()
}

func DoMonitorLog(method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startTime := time.Now()
		ip := getClientIp(ctx)

		blw := &bodyWriter{body: bytes.NewBufferString(""), ResponseWriter: ctx.Writer}
		ctx.Writer = blw
		ctx.Next()
		statusCode := ctx.Writer.Status()

		var resp ApiResp
		if statusCode == http.StatusOK && blw.body.String() != "" {
			if err := json.Unmarshal(blw.body.Bytes(), &resp); err != nil {
				log.Warn("DoMonitorLog Unmarshal err:", method, err)
				return
			}
			if resp.ErrNo != http_api.ApiCodeSuccess {
				log.Warn("DoMonitorLog:", method, resp.ErrNo, resp.ErrMsg)
			}
			if resp.ErrNo == http_api.ApiCodeSignError {
				resp.ErrNo = http_api.ApiCodeSuccess
			}
			if resp.ErrNo == http_api.ApiCodeAccountExpiringSoon {
				resp.ErrNo = http_api.ApiCodeSuccess
			}
			if resp.ErrNo == http_api.ApiCodeAccountIsExpired {
				resp.ErrNo = http_api.ApiCodeSuccess
			}
		}
		ApiSummary.WithLabelValues(method, ip, fmt.Sprint(statusCode), fmt.Sprint(resp.ErrNo), resp.ErrMsg).Observe(time.Since(startTime).Seconds())
	}
}

func DoMonitorLogRpc(apiResp *http_api.ApiResp, method, clientIp string, startTime time.Time) {
	pushLog := ReqPushLog{
		Index:   config.Cfg.Server.PushLogIndex,
		Method:  method,
		Ip:      clientIp,
		Latency: time.Since(startTime),
		ErrMsg:  apiResp.ErrMsg,
		ErrNo:   apiResp.ErrNo,
	}
	if apiResp.ErrNo != http_api.ApiCodeSuccess {
		log.Warn("DoMonitorLog:", method, apiResp.ErrNo, apiResp.ErrMsg)
	}
	PushLog(config.Cfg.Server.PushLogUrl, pushLog)
}

func getClientIp(ctx *gin.Context) string {
	return fmt.Sprintf("%v", ctx.Request.Header.Get("X-Real-IP"))
}

type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (b bodyWriter) Write(bys []byte) (int, error) {
	b.body.Write(bys)
	return b.ResponseWriter.Write(bys)
}
