package handle

import (
	"fmt"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqSmtInfo struct {
	ParentAccountId string `json:"parent_account_id"`
}

type RespSmtInfo struct {
	Root string `json:"root"`
}

func (h *HttpHandle) SmtInfo(ctx *gin.Context) {
	var (
		funcName               = "SmtInfo"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqSmtInfo
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx)

	if err = h.doSmtInfo(&req, &apiResp); err != nil {
		log.Error("doSmtInfo err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSmtInfo(req *ReqSmtInfo, apiResp *api_code.ApiResp) error {
	var resp RespSmtInfo
	tree := smt.NewSmtSrv(*h.SmtServerUrl, req.ParentAccountId)
	root, err := tree.GetSmtRoot()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("tree.Root() err: %s", err.Error())
	}
	resp.Root = root.String()

	apiResp.ApiRespOK(resp)
	return nil
}
