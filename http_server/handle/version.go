package handle

import (
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
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req), ctx)

	//time.Sleep(time.Minute * 3)
	if err = h.doVersion(&req, &apiResp); err != nil {
		log.Error("doVersion err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doVersion(req *ReqVersion, apiResp *api_code.ApiResp) error {
	var resp RespVersion

	resp.Version = req.Version
	apiResp.ApiRespOK(resp)
	return nil
}
