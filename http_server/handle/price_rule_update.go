package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/nervosnetwork/ckb-sdk-go/crypto/blake2b"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"gorm.io/gorm"
	"net/http"
)

type ReqPriceRuleUpdate struct {
	core.ChainTypeAddress
	Account string                      `json:"account" binding:"required"`
	List    witness.SubAccountRuleSlice `json:"list" binding:"required"`
}

func (h *HttpHandle) PriceRuleUpdate(ctx *gin.Context) {
	var (
		funcName               = "PriceRuleUpdate"
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
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req))

	if err = h.doPriceRuleUpdate(&req, &apiResp); err != nil {
		log.Error("doConfigAutoMintUpdate err:", err.Error(), funcName, clientIp, remoteAddrIP)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doPriceRuleUpdate(req *ReqPriceRuleUpdate, apiResp *api_code.ApiResp) error {
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

	txParams, whiteListMap, err := h.rulesTxAssemble(req, apiResp, []common.ActionDataType{common.ActionDataTypeSubAccountPriceRules})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "build tx error")
		return err
	}

	signKey, signList, txHash, err := h.buildTx(&paramBuildTx{
		txParams:  txParams,
		chainType: res.ChainType,
		address:   res.AddressHex,
		action:    common.DasActionConfigSubAccount,
		account:   req.Account,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "buildTx err: "+err.Error())
		return fmt.Errorf("buildTx err: %s", err.Error())
	}

	resp := RespConfigAutoMintUpdate{}
	resp.Action = common.DasActionConfigSubAccount
	resp.SignKey = signKey
	resp.List = append(resp.List, SignInfo{
		SignList: signList,
	})
	log.Info("doCustomScript:", toolib.JsonString(resp))

	if err := h.DbDao.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("parent_account_id=? and rule_type=?", parentAccountId, tables.RuleTypePriceRules).
			Delete(&tables.RuleWhitelist{}).Error; err != nil {
			return err
		}
		for accountId, whiteList := range whiteListMap {
			if err := tx.Create(tables.RuleWhitelist{
				TxHash:          txHash,
				ParentAccount:   req.Account,
				ParentAccountId: parentAccountId,
				RuleType:        tables.RuleTypePriceRules,
				RuleIndex:       whiteList.Index,
				Account:         whiteList.Account,
				AccountId:       accountId,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return err
	}

	apiResp.ApiRespOK(resp)
	return nil
}

type Whitelist struct {
	Index   int
	Account string
}

func (h *HttpHandle) rulesTxAssemble(req *ReqPriceRuleUpdate, apiResp *api_code.ApiResp, inputActionDataType []common.ActionDataType, enableSwitch ...witness.AutoDistribution) (*txbuilder.BuildTransactionParams, map[string]Whitelist, error) {
	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil, nil, err
	}
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))

	baseInfo, err := h.TxTool.GetBaseInfo()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "server error")
		return nil, nil, err
	}

	accountInfo, err := h.DbDao.GetAccountInfoByAccountId(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "internal error")
		return nil, nil, fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
	}
	if accountInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account no exist")
		return nil, nil, fmt.Errorf("account no exist")
	}
	accountOutpoint := common.String2OutPointStruct(accountInfo.Outpoint)
	accountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, accountOutpoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "server error")
		return nil, nil, err
	}

	subAccountCell, err := h.getSubAccountCell(baseInfo.ContractSubAcc, parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return nil, nil, fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
	}
	subAccountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccountCell.OutPoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return nil, nil, err
	}

	txParams := &txbuilder.BuildTransactionParams{}
	txParams.CellDeps = append(txParams.CellDeps,
		baseInfo.ContractAcc.ToCellDep(),
		baseInfo.ContractSubAcc.ToCellDep(),
		baseInfo.TimeCell.ToCellDep(),
		baseInfo.HeightCell.ToCellDep(),
		baseInfo.ConfigCellAcc.ToCellDep(),
		baseInfo.ConfigCellSubAcc.ToCellDep(),
	)

	dasLock, _, err := h.DasCore.Daf().HexToScript(*res)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "HexToArgs err")
		return nil, nil, err
	}
	balanceLiveCells, _, err := h.DasCore.GetBalanceCells(&core.ParamGetBalanceCells{
		DasCache:          h.DasCache,
		LockScript:        dasLock,
		CapacityNeed:      common.OneCkb,
		CapacityForChange: common.DasLockWithBalanceTypeOccupiedCkb,
		SearchOrder:       indexer.SearchOrderDesc,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return nil, nil, fmt.Errorf("GetBalanceCells err: %s", err.Error())
	}
	txParams.Inputs = append(txParams.Inputs,
		&types.CellInput{
			PreviousOutput: accountOutpoint,
		},
		&types.CellInput{
			PreviousOutput: subAccountCell.OutPoint,
		},
	)
	for _, v := range balanceLiveCells {
		txParams.Inputs = append(txParams.Inputs, &types.CellInput{
			PreviousOutput: v.OutPoint,
		})
	}

	// account cell
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: accountTx.Transaction.Outputs[accountOutpoint.Index].Capacity,
		Lock:     accountTx.Transaction.Outputs[accountOutpoint.Index].Lock,
		Type:     accountTx.Transaction.Outputs[accountOutpoint.Index].Type,
	})
	txParams.OutputsData = append(txParams.OutputsData, accountTx.Transaction.OutputsData[accountOutpoint.Index])

	// sub_account cell
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: subAccountTx.Transaction.Outputs[subAccountCell.TxIndex].Capacity,
		Lock:     subAccountTx.Transaction.Outputs[subAccountCell.TxIndex].Lock,
		Type:     subAccountTx.Transaction.Outputs[subAccountCell.TxIndex].Type,
	})
	subAccountCellDetail := witness.ConvertSubAccountCellOutputData(subAccountTx.Transaction.OutputsData[subAccountCell.TxIndex])
	subAccountCellDetail.Flag = witness.FlagTypeCustomRule
	if len(enableSwitch) > 0 {
		subAccountCellDetail.AutoDistribution = enableSwitch[0]
	} else if len(inputActionDataType) > 0 {
		subAccountCellDetail.AutoDistribution = witness.AutoDistributionEnable
	}
	newSubAccountCellOutputData := witness.BuildSubAccountCellOutputData(subAccountCellDetail)
	txParams.OutputsData = append(txParams.OutputsData, newSubAccountCellOutputData)

	for _, v := range balanceLiveCells {
		txParams.Outputs = append(txParams.Outputs, v.Output)
		txParams.OutputsData = append(txParams.OutputsData, v.OutputData)
	}

	var rulesResult [][]byte
	whiteListMap := make(map[string]Whitelist)
	if len(inputActionDataType) == 1 {
		ruleEntity := witness.NewSubAccountRuleEntity(req.Account)
		ruleEntity.Version = witness.SubAccountRuleVersionV1
		ruleEntity.Rules = req.List
		if err := ruleEntity.Check(); err != nil {
			return nil, nil, err
		}

		for idx, v := range ruleEntity.Rules {
			if v.Ast.Type == witness.Function &&
				v.Ast.Name == string(witness.FunctionInList) &&
				v.Ast.Expressions[0].Type == witness.Variable &&
				v.Ast.Expressions[0].Name == string(witness.Account) &&
				v.Ast.Expressions[1].Type == witness.Value &&
				v.Ast.Expressions[1].ValueType == witness.BinaryArray {

				accWhitelist := gconv.Strings(v.Ast.Expressions[1].Value)
				for _, v := range accWhitelist {
					accId := common.Bytes2Hex(common.GetAccountIdByAccount(v))
					whiteListMap[accId] = Whitelist{
						Index:   idx,
						Account: v,
					}
				}
			}
		}

		ruleData, err := ruleEntity.GenData()
		if err != nil {
			return nil, nil, err
		}
		// Assemble the split rules into one for easy hashing
		rulesBuilder := molecule.NewSubAccountRulesBuilder()
		for _, v := range ruleData {
			subAccountRules, err := molecule.SubAccountRulesFromSlice(v, true)
			if err != nil {
				return nil, nil, err
			}
			for i := uint(0); i < subAccountRules.ItemCount(); i++ {
				subAccountRule := subAccountRules.Get(i)
				rulesBuilder.Push(*subAccountRule)
			}
		}
		rules := rulesBuilder.Build()
		totalRules, err := ruleEntity.GenWitnessDataWithRuleData(inputActionDataType[0], [][]byte{rules.AsSlice()})
		if err != nil {
			return nil, nil, err
		}

		hash, err := blake2b.Blake256(totalRules[0])
		if err != nil {
			return nil, nil, err
		}

		switch inputActionDataType[0] {
		case common.ActionDataTypeSubAccountPriceRules:
			subAccountCellDetail.PriceRulesHash = hash[:10]
		case common.ActionDataTypeSubAccountPreservedRules:
			subAccountCellDetail.PreservedRulesHash = hash[:10]
		}

		rulesResult, err = ruleEntity.GenWitnessDataWithRuleData(inputActionDataType[0], ruleData)
		if err != nil {
			return nil, nil, err
		}
	}

	// witness
	actionWitness, err := witness.GenActionDataWitness(common.DasActionConfigSubAccount, common.Hex2Bytes(common.ParamOwner))
	if err != nil {
		return nil, nil, err
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	if len(inputActionDataType) == 1 {
		for _, v := range rulesResult {
			txParams.Witnesses = append(txParams.Witnesses, v)
		}
	}

	ruleConfig, err := h.DbDao.GetRuleConfigByAccountId(parentAccountId)
	if err != nil {
		return nil, nil, err
	}
	if ruleConfig.Id > 0 {
		subAccountConfigTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccountCell.OutPoint.TxHash)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
			return nil, nil, err
		}
		if err := witness.GetWitnessDataFromTx(subAccountConfigTx.Transaction, func(actionDataType common.ActionDataType, dataBys []byte) (bool, error) {
			if len(inputActionDataType) == 0 || inputActionDataType[0] != actionDataType {
				txParams.Witnesses = append(txParams.Witnesses, witness.GenDasDataWitnessWithByte(actionDataType, dataBys))
			}
			return true, nil
		}); err != nil {
			return nil, nil, err
		}
	}
	return txParams, whiteListMap, nil
}
