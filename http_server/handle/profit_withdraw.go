package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
	"time"
)

type ReqProfitWithdraw struct {
	core.ChainTypeAddress
	Account          string `json:"account"`
	IsWithdrawDotBit bool   `json:"is_withdraw_dot_bit"`
}

type RespProfitWithdraw struct {
	Hash   string           `json:"hash"`
	Action common.DasAction `json:"action"`
}

func (h *HttpHandle) ProfitWithdraw(ctx *gin.Context) {
	var (
		funcName               = "ProfitWithdraw"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqProfitWithdraw
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

	if err = h.doProfitWithdraw(&req, &apiResp); err != nil {
		log.Error("doProfitWithdraw err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doProfitWithdraw(req *ReqProfitWithdraw, apiResp *api_code.ApiResp) error {
	var resp RespProfitWithdraw

	hexAddress, err := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
		return nil
	}
	if req.Account == "" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "account is nil")
		return nil
	}
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}
	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}

	// check account
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "GetAccountInfoByAccountId err")
		return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
	} else if acc.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account not exist")
		return nil
	} else if acc.EnableSubAccount != tables.AccountEnableStatusOn {
		apiResp.ApiRespErr(api_code.ApiCodeEnableSubAccountIsOff, "sub-account not enabled")
		return nil
	}

	// check sub-account-cell
	subAccountLiveCell, err := h.DasCore.GetSubAccountCell(acc.AccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("GetSubAccountCell err: %s", err.Error())
	}
	subDataDetail := witness.ConvertSubAccountCellOutputData(subAccountLiveCell.OutputData)
	if req.IsWithdrawDotBit {
		if subDataDetail.DasProfit < common.MinCellOccupiedCkb {
			apiResp.ApiRespErr(api_code.ApiCodeProfitNotEnough, "insufficient earnings to withdraw")
			return nil
		}
	} else {
		if acc.OwnerChainType != hexAddress.ChainType || !strings.EqualFold(acc.Owner, hexAddress.AddressHex) {
			apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "owner permission required")
			return nil
		}
		if subDataDetail.OwnerProfit < common.DasLockWithBalanceTypeOccupiedCkb {
			apiResp.ApiRespErr(api_code.ApiCodeProfitNotEnough, "insufficient earnings to withdraw")
			return nil
		}
	}

	// build tx
	txParams, err := h.buildProfitWithdrawTx(&paramProfitWithdrawTx{
		acc:                &acc,
		subAccountLiveCell: subAccountLiveCell,
		isWithdrawDotBit:   req.IsWithdrawDotBit,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "buildProfitWithdrawTx err: "+err.Error())
		return fmt.Errorf("buildProfitWithdrawTx err: %s", err.Error())
	}
	// check fee
	txBuilder := txbuilder.NewDasTxBuilderFromBase(h.TxBuilderBase, nil)
	if err := txBuilder.BuildTransaction(txParams); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "BuildTransaction err: "+err.Error())
		return fmt.Errorf("BuildTransaction err: %s", err.Error())
	}
	sizeInBlock, _ := txBuilder.Transaction.SizeInBlock()
	changeCapacity := txBuilder.Transaction.Outputs[0].Capacity - sizeInBlock - 1000
	txBuilder.Transaction.Outputs[0].Capacity = changeCapacity
	//
	hash, err := txBuilder.SendTransaction()
	if err != nil {
		return doSendTransactionError(err, apiResp)
	}
	resp.Hash = hash.String()
	resp.Action = common.DasActionCollectSubAccountProfit

	taskInfo := tables.TableTaskInfo{
		Id:              0,
		TaskId:          "",
		TaskType:        tables.TaskTypeNormal,
		ParentAccountId: accountId,
		Action:          common.DasActionCollectSubAccountProfit,
		RefOutpoint:     "",
		BlockNumber:     0,
		Outpoint:        common.OutPoint2String(hash.Hex(), 0),
		Timestamp:       time.Now().UnixNano() / 1e6,
		SmtStatus:       tables.SmtStatusWriteComplete,
		TxStatus:        tables.TxStatusPending,
	}
	taskInfo.InitTaskId()
	if err := h.DbDao.CreateTask(&taskInfo); err != nil {
		log.Error("CreateTask err: ", err.Error())
	}

	apiResp.ApiRespOK(resp)
	return nil
}

type paramProfitWithdrawTx struct {
	acc                *tables.TableAccountInfo
	subAccountLiveCell *indexer.LiveCell
	isWithdrawDotBit   bool
}

