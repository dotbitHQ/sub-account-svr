package handle

import (
	"das_sub_account/config"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"gorm.io/gorm"
	"net/http"
)

func (h *HttpHandle) PreservedRuleUpdate(ctx *gin.Context) {
	var (
		funcName               = "PreservedRuleUpdate"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqPriceRuleUpdate
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

	if err = h.doPreservedRuleUpdate(&req, &apiResp); err != nil {
		log.Error("doConfigAutoMintUpdate err:", err.Error(), funcName, clientIp)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doPreservedRuleUpdate(req *ReqPriceRuleUpdate, apiResp *api_code.ApiResp) error {
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

	action := common.DasActionConfigSubAccount
	if err := h.check(address, req.Account, action, apiResp); err != nil {
		return err
	}
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))

	txParams, whiteListMap, err := h.rulesTxAssemble(RulesTxAssembleParams{
		Req:                 req,
		ApiResp:             apiResp,
		InputActionDataType: common.ActionDataTypeSubAccountPreservedRules,
	})
	if err != nil {
		return err
	}

	signKey, signList, txHash, err := h.buildTx(&paramBuildTx{
		txParams:  txParams,
		chainType: res.ChainType,
		address:   res.AddressHex,
		action:    action,
		account:   req.Account,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "buildTx err: "+err.Error())
		return fmt.Errorf("buildTx err: %s", err.Error())
	}

	resp := RespConfigAutoMintUpdate{}
	resp.Action = action
	resp.SignKey = signKey
	resp.List = append(resp.List, SignInfo{
		SignList: signList,
	})
	log.Info("doPreservedRuleUpdate:", toolib.JsonString(resp))

	if err := h.DbDao.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tx_hash=? and tx_status=?", txHash, tables.TxStatusPending).
			Delete(&tables.RuleWhitelist{}).Error; err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		for accountId, whiteList := range whiteListMap {
			if err := tx.Create(&tables.RuleWhitelist{
				TxHash:          txHash,
				ParentAccount:   req.Account,
				ParentAccountId: parentAccountId,
				RuleType:        tables.RuleTypePreservedRules,
				RuleIndex:       whiteList.Index,
				Account:         whiteList.Account,
				AccountId:       accountId,
				TxStatus:        tables.TxStatusPending,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	apiResp.ApiRespOK(resp)
	return nil
}
