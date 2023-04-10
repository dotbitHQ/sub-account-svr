package handle

import (
	"das_sub_account/http_server/api_code"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqAutoOrderCreate struct {
}

type RespAutoOrderCreate struct {
}

func (h *HttpHandle) AutoOrderCreate(ctx *gin.Context) {
	var (
		funcName = "AutoOrderCreate"
		clientIp = GetClientIp(ctx)
		req      ReqAutoOrderCreate
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

	if err = h.doAutoOrderCreate(&req, &apiResp); err != nil {
		log.Error("doAutoOrderCreate err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doAutoOrderCreate(req *ReqAutoOrderCreate, apiResp *api_code.ApiResp) error {
	var resp RespAutoOrderCreate

	apiResp.ApiRespOK(resp)
	return nil
}
