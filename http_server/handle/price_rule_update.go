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
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/nervosnetwork/ckb-sdk-go/crypto/blake2b"
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
	log.Info("doPriceRuleUpdate:", toolib.JsonString(resp))

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

	txParams.Inputs = append(txParams.Inputs,
		&types.CellInput{
			PreviousOutput: accountOutpoint,
		},
		&types.CellInput{
			PreviousOutput: subAccountCell.OutPoint,
		},
	)

	// account cell
	accountCellOutput := accountTx.Transaction.Outputs[accountOutpoint.Index]
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: accountCellOutput.Capacity,
		Lock:     accountCellOutput.Lock,
		Type:     accountCellOutput.Type,
	})
	txParams.OutputsData = append(txParams.OutputsData, accountTx.Transaction.OutputsData[accountOutpoint.Index])

	// sub_account cell
	subAccountCellOutput := subAccountTx.Transaction.Outputs[subAccountCell.OutPoint.Index]
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: subAccountCellOutput.Capacity,
		Lock:     subAccountCellOutput.Lock,
		Type:     subAccountCellOutput.Type,
	})
	subAccountCellDetail := witness.ConvertSubAccountCellOutputData(subAccountTx.Transaction.OutputsData[subAccountCell.OutPoint.Index])
	subAccountCellDetail.Flag = witness.FlagTypeCustomRule
	if len(enableSwitch) > 0 {
		subAccountCellDetail.AutoDistribution = enableSwitch[0]
	} else if len(inputActionDataType) > 0 {
		subAccountCellDetail.AutoDistribution = witness.AutoDistributionEnable
	}

	rulesResult := make([][]byte, 0)
	whiteListMap := make(map[string]Whitelist)
	// Assemble price rules and calculate rule hash
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
				v.Ast.Arguments[0].Type == witness.Variable &&
				v.Ast.Arguments[0].Name == string(witness.Account) &&
				v.Ast.Arguments[1].Type == witness.Value &&
				v.Ast.Arguments[1].ValueType == witness.BinaryArray {

				accWhitelist := gconv.Strings(v.Ast.Arguments[1].Value)
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
		rulesData := make([]byte, 0)
		for _, v := range ruleData {
			rulesData = append(rulesData, v...)
		}

		hash := make([]byte, 10)
		if len(rulesData) > 0 {
			blakeHash, err := blake2b.Blake256(rulesData)
			if err != nil {
				return nil, nil, err
			}
			hash = blakeHash[:10]
		}

		switch inputActionDataType[0] {
		case common.ActionDataTypeSubAccountPriceRules:
			subAccountCellDetail.PriceRulesHash = hash
		case common.ActionDataTypeSubAccountPreservedRules:
			subAccountCellDetail.PreservedRulesHash = hash
		}

		// add actionDataType to prefix
		rulesResult, err = ruleEntity.GenDasData(inputActionDataType[0], ruleData)
		if err != nil {
			return nil, nil, err
		}
	}
	newSubAccountCellOutputData := witness.BuildSubAccountCellOutputData(subAccountCellDetail)
	txParams.OutputsData = append(txParams.OutputsData, newSubAccountCellOutputData)

	// witness
	actionWitness, err := witness.GenActionDataWitness(common.DasActionConfigSubAccount, common.Hex2Bytes(common.ParamOwner))
	if err != nil {
		return nil, nil, err
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	// witness account cell
	accBuilderMap, err := witness.AccountIdCellDataBuilderFromTx(accountTx.Transaction, common.DataTypeNew)
	if err != nil {
		return nil, nil, fmt.Errorf("AccountIdCellDataBuilderFromTx err: %s", err.Error())
	}
	accBuilder, ok := accBuilderMap[parentAccountId]
	if !ok {
		return nil, nil, fmt.Errorf("accBuilderMap is nil: %s", parentAccountId)
	}
	accWitness, _, _ := accBuilder.GenWitness(&witness.AccountCellParam{
		OldIndex: 0,
		NewIndex: 0,
		Action:   common.DasActionConfigSubAccount,
	})
	txParams.Witnesses = append(txParams.Witnesses, accWitness)

	// witness sub_account cell
	subAccountConfigTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccountCell.OutPoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return nil, nil, err
	}
	if err := witness.GetWitnessDataFromTx(subAccountConfigTx.Transaction, func(actionDataType common.ActionDataType, dataBys []byte, index int) (bool, error) {
		if (len(inputActionDataType) == 0 || inputActionDataType[0] != actionDataType) &&
			(actionDataType == common.ActionDataTypeSubAccountPriceRules ||
				actionDataType == common.ActionDataTypeSubAccountPreservedRules) {

			if len(inputActionDataType) > 0 && inputActionDataType[0] == actionDataType {
				return true, nil
			}
			txParams.Witnesses = append(txParams.Witnesses, witness.GenDasDataWitnessWithByte(actionDataType, dataBys))
		}
		return true, nil
	}); err != nil {
		return nil, nil, err
	}

	// rule witness
	for _, v := range rulesResult {
		txParams.Witnesses = append(txParams.Witnesses, v)
	}
	return txParams, whiteListMap, nil
}
