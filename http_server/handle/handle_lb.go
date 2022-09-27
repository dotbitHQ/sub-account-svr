package handle

import (
	"das_sub_account/http_server/api_code"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func (h *HttpHandle) LBProxy(ctx *gin.Context) {
	var (
		funcName = "LBProxy"
		clientIp = GetClientIp(ctx)
		apiResp  api_code.ApiResp
		err      error
	)
	log.Info("ApiReq:", funcName, clientIp)

	server := h.LB.GetServer(clientIp)
	if server.Url == "" {
		log.Error("h.LB.GetServer err: server url is nil")
		apiResp.ApiRespErr(api_code.ApiCodeError500, "proxy server is nil")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	log.Info("LBProxy:", server.Name, server.Url)
	u, err := url.Parse(server.Url)
	if err != nil {
		log.Error("url.Parse err: %s", err.Error())
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ServeHTTP(ctx.Writer, ctx.Request)
	ctx.Abort()
}

func (h *HttpHandle) LBSubAccountCreate(ctx *gin.Context) {

}

func (h *HttpHandle) LBSubAccountEdit(ctx *gin.Context) {

}