func (h *HttpHandle) buildProfitWithdrawTx(p *paramProfitWithdrawTx) (*txbuilder.BuildTransactionParams, error) {
	var txParams txbuilder.BuildTransactionParams

	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		Since:          0,
		PreviousOutput: p.subAccountLiveCell.OutPoint,
	})

	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: p.subAccountLiveCell.Output.Capacity,
		Lock:     p.subAccountLiveCell.Output.Lock,
		Type:     p.subAccountLiveCell.Output.Type,
	})

	subDataDetail := witness.ConvertSubAccountCellOutputData(p.subAccountLiveCell.OutputData)
	if p.isWithdrawDotBit {
		txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
			Capacity: subDataDetail.DasProfit,
			Lock:     h.DasCore.GetDasLock(),
			Type:     nil,
		})

		txParams.Outputs[0].Capacity -= subDataDetail.DasProfit
		subDataDetail.DasProfit = 0
		subAccountOutputData := witness.BuildSubAccountCellOutputData(subDataDetail)
		txParams.OutputsData = append(txParams.OutputsData, subAccountOutputData)
		txParams.OutputsData = append(txParams.OutputsData, []byte{})
	} else {
		ownerHex, err := h.DasCore.Daf().NormalToHex(core.DasAddressNormal{
			ChainType:     p.acc.OwnerChainType,
			AddressNormal: p.acc.Owner,
			Is712:         true,
		})
		if err != nil {
			return nil, fmt.Errorf("owner NormalToHex err: %s", err.Error())
		}
		ownerLock, ownerType, err := h.DasCore.Daf().HexToScript(ownerHex)
		if err != nil {
			return nil, fmt.Errorf("owner HexToScript err: %s", err.Error())
		}

		txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
			Capacity: subDataDetail.OwnerProfit,
			Lock:     ownerLock,
			Type:     ownerType,
		})

		txParams.Outputs[0].Capacity -= subDataDetail.OwnerProfit
		subDataDetail.OwnerProfit = 0
		if subDataDetail.DasProfit >= common.MinCellOccupiedCkb {
			txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
				Capacity: subDataDetail.DasProfit,
				Lock:     h.DasCore.GetDasLock(),
				Type:     nil,
			})

			txParams.Outputs[0].Capacity -= subDataDetail.DasProfit
			subDataDetail.DasProfit = 0
		}
		subAccountOutputData := witness.BuildSubAccountCellOutputData(subDataDetail)
		txParams.OutputsData = append(txParams.OutputsData, subAccountOutputData)
		txParams.OutputsData = append(txParams.OutputsData, []byte{})
		if len(txParams.Outputs) == 3 {
			txParams.OutputsData = append(txParams.OutputsData, []byte{})
		}
	}

	// witness
	actionWitness, err := witness.GenActionDataWitnessV2(common.DasActionCollectSubAccountProfit, nil, "")
	if err != nil {
		return nil, fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	// account
	accountOutPoint := common.String2OutPointStruct(p.acc.Outpoint)
	txAcc, err := h.DasCore.Client().GetTransaction(h.Ctx, accountOutPoint.TxHash)
	if err != nil {
		return nil, fmt.Errorf("GetTransaction err: %s", err.Error())
	}
	mapAcc, err := witness.AccountIdCellDataBuilderFromTx(txAcc.Transaction, common.DataTypeNew)
	if err != nil {
		return nil, fmt.Errorf("AccountIdCellDataBuilderFromTx err: %s", err.Error())
	}
	if item, ok := mapAcc[p.acc.AccountId]; !ok {
		return nil, fmt.Errorf("not exist acc builder: %s", p.acc.AccountId)
	} else {
		accWitness, _, _ := item.GenWitness(&witness.AccountCellParam{
			OldIndex: 0,
			NewIndex: 0,
			Action:   common.DasActionCollectSubAccountProfit,
		})
		txParams.Witnesses = append(txParams.Witnesses, accWitness)
	}

	// cell deps
	heightCell, err := h.DasCore.GetHeightCell()
	if err != nil {
		return nil, fmt.Errorf("GetHeightCell err: %s", err.Error())
	}
	timeCell, err := h.DasCore.GetTimeCell()
	if err != nil {
		return nil, fmt.Errorf("GetTimeCell err: %s", err.Error())
	}
	configCellSubAcc, err := core.GetDasConfigCellInfo(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	contractDasLock, err := core.GetDasContractInfo(common.DasContractNameDispatchCellType)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	contractSubAccount, err := core.GetDasContractInfo(common.DASContractNameSubAccountCellType)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}

	txParams.CellDeps = append(txParams.CellDeps,
		&types.CellDep{
			OutPoint: accountOutPoint,
			DepType:  types.DepTypeCode,
		},
		contractDasLock.ToCellDep(),
		heightCell.ToCellDep(),
		timeCell.ToCellDep(),
		configCellSubAcc.ToCellDep(),
		contractSubAccount.ToCellDep(),
	)

	return &txParams, nil
}
