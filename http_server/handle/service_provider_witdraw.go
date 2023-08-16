package handle

import (
	"das_sub_account/config"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"das_sub_account/txtool"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/molecule"
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

type ReqServiceProviderWithdraw struct {
	ServiceProviderAddress string `json:"service_provider_address" binding:"required"`
}

type RespServiceProviderWithdraw struct {
	Hash   []string         `json:"hash"`
	Action common.DasAction `json:"action"`
}

func (h *HttpHandle) ServiceProviderWithdraw(ctx *gin.Context) {
	var (
		funcName               = "ServiceProviderWithdraw"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqServiceProviderWithdraw
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req))

	if err = h.doServiceProviderWithdraw(&req, &apiResp); err != nil {
		log.Error("doServiceProviderWithdraw err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doServiceProviderWithdraw(req *ReqServiceProviderWithdraw, apiResp *api_code.ApiResp) error {
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}
	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}

	hashList, err := h.buildServiceProviderWithdraw(req.ServiceProviderAddress)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return err
	}

	resp := &RespServiceProviderWithdraw{
		Action: common.DasActionCollectSubAccountChannelProfit,
		Hash:   hashList,
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) buildServiceProviderWithdraw(providerAddress string) (txHash []string, err error) {
	parseAddress, err := address.Parse(providerAddress)
	if err != nil {
		return nil, err
	}
	providerId := common.Bytes2Hex(parseAddress.Script.Args)

	latestExpenditure, err := h.DbDao.GetLatestSubAccountAutoMintStatementByType(providerId, tables.SubAccountAutoMintTxTypeExpenditure)
	if err != nil {
		return nil, err
	}
	list, err := h.DbDao.FindSubAccountAutoMintStatements(providerId, tables.SubAccountAutoMintTxTypeIncome, latestExpenditure.BlockNumber)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errors.New("no income balance can withdraw")
	}

	skipParentMap := make(map[string]bool)
	parentMap := make(map[string]decimal.Decimal)
	for _, v := range list {
		if skipParentMap[v.ParentAccountId] {
			continue
		}
		history, err := h.DbDao.GetLatestSubAccountAutoMintWithdrawHistory(providerId, v.ParentAccountId)
		if err != nil {
			return nil, err
		}
		if history.Id > 0 {
			statementInfo, err := h.DbDao.GetSubAccountAutoMintByTxHash(history.TxHash)
			if err != nil {
				return nil, err
			}
			if statementInfo.Id == 0 {
				skipParentMap[v.ParentAccountId] = true
				continue
			}
		}

		tx, err := h.DasCore.Client().GetTransaction(h.Ctx, types.HexToHash(v.TxHash))
		if err != nil {
			return nil, err
		}
		builder := witness.SubAccountNewBuilder{}
		dataBys := tx.Transaction.Witnesses[v.WitnessIndex][common.WitnessDasTableTypeEndIndex:]
		subAccountNew, err := builder.ConvertSubAccountNewFromBytes(dataBys)
		if err != nil {
			return nil, err
		}
		if common.Bytes2Hex(subAccountNew.EditValue[:20]) != providerId {
			err = fmt.Errorf("data error txHash: %s, provider: %s", v.TxHash, providerId)
			log.Error(err)
			return nil, err
		}

		price, err := molecule.Bytes2GoU64(subAccountNew.EditValue[20:])
		if err != nil {
			return nil, err
		}
		log.Infof("txHash: %s, provider: %s, price: %d", v.TxHash, providerId, price)

		subAccountCell := tx.Transaction.Outputs[0]
		parentAccountId := common.Bytes2Hex(subAccountCell.Type.Args)
		parentMap[parentAccountId] = parentMap[parentAccountId].Add(decimal.NewFromInt(int64(price)))
	}

	parentMapNew := make(map[string]decimal.Decimal)
	for k, v := range parentMap {
		if uint64(v.IntPart()) < 61*common.OneCkb {
			continue
		}
		parentMapNew[k] = v
	}

	if len(parentMapNew) == 0 {
		return nil, errors.New("no income balance can withdraw")
	}

	txParamsList := make([]*txbuilder.BuildTransactionParams, 0)
	for parentAccountId, price := range parentMapNew {
		txParams := &txbuilder.BuildTransactionParams{}
		// CellDeps
		contractSubAccount, err := core.GetDasContractInfo(common.DASContractNameSubAccountCellType)
		if err != nil {
			return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
		}
		contractBalanceCell, err := core.GetDasContractInfo(common.DasContractNameBalanceCellType)
		if err != nil {
			return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
		}
		configCellSubAcc, err := core.GetDasConfigCellInfo(common.ConfigCellTypeArgsSubAccount)
		if err != nil {
			return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
		}
		txParams.CellDeps = append(txParams.CellDeps,
			contractSubAccount.ToCellDep(),
			contractBalanceCell.ToCellDep(),
			configCellSubAcc.ToCellDep(),
		)

		// inputs cell
		subAccountCell, err := h.DasCore.GetSubAccountCell(parentAccountId)
		if err != nil {
			return nil, err
		}
		txParams.Inputs = append(txParams.Inputs, &types.CellInput{
			PreviousOutput: subAccountCell.OutPoint,
		})

		change, liveBalanceCell, err := h.TxTool.GetBalanceCell(&txtool.ParamBalance{
			DasLock:      h.TxTool.ServerScript,
			NeedCapacity: common.OneCkb,
		})
		if err != nil {
			return nil, err
		}
		for _, v := range liveBalanceCell {
			txParams.Inputs = append(txParams.Inputs, &types.CellInput{
				PreviousOutput: v.OutPoint,
			})
		}

		subAccountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccountCell.OutPoint.TxHash)
		if err != nil {
			return nil, err
		}

		// sub_account cell
		subAccountCellOutput := subAccountTx.Transaction.Outputs[subAccountCell.OutPoint.Index]
		txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
			Capacity: subAccountCellOutput.Capacity - uint64(price.IntPart()),
			Lock:     subAccountCellOutput.Lock,
			Type:     subAccountCellOutput.Type,
		})
		subAccountCellDetail := witness.ConvertSubAccountCellOutputData(subAccountCell.OutputData)
		subAccountCellDetail.DasProfit -= uint64(price.IntPart())
		txParams.OutputsData = append(txParams.OutputsData, witness.BuildSubAccountCellOutputData(subAccountCellDetail))

		// provider balance_cell
		txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
			Capacity: uint64(price.IntPart()),
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
			return nil, err
		}
		txParams.Witnesses = append(txParams.Witnesses, actionWitness)

		if err := witness.GetWitnessDataFromTx(subAccountTx.Transaction, func(actionDataType common.ActionDataType, dataBys []byte, index int) (bool, error) {
			if actionDataType == common.ActionDataTypeSubAccountPriceRules ||
				actionDataType == common.ActionDataTypeSubAccountPreservedRules {
				txParams.Witnesses = append(txParams.Witnesses, witness.GenDasDataWitnessWithByte(actionDataType, dataBys))
			}
			return true, nil
		}); err != nil {
			return nil, err
		}
		txParamsList = append(txParamsList, txParams)
	}

	txHashList := make([]string, 0, len(txParamsList))
	if err := h.DbDao.Transaction(func(tx *gorm.DB) error {
		txBuilders := make([]*txbuilder.DasTxBuilder, 0, len(txParamsList))
		for _, txParams := range txParamsList {
			txBuilder := txbuilder.NewDasTxBuilderFromBase(h.TxBuilderBase, nil)
			if err := txBuilder.BuildTransaction(txParams); err != nil {
				return fmt.Errorf("BuildTransaction err: %s", err.Error())
			}
			sizeInBlock, _ := txBuilder.Transaction.SizeInBlock()
			latestIndex := len(txBuilder.Transaction.Outputs) - 1
			changeCapacity := txBuilder.Transaction.Outputs[latestIndex].Capacity - sizeInBlock - 1000
			txBuilder.Transaction.Outputs[latestIndex].Capacity = changeCapacity
			txBuilders = append(txBuilders, txBuilder)

			hash, err := txBuilder.Transaction.ComputeHash()
			if err != nil {
				return err
			}
			hashStr := hash.Hex()

			txHashList = append(txHashList, hashStr)
			parentAccountId := common.Bytes2Hex(txBuilder.Transaction.Outputs[0].Type.Args)

			taskInfo := &tables.TableTaskInfo{
				TaskType:        tables.TaskTypeChain,
				ParentAccountId: parentAccountId,
				Action:          common.DasActionCollectSubAccountChannelProfit,
				Outpoint:        common.OutPoint2String(hashStr, 0),
				Timestamp:       time.Now().UnixNano() / 1e6,
				SmtStatus:       tables.SmtStatusWriteComplete,
				TxStatus:        tables.TxStatusPending,
				SvrName:         config.Cfg.Slb.SvrName,
			}
			taskInfo.InitTaskId()

			if err := tx.Create(taskInfo).Error; err != nil {
				return err
			}

			for i := 1; i < len(txParams.Outputs)-1; i++ {
				if err := tx.Create(&tables.TableSubAccountAutoMintWithdrawHistory{
					TaskId:            taskInfo.TaskId,
					ParentAccountId:   parentAccountId,
					ServiceProviderId: common.Bytes2Hex(txBuilder.Transaction.Outputs[i].Lock.Args),
					TxHash:            hashStr,
					Price:             decimal.NewFromInt(int64(txBuilder.Transaction.Outputs[i].Capacity)),
				}).Error; err != nil {
					return err
				}
			}
		}

		for _, v := range txBuilders {
			hash, err := v.SendTransaction()
			if err != nil {
				return err
			}
			log.Infof("SendTransaction hash: %s", hash.Hex())
		}
		return nil
	}); err != nil {
		log.Error("CreateTask err: ", err.Error())
		return nil, err
	}
	return txHashList, nil
}
