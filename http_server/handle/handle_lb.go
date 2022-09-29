package handle

import (
	"bytes"
	"das_sub_account/http_server/api_code"
	"encoding/json"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/scorpiotzh/toolib"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func (h *LBHttpHandle) LBProxy(ctx *gin.Context) {
	var (
		funcName = "LBProxy"
		clientIp = GetClientIp(ctx)
		apiResp  api_code.ApiResp
	)
	log.Info("ApiReq:", funcName, clientIp, ctx.Request.URL.Path)

	// slb by ip
	h.doLBProxy(ctx, &apiResp, clientIp)

}

func (h *LBHttpHandle) doLBProxy(ctx *gin.Context, apiResp *api_code.ApiResp, serverKey string) {
	server := h.LB.GetServer(serverKey)
	if server.Url == "" {
		log.Error("h.LB.GetServer err: server url is nil")
		apiResp.ApiRespErr(api_code.ApiCodeError500, "proxy server is nil")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	origin := ctx.GetHeader("origin")
	log.Info("LBProxy:", serverKey, server.Name, server.Url, origin)
	u, err := url.Parse(server.Url)
	if err != nil {
		log.Error("url.Parse err: %s", err.Error())
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ModifyResponse = func(response *http.Response) error {
		for k, v := range response.Header {
			if len(v) > 1 {
				log.Info("doLBProxy:", k, len(v))
				response.Header[k] = v[:1]
			}
		}
		return nil
	}
	proxy.ServeHTTP(ctx.Writer, ctx.Request)
	//ctx.Abort()
}

func (h *LBHttpHandle) LBSubAccountCreate(ctx *gin.Context) {
	var (
		funcName = "LBSubAccountCreate"
		clientIp = GetClientIp(ctx)
		apiResp  api_code.ApiResp
		req      ReqSubAccountCreate
	)
	log.Info("ApiReq:", funcName, clientIp)

	bodyBytes, _ := ctx.GetRawData()
	ctx.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	log.Info("LBSubAccountCreate:", string(bodyBytes))
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Error("json.Unmarshal err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	serverKey := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	h.doLBProxy(ctx, &apiResp, serverKey)
}

func (h *LBHttpHandle) LBTransactionSend(ctx *gin.Context) {
	var (
		funcName = "LBTransactionSend"
		clientIp = GetClientIp(ctx)
		apiResp  api_code.ApiResp
		req      ReqTransactionSend
	)
	log.Info("ApiReq:", funcName, clientIp)

	bodyBytes, _ := ctx.GetRawData()
	ctx.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	log.Info("LBTransactionSend:", string(bodyBytes))
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Error("json.Unmarshal err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	serverKey := clientIp
	if req.Action == common.DasActionEditSubAccount {
		var editCache EditSubAccountCache
		if txStr, err := h.RC.GetSignTxCache(req.SignKey); err != nil {
			if err == redis.Nil {
				apiResp.ApiRespErr(api_code.ApiCodeTxExpired, "sign key not exist(tx expired)")
			} else {
				apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
			}
			ctx.JSON(http.StatusOK, apiResp)
			return
		} else if err = json.Unmarshal([]byte(txStr), &editCache); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "json.Unmarshal err")
			ctx.JSON(http.StatusOK, apiResp)
			return
		}
		log.Info("EditSubAccountCache:", toolib.JsonString(&editCache))
		serverKey = editCache.ParentAccountId
	}

	h.doLBProxy(ctx, &apiResp, serverKey)
}
