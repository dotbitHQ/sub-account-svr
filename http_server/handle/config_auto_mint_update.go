package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
)

type ReqConfigAutoMintUpdate struct {
	core.ChainTypeAddress
	Account string `json:"account" binding:"required"`
	Enable  bool   `json:"enable"`
}

type RespConfigAutoMintUpdate struct {
	SignInfo
}

func (h *HttpHandle) ConfigAutoMintUpdate(ctx *gin.Context) {
	var (
		funcName = "ConfigAutoMintUpdate"
		clientIp = GetClientIp(ctx)
		req      ReqConfigAutoMintUpdate
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

	if err = h.doConfigAutoMintUpdate(&req, &apiResp); err != nil {
		log.Error("doConfigAutoMintUpdate err:", err.Error(), funcName, clientIp)
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
	address := res.AddressHex
	if strings.HasPrefix(res.AddressHex, common.HexPreFix) {
		address = strings.ToLower(res.AddressHex)
	}

	if err := h.checkAuth(address, req.Account, apiResp); err != nil {
		return err
	}
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))

	baseInfo, err := h.TxTool.GetBaseInfo()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "server error")
		return err
	}

	accountCell, err := h.getAccountOrSubAccountCell(baseInfo.ContractAcc, parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
	}
	accountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, accountCell.OutPoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return err
	}
	subAccountCell, err := h.getAccountOrSubAccountCell(baseInfo.ContractAcc, parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
	}
	subAccountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccountCell.OutPoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return err
	}

	txParams := &txbuilder.BuildTransactionParams{}
	txParams.CellDeps = append(txParams.CellDeps,
		baseInfo.ContractAcc.ToCellDep(),
		baseInfo.ConfigCellSubAcc.ToCellDep(),
		baseInfo.TimeCell.ToCellDep(),
		baseInfo.HeightCell.ToCellDep(),
		baseInfo.ConfigCellAcc.ToCellDep(),
		baseInfo.ConfigCellSubAcc.ToCellDep(),
	)

	txParams.Inputs = append(txParams.Inputs,
		&types.CellInput{
			PreviousOutput: accountCell.OutPoint,
		},
		&types.CellInput{
			PreviousOutput: subAccountCell.OutPoint,
		},
	)

	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: accountTx.Transaction.Outputs[accountCell.TxIndex].Capacity,
		Lock:     accountTx.Transaction.Outputs[accountCell.TxIndex].Lock,
		Type:     accountTx.Transaction.Outputs[accountCell.TxIndex].Type,
	})
	txParams.OutputsData = append(txParams.OutputsData, accountTx.Transaction.OutputsData[accountCell.TxIndex])

	dasLock, _, err := h.DasCore.Daf().HexToScript(*res)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "HexToArgs err")
		return err
	}

	liveCells, _, err := h.DasCore.GetBalanceCells(&core.ParamGetBalanceCells{
		DasCache:          h.DasCache,
		LockScript:        dasLock,
		CapacityNeed:      feeCapacity,
		CapacityForChange: common.DasLockWithBalanceTypeOccupiedCkb,
		SearchOrder:       indexer.SearchOrderDesc,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return fmt.Errorf("GetBalanceCells err: %s", err.Error())
	}

	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: p.subAccountCapacity,
		Lock:     contractAlwaysSuccess.ToScript(nil),
		Type:     contractSubAccount.ToScript(common.Hex2Bytes(p.acc.AccountId)),
	})
	subDataDetail := witness.SubAccountCellDataDetail{
		Action:           common.DasActionEnableSubAccount,
		SmtRoot:          smt.H256Zero(),
		DasProfit:        0,
		OwnerProfit:      0,
		CustomScriptArgs: nil,
	}
	subAccountOutputData := witness.BuildSubAccountCellOutputData(subDataDetail)

	subAccountCellData := witness.ConvertSubAccountCellOutputData(subAccountTx.Transaction.OutputsData[subAccountCell.TxIndex])
	if req.Enable {
		subAccountCellData.CustomScriptConfig[0] = 255
	} else {
		subAccountCellData.CustomScriptConfig[0] = 0
	}

	newSubAccountCellOutputData := witness.BuildSubAccountCellOutputData(subAccountCellData)
	txParams.OutputsData = append(txParams.OutputsData, newSubAccountCellOutputData)
}

func (h *HttpHandle) getAccountOrSubAccountCell(contract *core.DasContractInfo, parentAccountId string) (*indexer.LiveCell, error) {
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
