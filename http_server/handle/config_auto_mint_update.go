package handle

import (
	"das_sub_account/config"
	"das_sub_account/internal"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqConfigAutoMintUpdate struct {
	core.ChainTypeAddress
	Account string `json:"account" binding:"required"`
	Enable  bool   `json:"enable"`
}

type RespConfigAutoMintUpdate struct {
	SignInfoList
}

func (h *HttpHandle) ConfigAutoMintUpdate(ctx *gin.Context) {
	var (
		funcName               = "ConfigAutoMintUpdate"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqConfigAutoMintUpdate
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx)

	if err = h.doConfigAutoMintUpdate(&req, &apiResp); err != nil {
		log.Error("doConfigAutoMintUpdate err:", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doConfigAutoMintUpdate(req *ReqConfigAutoMintUpdate, apiResp *api_code.ApiResp) error {
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

	enable := witness.AutoDistributionDefault
	if req.Enable {
		enable = witness.AutoDistributionEnable
	}

	txParams, _, err := h.rulesTxAssemble(RulesTxAssembleParams{
		Req: &ReqPriceRuleUpdate{
			ChainTypeAddress: req.ChainTypeAddress,
			Account:          req.Account,
		},
		ApiResp:          apiResp,
		AutoDistribution: enable,
	})
	if err != nil {
		return err
	}

	signList, _, err := h.buildTx(&paramBuildTx{
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
	resp.SignKey = signList.SignKey
	resp.List = signList.List
	resp.SignList = signList.List[0].SignList
	log.Info("doConfigAutoMintUpdate:", toolib.JsonString(resp))

	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) getSubAccountCell(contract *core.DasContractInfo, parentAccountId string) (*indexer.LiveCell, error) {
	searchKey := indexer.SearchKey{
		Script:     contract.ToScript(common.Hex2Bytes(parentAccountId)),
		ScriptType: indexer.ScriptTypeType,
		ArgsLen:    0,
		Filter:     nil,
	}
	liveCell, err := h.DasCore.Client().GetCells(h.Ctx, &searchKey, indexer.SearchOrderDesc, 1, "")
	if err != nil {
		return nil, fmt.Errorf("GetCells err: %s", err.Error())
	}
	if subLen := len(liveCell.Objects); subLen != 1 {
		return nil, fmt.Errorf("sub account outpoint len: %d", subLen)
	}
	return liveCell.Objects[0], nil
}
