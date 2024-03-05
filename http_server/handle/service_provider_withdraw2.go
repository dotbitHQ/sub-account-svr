package handle

import (
	"das_sub_account/config"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
	"time"
)

type ReqServiceProviderWithdraw2 struct {
	ServiceProviderAddress string `json:"service_provider_address" binding:"required"`
	Account                string `json:"account" binding:"required"`
	Withdraw               bool   `json:"withdraw"`
}

type RespServiceProviderWithdraw2 struct {
	Hash   string          `json:"hash"`
	Amount decimal.Decimal `json:"amount"`
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

	if err := h.buildServiceProviderWithdraw2Tx(req, &resp); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("buildServiceProviderWithdraw2Tx err: %s", err.Error())
	}
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) buildServiceProviderWithdraw2Tx(req *ReqServiceProviderWithdraw2, resp *RespServiceProviderWithdraw2) error {
	spAddr, err := address.Parse(req.ServiceProviderAddress)
	if err != nil {
		return fmt.Errorf("address.Parse err: %s", err.Error())
	}
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	providerId := common.Bytes2Hex(spAddr.Script.Args)

	// check pending tx
	task, err := h.DbDao.GetPendingTaskByParentIdAndActionAndTxStatus(parentAccountId, common.DasActionCollectSubAccountChannelProfit, tables.TxStatusPending)
	if err != nil {
		return err
	}
	if task.Id > 0 {
		return fmt.Errorf("have pending task: %s", task.TaskId)
	}

	task, err = h.DbDao.GetPendingTaskByParentIdAndActionAndTxStatus(parentAccountId, common.DasActionCollectSubAccountChannelProfit, tables.TxStatusCommitted)
	if err != nil {
		return err
	}
	if task.Id > 0 {
		txHash, _ := common.String2OutPoint(task.Outpoint)
		statement, err := h.DbDao.GetSubAccountAutoMintByTxHash(txHash)
		if err != nil {
			return err
		}
		if statement.Id == 0 {
			return fmt.Errorf("have pending task: %s", task.TaskId)
		}
	}

	latestExpenditure, err := h.DbDao.GetLatestSubAccountAutoMintStatementByType2(providerId, parentAccountId, tables.SubAccountAutoMintTxTypeExpenditure)
	if err != nil {
		return fmt.Errorf("GetLatestSubAccountAutoMintStatementByType2 err: %s", err.Error())
	}

	list, err := h.DbDao.FindSubAccountAutoMintStatements2(providerId, parentAccountId, tables.SubAccountAutoMintTxTypeIncome, latestExpenditure.BlockNumber)
	if err != nil {
		return fmt.Errorf("FindSubAccountAutoMintStatements err: %s", err.Error())
	}
	if len(list) == 0 {
		return nil
	}

	minPrice, err := decimal.NewFromString(config.Cfg.Das.AutoMint.MinPrice)
	if err != nil {
		return err
	}
	platformFeeRatio, err := decimal.NewFromString(config.Cfg.Das.AutoMint.PlatformFeeRatio)
	if err != nil {
		return err
	}
	serviceFeeRate, err := decimal.NewFromString(config.Cfg.Das.AutoMint.ServiceFeeRatio)
	if err != nil {
		return err
	}

	minPriceFee := minPrice.
		Add(decimal.NewFromFloat(config.Cfg.Das.AutoMint.ServiceFeeMin)).
		Div(platformFeeRatio.Add(serviceFeeRate)).
		Mul(decimal.New(1, 6))
	feeRate := decimal.NewFromInt(1).Sub(platformFeeRatio)

	amount := decimal.Zero
	for _, statement := range list {
		tx, err := h.DasCore.Client().GetTransaction(h.Ctx, types.HexToHash(statement.TxHash))
		if err != nil {
			return err
		}
		builder := witness.SubAccountNewBuilder{}
		dataBys := tx.Transaction.Witnesses[statement.WitnessIndex][common.WitnessDasTableTypeEndIndex:]
		subAccountNew, err := builder.ConvertSubAccountNewFromBytes(dataBys)
		if err != nil {
			return err
		}
		subAccId := subAccountNew.CurrentSubAccountData.AccountId

		outpoint := common.OutPoint2String(statement.TxHash, 0)
		task, err := h.DbDao.GetTaskByOutpointWithParentAccountId(statement.ParentAccountId, outpoint)
		if err != nil {
			return fmt.Errorf("GetTaskByOutpointWithParentAccountId err: %s", err.Error())
		}

		var orderInfo tables.OrderInfo
		if task.Id > 0 {
			smtRecord, err := h.DbDao.GetChainSmtRecordListByTaskIdAndAccId(task.TaskId, subAccId)
			if err != nil {
				return fmt.Errorf("GetChainSmtRecordListByTaskIdAndAccId err: %s", err.Error())
			}
			if smtRecord.Id == 0 {
				return fmt.Errorf("GetChainSmtRecordListByTaskIdAndAccId err: subAccount not found, task_id=%s, sub_acc=%s", task.TaskId, subAccountNew.CurrentSubAccountData.Account())
			}
			if smtRecord.OrderID == "" {
				return fmt.Errorf("GetChainSmtRecordListByTaskIdAndAccId err: order_id is empty, task_id=%s, sub_acc=%s", task.TaskId, subAccountNew.CurrentSubAccountData.Account())
			}
			orderInfo, err = h.DbDao.GetOrderByOrderID(smtRecord.OrderID)
			if err != nil {
				return fmt.Errorf("GetOrderByOrderID err: %s", err.Error())
			}
			if orderInfo.Id == 0 {
				return fmt.Errorf("GetOrderByOrderID err: order not found, order_id=%s", smtRecord.OrderID)
			}
		}

		minCkbFee := decimal.NewFromInt(int64(config.PriceToCKB(uint64(minPriceFee.IntPart()), uint64(statement.Quote.IntPart()), statement.Years)))
		minCKB := decimal.NewFromInt(int64(config.PriceToCKB(uint64(minPrice.Mul(decimal.New(1, 6)).IntPart()), uint64(statement.Quote.IntPart()), statement.Years)))
		if (task.Id == 0 || orderInfo.CouponCode == "") && statement.Price.GreaterThan(minCkbFee) {
			// other provider or self but not use coupon
			amount = amount.Add(statement.Price.Mul(feeRate))
		} else if statement.Price.Sub(minCKB).GreaterThan(decimal.Zero) {
			amount = amount.Add(statement.Price.Sub(minCKB))
		}
	}
	log.Info("ServiceProviderWithdraw2:", req.ServiceProviderAddress, req.Account, amount.String())

	minCellOccupiedCkb := decimal.NewFromInt(int64(common.MinCellOccupiedCkb))
	if amount.LessThanOrEqual(minCellOccupiedCkb) {
		return fmt.Errorf("transfer ckb less than %s", minCellOccupiedCkb)
	}
	resp.Amount = amount

	if !req.Withdraw {
		return nil
	}

	// build tx ==================

	txParams := &txbuilder.BuildTransactionParams{}
	// CellDeps
	contractSubAccount, err := core.GetDasContractInfo(common.DASContractNameSubAccountCellType)
	if err != nil {
		return fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	contractBalanceCell, err := core.GetDasContractInfo(common.DasContractNameBalanceCellType)
	if err != nil {
		return fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	configCellSubAcc, err := core.GetDasConfigCellInfo(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		return fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	txParams.CellDeps = append(txParams.CellDeps,
		contractSubAccount.ToCellDep(),
		contractBalanceCell.ToCellDep(),
		configCellSubAcc.ToCellDep(),
	)

	// inputs cell
	subAccountCell, err := h.DasCore.GetSubAccountCell(parentAccountId)
	if err != nil {
		return fmt.Errorf("GetSubAccountCell err: %s", err.Error())
	}
	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: subAccountCell.OutPoint,
	})

	//change, liveBalanceCell, err := h.TxTool.GetBalanceCell(&txtool.ParamBalance{
	//	DasLock:      h.TxTool.ServerScript,
	//	NeedCapacity: common.OneCkb,
	//})
	//if err != nil {
	//	return fmt.Errorf("GetBalanceCell err: %s", err.Error())
	//}

	change, liveBalanceCell, err := h.DasCore.GetBalanceCellWithLock(&core.ParamGetBalanceCells{
		LockScript:        h.ServerScript,
		CapacityNeed:      common.OneCkb,
		DasCache:          h.DasCache,
		CapacityForChange: common.DasLockWithBalanceTypeMinCkbCapacity,
		SearchOrder:       indexer.SearchOrderAsc,
	})
	if err != nil {
		return fmt.Errorf("GetBalanceCellWithLock err %s", err.Error())
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
		return fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	txBuilder := txbuilder.NewDasTxBuilderFromBase(h.TxBuilderBase, nil)
	if err := txBuilder.BuildTransaction(txParams); err != nil {
		return fmt.Errorf("BuildTransaction err: %s", err.Error())
	}
	sizeInBlock, _ := txBuilder.Transaction.SizeInBlock()
	latestIndex := len(txBuilder.Transaction.Outputs) - 1
	changeCapacity := txBuilder.Transaction.Outputs[latestIndex].Capacity - sizeInBlock - 1000
	txBuilder.Transaction.Outputs[latestIndex].Capacity = changeCapacity

	hash, err := txBuilder.SendTransaction()
	if err != nil {
		return fmt.Errorf("SendTransaction err: %s", err.Error())
	}
	resp.Hash = hash.Hex()

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
	if err := h.DbDao.CreateTask(taskInfo); err != nil {
		log.Error("CreateTask err: ", err.Error(), hash.Hex())
		return err
	}
	return nil
}
