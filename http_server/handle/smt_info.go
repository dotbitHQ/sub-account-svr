package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"fmt"
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
		funcName = "SmtInfo"
		clientIp = GetClientIp(ctx)
		req      ReqSmtInfo
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

	if err = h.doSmtInfo(&req, &apiResp); err != nil {
		log.Error("doSmtInfo err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSmtInfo(req *ReqSmtInfo, apiResp *api_code.ApiResp) error {
	var resp RespSmtInfo

	mongoStore := smt.NewMongoStore(h.Ctx, h.Mongo, config.Cfg.DB.Mongo.SmtDatabase, req.ParentAccountId)
	tree := smt.NewSparseMerkleTree(mongoStore)

	root, err := tree.Root()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("tree.Root() err: %s", err.Error())
	}
	resp.Root = root.String()

	apiResp.ApiRespOK(resp)
	return nil
}
