package handle

import (
	"das_sub_account/http_server/api_code"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqAutoOrderHash struct {
}

type RespAutoOrderHash struct {
}

func (h *HttpHandle) AutoOrderHash(ctx *gin.Context) {
	var (
		funcName = "AutoOrderHash"
		clientIp = GetClientIp(ctx)
		req      ReqAutoOrderHash
		apiResp  api_code.ApiResp
		err      error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req))

	if err = h.doAutoOrderHash(&req, &apiResp); err != nil {
		log.Error("doAutoOrderHash err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doAutoOrderHash(req *ReqAutoOrderHash, apiResp *api_code.ApiResp) error {
	var resp RespAutoOrderHash

	apiResp.ApiRespOK(resp)
	return nil
}
