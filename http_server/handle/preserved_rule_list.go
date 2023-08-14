package handle

import (
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqPreservedRuleList struct {
	core.ChainTypeAddress
	Account string `json:"account" binding:"required"`
}

func (h *HttpHandle) PreservedRuleList(ctx *gin.Context) {
	var (
		funcName               = "PreservedRuleList"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqPriceRuleList
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req))

	if err = h.doRuleList(common.ActionDataTypeSubAccountPreservedRules, &req, &apiResp); err != nil {
		log.Error("doRuleList err:", err.Error(), funcName, clientIp, remoteAddrIP)
	}
	ctx.JSON(http.StatusOK, apiResp)
}
