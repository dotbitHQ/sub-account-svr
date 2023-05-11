package handle

import (
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/scorpiotzh/toolib"
	"math"
	"net/http"
)

type ReqPriceRuleList struct {
	Account string `json:"account" binding:"required"`
}

type RespPriceRuleList struct {
	List interface{} `json:"list"`
}

func (h *HttpHandle) PriceRuleList(ctx *gin.Context) {
	var (
		funcName               = "PriceRuleList"
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

	if err = h.doRuleList(common.ActionDataTypeSubAccountPriceRules, &req, &apiResp); err != nil {
		log.Error("doPriceRuleList err:", err.Error(), funcName, clientIp)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doRuleList(actionDataType common.ActionDataType, req *ReqPriceRuleList, apiResp *api_code.ApiResp) error {
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	if err := h.checkForSearch(parentAccountId, apiResp); err != nil {
		return err
	}

	baseInfo, err := h.TxTool.GetBaseInfo()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "server error")
		return err
	}

	subAccountCell, err := h.getSubAccountCell(baseInfo.ContractSubAcc, parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
	}
	subAccountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccountCell.OutPoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return err
	}

	subAccountEntity := witness.NewSubAccountRuleEntity(req.Account)
	if err := subAccountEntity.ParseFromTx(subAccountTx.Transaction, actionDataType); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return err
	}

	for idx, v := range subAccountEntity.Rules {
		ruleType := tables.RuleTypePreservedRules
		if actionDataType == common.ActionDataTypeSubAccountPriceRules {
			ruleType = tables.RuleTypePriceRules
			subAccountEntity.Rules[idx].Price /= math.Pow10(6)
		}

		if v.Ast.Type == witness.Function &&
			v.Ast.Name == string(witness.FunctionInList) &&
			v.Ast.Arguments[0].Type == witness.Variable &&
			v.Ast.Arguments[0].Name == string(witness.Account) &&
			v.Ast.Arguments[1].Type == witness.Value &&
			v.Ast.Arguments[1].ValueType == witness.BinaryArray {

			accIdWhitelist := gconv.Strings(v.Ast.Arguments[1].Value)
			rules, err := h.DbDao.GetRulesBySubAccountIds(parentAccountId, ruleType, accIdWhitelist)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
				return err
			}
			if len(rules) != len(accIdWhitelist) {
				err := errors.New("data aberrant")
				apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
				return err
			}

			accWhitelist := make([]string, 0, len(accIdWhitelist))
			for _, v := range rules {
				accWhitelist = append(accWhitelist, v.Account)
			}
			subAccountEntity.Rules[idx].Ast.Arguments[1].Value = accWhitelist
		}
	}

	var resp RespPriceRuleList
	resp.List = subAccountEntity.Rules
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) checkForSearch(parentAccountId string, apiResp *api_code.ApiResp) error {
	acc, err := h.DbDao.GetAccountInfoByAccountId(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "search account err")
		return fmt.Errorf("SearchAccount err: %s", err.Error())
	}
	if acc.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account not exist")
		return fmt.Errorf("account not exist: %s", parentAccountId)
	}
	accountInfo, err := h.DbDao.GetAccountInfoByAccountId(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "search account err")
		return fmt.Errorf("SearchAccount err: %s", err.Error())
	}
	if accountInfo.EnableSubAccount != tables.AccountEnableStatusOn {
		err = errors.New("sub account no enable, please enable sub_account before use")
		apiResp.ApiRespErr(api_code.ApiCodeSubAccountNoEnable, err.Error())
		return err
	}
	return nil
}
