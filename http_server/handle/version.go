package handle

import (
	"context"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqVersion struct {
	Version string `json:"version"`
}

type RespVersion struct {
	Version string `json:"version"`
}

func (h *HttpHandle) Version(ctx *gin.Context) {
	var (
		funcName               = "Version"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqVersion
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx.Request.Context())
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req), ctx.Request.Context())

	//time.Sleep(time.Minute * 3)
	if err = h.doVersion(ctx.Request.Context(), &req, &apiResp); err != nil {
		log.Error("doVersion err:", err.Error(), funcName, clientIp, ctx.Request.Context())
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doVersion(ctx context.Context, req *ReqVersion, apiResp *api_code.ApiResp) error {
	var resp RespVersion
	resp.Version = req.Version
	apiResp.ApiRespOK(resp)
	return nil
}
