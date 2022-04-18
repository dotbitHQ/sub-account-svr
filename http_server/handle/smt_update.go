package handle

import (
	"context"
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/smt"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqSmtUpdate struct {
	ParentAccountId string `json:"parent_account_id"`
	SubAccountId    string `json:"sub_account_id"`
	Value           string `json:"value"`
}

type RespSmtUpdate struct {
	Root string `json:"root"`
}

func (h *HttpHandle) SmtUpdate(ctx *gin.Context) {
	var (
		funcName = "SmtUpdate"
		clientIp = GetClientIp(ctx)
		req      ReqSmtUpdate
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

	if err = h.doSmtUpdate(&req, &apiResp); err != nil {
		log.Error("doSmtUpdate err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSmtUpdate(req *ReqSmtUpdate, apiResp *api_code.ApiResp) error {
	var resp RespSmtUpdate

	if req.ParentAccountId == "" || req.SubAccountId == "" || req.Value == "" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil
	}
	log.Info("doSmtUpdate:", req.ParentAccountId, req.SubAccountId, req.Value)

	parentAccountId := req.ParentAccountId
	// do check
	resCheck, err := h.TxTool.DoCheckBeforeBuildTx(parentAccountId)
	if err != nil {
		if resCheck != nil && resCheck.Continue {
			apiResp.ApiRespErr(api_code.ApiCodeTaskInProgress, "task in progress")
			return nil
		}
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("DoCheckBeforeBuildTx err: %s", err.Error())
	}

	// lock smt and unlock
	if err := h.RC.LockWithRedis(parentAccountId); err != nil {
		if err == cache.ErrDistributedLockPreemption {
			apiResp.ApiRespErr(api_code.ApiCodeDistributedLockPreemption, err.Error())
		} else {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		}
		return fmt.Errorf("LockWithRedis err: %s", err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err := h.RC.UnLockWithRedis(parentAccountId); err != nil {
			fmt.Println("UnLockWithRedis:", err.Error())
		}
		cancel()
	}()
	h.RC.DoLockExpire(ctx, parentAccountId)

	// get smt tree
	mongoStore := smt.NewMongoStore(h.Ctx, h.Mongo, config.Cfg.DB.Mongo.SmtDatabase, parentAccountId)
	tree := smt.NewSparseMerkleTree(mongoStore)

	key := smt.AccountIdToSmtH256(req.SubAccountId)
	value := common.Hex2Bytes(req.Value)

	if err := tree.Update(key, value); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("tree.Update err: %s", err.Error())
	}

	if root, err := tree.Root(); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("tree.Root err: %s", err.Error())
	} else {
		resp.Root = root.String()
	}

	apiResp.ApiRespOK(resp)
	return nil
}
