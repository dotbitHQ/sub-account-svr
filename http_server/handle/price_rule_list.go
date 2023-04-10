package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/scorpiotzh/toolib"
	"math"
	"net/http"
)

type ReqPriceRuleList struct {
	core.ChainTypeAddress
	Account string `json:"account" binding:"required"`
}

func (h *HttpHandle) PriceRuleList(ctx *gin.Context) {
	var (
		funcName = "PriceRuleList"
		clientIp = GetClientIp(ctx)
		req      ReqPriceRuleList
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

	if err = h.doRuleList(common.ActionDataTypeSubAccountPriceRules, &req, &apiResp); err != nil {
		log.Error("doPriceRuleList err:", err.Error(), funcName, clientIp)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doRuleList(actionDataType common.ActionDataType, req *ReqPriceRuleList, apiResp *api_code.ApiResp) error {
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}
	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}
	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return err
	}
	address := common.FormatAddressPayload(res.AddressPayload, res.DasAlgorithmId)

	if err := h.checkAuth(address, req.Account, apiResp); err != nil {
		return err
	}
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))

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

	resRuleList := make(Rules, 0)

	for idx, v := range subAccountEntity.Rules {
		rule := Rule{
			Name:       v.Name,
			Note:       v.Note,
			Price:      float64(v.Price) / math.Pow10(6),
			Whitelist:  []string{},
			Conditions: []Condition{},
		}

		if len(v.Ast.Expressions) != 2 {
			err := errors.New("rule expressions length error")
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return err
		}
		if v.Ast.Expressions[0].Type != witness.Variable || v.Ast.Expressions[1].Type != witness.Value {
			err := errors.New("rule expressions struct error")
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return err
		}

		funcName := witness.FunctionType(v.Ast.Name)
		if funcName == witness.FunctionInList {
			// whitelist process
			rule.Type = RuleTypeWhitelist

			if len(v.Ast.Expressions) != 2 {
				err := errors.New("rule expressions length error")
				apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
				return err
			}
			if v.Ast.Expressions[0].Type != witness.Variable || v.Ast.Expressions[1].Type != witness.Value {
				err := errors.New("rule expressions struct error")
				apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
				return err
			}

			subAccountIds := gconv.Strings(v.Ast.Expressions[1].Value)
			whitelist, err := h.DbDao.FindRulesBySubAccountIds(subAccountCell.OutPoint.TxHash.Hex(), parentAccountId, tables.RuleTypePriceRules, idx)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
				return err
			}
			if len(whitelist) != len(subAccountIds) {
				err = errors.New("rule data error")
				apiResp.ApiRespErr(api_code.ApiCodeRuleDataErr, err.Error())
				return err
			}

			whitelistMap := make(map[string]tables.RuleWhitelist)
			for _, v := range whitelist {
				whitelistMap[v.AccountId] = v
			}
			for _, v := range subAccountIds {
				subAcc, ok := whitelistMap[v]
				if !ok {
					err = errors.New("rule data error")
					apiResp.ApiRespErr(api_code.ApiCodeRuleDataErr, err.Error())
					return err
				}
				rule.Whitelist = append(rule.Whitelist, subAcc.Account)
			}
			continue
		}

		rule.Type = RuleTypeConditions
		switch v.Ast.Type {
		case witness.Function:
			cond := Condition{
				VarName: witness.VariableName(v.Ast.Expressions[0].Name),
				Op:      v.Ast.Name,
			}
			switch v.Ast.Name {
			case string(witness.FunctionIncludeCharts):
				cond.Value = gconv.Strings(v.Ast.Expressions[1].Value)
			case string(witness.FunctionOnlyIncludeCharset):
				cond.Value = gconv.String(v.Ast.Expressions[1].Value)
			}
			rule.Conditions = append(rule.Conditions, cond)
		case witness.Operator:
			cond := Condition{
				VarName: witness.VariableName(v.Ast.Expressions[0].Name),
				Op:      string(v.Ast.Symbol),
				Value:   gconv.Uint8(v.Ast.Expressions[1].Value),
			}
			rule.Conditions = append(rule.Conditions, cond)
		}
		resRuleList = append(resRuleList, rule)
	}

	apiResp.ApiRespOK(struct {
		List Rules `json:"list"`
	}{
		List: resRuleList,
	})
	return nil
}
