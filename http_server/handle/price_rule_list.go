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
	"net/http"
)

type ReqPriceRuleList struct {
	core.ChainTypeAddress
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

	if err := h.check(address, req.Account, apiResp); err != nil {
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

	for idx, v := range subAccountEntity.Rules {
		if v.Ast.Type == witness.Function &&
			v.Ast.Name == string(witness.FunctionInList) &&
			v.Ast.Expressions[0].Type == witness.Variable &&
			v.Ast.Expressions[0].Name == string(witness.Account) &&
			v.Ast.Expressions[1].Type == witness.Value &&
			v.Ast.Expressions[1].ValueType == witness.BinaryArray {

			accIdWhitelist := gconv.Strings(v.Ast.Expressions[1].Value)
			accWhitelist := make([]string, 0, len(accIdWhitelist))
			for _, v := range accIdWhitelist {
				rule, err := h.DbDao.GetRulesBySubAccountId(parentAccountId, tables.RuleTypePriceRules, v)
				if err != nil {
					apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
					return err
				}
				if rule.Id == 0 {
					err := errors.New("data aberrant")
					apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
					return err
				}
				accWhitelist = append(accWhitelist, rule.Account)
			}
			subAccountEntity.Rules[idx].Ast.Expressions[1].Value = accWhitelist
		}
	}

	var resp RespPriceRuleList
	resp.List = subAccountEntity.Rules
	apiResp.ApiRespOK(resp)
	return nil
}
