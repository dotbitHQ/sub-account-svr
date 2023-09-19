package handle

import (
	"das_sub_account/config"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"das_sub_account/txtool"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type ReqServiceProviderWithdraw2 struct {
	ServiceProviderAddress string `json:"service_provider_address" binding:"required"`
	Account                string `json:"account" binding:"required"`
	Withdraw               bool   `json:"withdraw"`
}

type RespServiceProviderWithdraw2 struct {
	Hash string `json:"hash"`
}

func (h *HttpHandle) ServiceProviderWithdraw2(ctx *gin.Context) {
	var (
		funcName               = "ServiceProviderWithdraw2"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqServiceProviderWithdraw2
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req))

	//time.Sleep(time.Minute * 3)
	if err = h.doServiceProviderWithdraw2(&req, &apiResp); err != nil {
		log.Error("doServiceProviderWithdraw2 err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doServiceProviderWithdraw2(req *ReqServiceProviderWithdraw2, apiResp *api_code.ApiResp) error {
	var resp RespServiceProviderWithdraw2

	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}
	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}

	//
	if hash, err := h.buildServiceProviderWithdraw2Tx(req); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("buildServiceProviderWithdraw2Tx err: %s", err.Error())
	} else {
		resp.Hash = hash
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) buildServiceProviderWithdraw2Tx(req *ReqServiceProviderWithdraw2) (string, error) {
	spAddr, err := address.Parse(req.ServiceProviderAddress)
	if err != nil {
		return "", fmt.Errorf("address.Parse err: %s", err.Error())
	}
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	providerId := common.Bytes2Hex(spAddr.Script.Args)

	latestExpenditure, err := h.DbDao.GetLatestSubAccountAutoMintStatementByType2(providerId, parentAccountId, tables.SubAccountAutoMintTxTypeExpenditure)
	if err != nil {
		return "", fmt.Errorf("GetLatestSubAccountAutoMintStatementByType2 err: %s", err.Error())
	}

	list, err := h.DbDao.FindSubAccountAutoMintStatements2(providerId, parentAccountId, tables.SubAccountAutoMintTxTypeIncome, latestExpenditure.BlockNumber)
	if err != nil {
		return "", fmt.Errorf("FindSubAccountAutoMintStatements err: %s", err.Error())
	}
	if len(list) == 0 {
		return "", nil
	}
	// testnet 2023-09-15
	minPrice := uint64(990000)
	amount := decimal.Zero
	for _, v := range list {
		log.Info(v.ServiceProviderId, v.Quote, v.Years, v.Price)
		if v.Quote.Cmp(decimal.Zero) == 0 || v.Years == 0 {
			amount = amount.Add(v.Price)
			continue
		}
		minCKB := decimal.NewFromInt(int64(config.PriceToCKB(minPrice, uint64(v.Quote.IntPart()), v.Years)))
		backCKB := v.Price.Sub(minCKB)
		if backCKB.Cmp(decimal.Zero) == 1 {
			amount = amount.Add(backCKB)
		}
	}
	log.Info("ServiceProviderWithdraw2:", req.ServiceProviderAddress, req.Account, amount.String())
	if amount.Cmp(decimal.NewFromInt(int64(common.MinCellOccupiedCkb))) < 1 {
		return "", nil
	}
	if !req.Withdraw {
		return "", nil
	}

	// build tx ==================

	txParams := &txbuilder.BuildTransactionParams{}
	// CellDeps
	contractSubAccount, err := core.GetDasContractInfo(common.DASContractNameSubAccountCellType)
	if err != nil {
		return "", fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	contractBalanceCell, err := core.GetDasContractInfo(common.DasContractNameBalanceCellType)
	if err != nil {
		return "", fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	configCellSubAcc, err := core.GetDasConfigCellInfo(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		return "", fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	txParams.CellDeps = append(txParams.CellDeps,
		contractSubAccount.ToCellDep(),
		contractBalanceCell.ToCellDep(),
		configCellSubAcc.ToCellDep(),
	)

	// inputs cell
	subAccountCell, err := h.DasCore.GetSubAccountCell(parentAccountId)
	if err != nil {
		return "", fmt.Errorf("GetSubAccountCell err: %s", err.Error())
	}
	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: subAccountCell.OutPoint,
	})

	change, liveBalanceCell, err := h.TxTool.GetBalanceCell(&txtool.ParamBalance{
		DasLock:      h.TxTool.ServerScript,
		NeedCapacity: common.OneCkb,
	})
	if err != nil {
		return "", fmt.Errorf("GetBalanceCell err: %s", err.Error())
	}
	for _, v := range liveBalanceCell {
		txParams.Inputs = append(txParams.Inputs, &types.CellInput{
			PreviousOutput: v.OutPoint,
		})
	}

	// sub_account cell
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: subAccountCell.Output.Capacity - uint64(amount.IntPart()),
		Lock:     subAccountCell.Output.Lock,
		Type:     subAccountCell.Output.Type,
	})
	subAccountCellDetail := witness.ConvertSubAccountCellOutputData(subAccountCell.OutputData)
	subAccountCellDetail.DasProfit -= uint64(amount.IntPart())
	txParams.OutputsData = append(txParams.OutputsData, witness.BuildSubAccountCellOutputData(subAccountCellDetail))

	// provider balance_cell
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: uint64(amount.IntPart()),
		Lock:     common.GetNormalLockScript(providerId),
	})
	txParams.OutputsData = append(txParams.OutputsData, []byte{})

	// change balance_cell
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: change + common.OneCkb,
		Lock:     h.ServerScript,
	})
	txParams.OutputsData = append(txParams.OutputsData, []byte{})

	actionWitness, err := witness.GenActionDataWitness(common.DasActionCollectSubAccountChannelProfit, nil)
	if err != nil {
		return "", fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	//
	txBuilder := txbuilder.NewDasTxBuilderFromBase(h.TxBuilderBase, nil)
	if err := txBuilder.BuildTransaction(txParams); err != nil {
		return "", fmt.Errorf("BuildTransaction err: %s", err.Error())
	}
	sizeInBlock, _ := txBuilder.Transaction.SizeInBlock()
	latestIndex := len(txBuilder.Transaction.Outputs) - 1
	changeCapacity := txBuilder.Transaction.Outputs[latestIndex].Capacity - sizeInBlock - 1000
	txBuilder.Transaction.Outputs[latestIndex].Capacity = changeCapacity

	hash, err := txBuilder.Transaction.ComputeHash()
	if err != nil {
		return "", fmt.Errorf("ComputeHash err: %s", err.Error())
	}
	log.Info("buildServiceProviderWithdraw2Tx:", req.ServiceProviderAddress, req.Account, hash.Hex())
	taskInfo := &tables.TableTaskInfo{
		TaskType:        tables.TaskTypeChain,
		ParentAccountId: parentAccountId,
		Action:          common.DasActionCollectSubAccountChannelProfit,
		Outpoint:        common.OutPoint2String(hash.Hex(), 0),
		Timestamp:       time.Now().UnixMilli(),
		SmtStatus:       tables.SmtStatusWriteComplete,
		TxStatus:        tables.TxStatusPending,
	}
	taskInfo.InitTaskId()
	if _, err := txBuilder.SendTransaction(); err != nil {
		return "", fmt.Errorf("SendTransaction err: %s", err.Error())
	}

	if err := h.DbDao.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(taskInfo).Error; err != nil {
			return err
		}
		if err := tx.Create(&tables.TableSubAccountAutoMintWithdrawHistory{
			TaskId:            taskInfo.TaskId,
			ParentAccountId:   parentAccountId,
			ServiceProviderId: providerId,
			TxHash:            hash.Hex(),
			Price:             amount,
		}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return "", fmt.Errorf("Transaction err: %s", err.Error())
	}

	return hash.Hex(), nil
}
