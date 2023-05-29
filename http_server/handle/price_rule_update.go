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
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/nervosnetwork/ckb-sdk-go/crypto/blake2b"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"math"
	"net/http"
	"strings"
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

	action := common.DasActionConfigSubAccount
	if err := h.check(address, req.Account, action, apiResp); err != nil {
		return err
	}
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))

	txParams, whiteListMap, err := h.rulesTxAssemble(RulesTxAssembleParams{
		Req:                 req,
		ApiResp:             apiResp,
		InputActionDataType: common.ActionDataTypeSubAccountPriceRules,
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
	log.Info("doPriceRuleUpdate:", toolib.JsonString(resp))

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
				RuleType:        tables.RuleTypePriceRules,
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

type RulesTxAssembleParams struct {
	Req                 *ReqPriceRuleUpdate
	ApiResp             *api_code.ApiResp
	InputActionDataType common.ActionDataType
	AutoDistribution    witness.AutoDistribution
}

func (h *HttpHandle) rulesTxAssemble(params RulesTxAssembleParams) (*txbuilder.BuildTransactionParams, map[string]Whitelist, error) {
	if params.Req == nil || params.ApiResp == nil {
		return nil, nil, fmt.Errorf("params invalid")
	}

	res, err := params.Req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		params.ApiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil, nil, err
	}
	address := common.FormatAddressPayload(res.AddressPayload, res.DasAlgorithmId)

	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(params.Req.Account))
	baseInfo, err := h.TxTool.GetBaseInfo()
	if err != nil {
		params.ApiResp.ApiRespErr(api_code.ApiCodeError500, "server error")
		return nil, nil, err
	}

	accountInfo, err := h.DbDao.GetAccountInfoByAccountId(parentAccountId)
	if err != nil {
		params.ApiResp.ApiRespErr(api_code.ApiCodeDbError, "internal error")
		return nil, nil, fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
	}
	if accountInfo.Id == 0 {
		params.ApiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account no exist")
		return nil, nil, fmt.Errorf("account no exist")
	}
	accountOutpoint := common.String2OutPointStruct(accountInfo.Outpoint)
	accountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, accountOutpoint.TxHash)
	if err != nil {
		params.ApiResp.ApiRespErr(api_code.ApiCodeError500, "server error")
		return nil, nil, err
	}

	subAccountCell, err := h.getSubAccountCell(baseInfo.ContractSubAcc, parentAccountId)
	if err != nil {
		params.ApiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return nil, nil, fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
	}
	subAccountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccountCell.OutPoint.TxHash)
	if err != nil {
		params.ApiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
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
	if params.InputActionDataType == "" {
		subAccountCellDetail.AutoDistribution = params.AutoDistribution
	} else {
		subAccountCellDetail.AutoDistribution = witness.AutoDistributionEnable
	}

	reqRuleData := make([][]byte, 0)
	rulesResult := make([][]byte, 0)
	whiteListMap := make(map[string]Whitelist)
	// Assemble price rules and calculate rule hash
	if params.InputActionDataType != "" {
		ruleEntity := witness.NewSubAccountRuleEntity(params.Req.Account)
		ruleEntity.Rules = params.Req.List
		if err := ruleEntity.Check(); err != nil {
			params.ApiResp.ApiRespErr(api_code.ApiCodeRuleFormatErr, err.Error())
			return nil, nil, err
		}

		token, err := h.DbDao.GetTokenById(tables.TokenIdCkb)
		if err != nil {
			params.ApiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return nil, nil, err
		}

		builder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount)
		if err != nil {
			params.ApiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return nil, nil, fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
		}
		newSubAccountPrice, _ := molecule.Bytes2GoU64(builder.ConfigCellSubAccount.NewSubAccountPrice().RawData())

		preservedInList := 0

		for idx, v := range ruleEntity.Rules {
			if params.InputActionDataType == common.ActionDataTypeSubAccountPriceRules {
				if v.Price <= 0 {
					err = fmt.Errorf("price not be less than min %d", newSubAccountPrice)
					params.ApiResp.ApiRespErr(api_code.ApiCodePriceRulePriceNotBeLessThanMin, err.Error())
					return nil, nil, err
				}

				if math.Round(v.Price*10000)/10000 != v.Price {
					err = errors.New("price most be two decimal places")
					params.ApiResp.ApiRespErr(api_code.ApiCodePriceMostReserveTwoDecimal, err.Error())
					return nil, nil, err
				}

				price := decimal.NewFromInt(int64(newSubAccountPrice)).Mul(token.Price).Div(decimal.NewFromFloat(math.Pow10(int(token.Decimals))))
				if price.GreaterThan(decimal.NewFromFloat(v.Price)) {
					err = fmt.Errorf("price not be less than min: %s$", price)
					params.ApiResp.ApiRespErr(api_code.ApiCodePriceRulePriceNotBeLessThanMin, err.Error())
					return nil, nil, err
				}
				ruleEntity.Rules[idx].Price *= math.Pow10(6)
			}

			if v.Ast.Type == witness.Function &&
				v.Ast.Name == string(witness.FunctionInList) &&
				v.Ast.Arguments[0].Type == witness.Variable &&
				v.Ast.Arguments[0].Name == string(witness.Account) &&
				v.Ast.Arguments[1].Type == witness.Value {

				accWhitelist := gconv.Strings(v.Ast.Arguments[1].Value)

				if len(accWhitelist) > 999 {
					err = errors.New("account list most be less than 1000")
					params.ApiResp.ApiRespErr(api_code.ApiCodeInListMostBeLessThan1000, err.Error())
					return nil, nil, err
				}

				if params.InputActionDataType == common.ActionDataTypeSubAccountPreservedRules {
					preservedInList += 1
					if preservedInList > 1 {
						err = errors.New("preserved in_list rules most be one")
						params.ApiResp.ApiRespErr(api_code.ApiCodePreservedRulesMostBeOne, err.Error())
						return nil, nil, err
					}
				}

				for _, v := range accWhitelist {
					accountName := v + "." + params.Req.Account
					h.checkSubAccountName(params.ApiResp, accountName)
					if params.ApiResp.ErrNo != api_code.ApiCodeSuccess {
						return nil, nil, errors.New("account name invalid")
					}
					accId := common.Bytes2Hex(common.GetAccountIdByAccount(accountName))
					if _, ok := whiteListMap[accId]; ok {
						err = fmt.Errorf("account: %s repeat", accountName)
						params.ApiResp.ApiRespErr(api_code.ApiCodeAccountRepeat, err.Error())
						return nil, nil, err
					}
					whiteListMap[accId] = Whitelist{
						Index:   idx,
						Account: accountName,
					}
				}
			}
		}

		reqRuleData, err = ruleEntity.GenData()
		if err != nil {
			return nil, nil, err
		}
		// add actionDataType to prefix
		rulesResult, err = ruleEntity.GenDasData(params.InputActionDataType, reqRuleData)
		if err != nil {
			return nil, nil, err
		}
	}

	// witness
	var witnessParams []byte
	if strings.EqualFold(accountInfo.Owner, address) {
		witnessParams = common.Hex2Bytes(common.ParamOwner)
	} else if strings.EqualFold(accountInfo.Manager, address) {
		witnessParams = common.Hex2Bytes(common.ParamManager)
	}
	actionWitness, err := witness.GenActionDataWitnessV3(common.DasActionConfigSubAccount, witnessParams)
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

	// rule witness
	ruleWitnessSize := 0
	hashMap := map[common.ActionDataType][][]byte{
		common.ActionDataTypeSubAccountPriceRules:     make([][]byte, 0),
		common.ActionDataTypeSubAccountPreservedRules: make([][]byte, 0),
	}
	if params.InputActionDataType != "" {
		hashMap[params.InputActionDataType] = reqRuleData
	}
	for _, v := range rulesResult {
		ruleWitnessSize += len(v)
		txParams.Witnesses = append(txParams.Witnesses, v)
	}

	subAccountConfigTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccountCell.OutPoint.TxHash)
	if err != nil {
		params.ApiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return nil, nil, err
	}
	if err := witness.GetWitnessDataFromTx(subAccountConfigTx.Transaction, func(actionDataType common.ActionDataType, dataBys []byte, index int) (bool, error) {
		if (params.InputActionDataType == "" || params.InputActionDataType != actionDataType) &&
			(actionDataType == common.ActionDataTypeSubAccountPriceRules || actionDataType == common.ActionDataTypeSubAccountPreservedRules) {
			ruleBytes := witness.GenDasDataWitnessWithByte(actionDataType, dataBys)
			ruleWitnessSize += len(ruleBytes)
			txParams.Witnesses = append(txParams.Witnesses, ruleBytes)
			hashMap[actionDataType] = append(hashMap[actionDataType], dataBys[12:])
		}
		return true, nil
	}); err != nil {
		return nil, nil, err
	}

	for actionDataType, ruleData := range hashMap {
		hash, err := ruleWitnessHash(ruleData)
		if err != nil {
			return nil, nil, err
		}
		switch actionDataType {
		case common.ActionDataTypeSubAccountPriceRules:
			subAccountCellDetail.PriceRulesHash = hash
		case common.ActionDataTypeSubAccountPreservedRules:
			subAccountCellDetail.PreservedRulesHash = hash
		}
	}
	newSubAccountCellOutputData := witness.BuildSubAccountCellOutputData(subAccountCellDetail)
	txParams.OutputsData = append(txParams.OutputsData, newSubAccountCellOutputData)

	// rule witness most size check
	if ruleWitnessSize > 441*1e3 {
		err = errors.New("rule size exceeds limit")
		params.ApiResp.ApiRespErr(api_code.ApiCodeRuleSizeExceedsLimit, err.Error())
		return nil, nil, err
	}
	return txParams, whiteListMap, nil
}

func ruleWitnessHash(ruleData [][]byte) ([]byte, error) {
	hash := make([]byte, 10)
	if len(ruleData) > 0 {
		totalHash := make([]byte, 0)
		for _, v := range ruleData {
			hashData, err := blake2b.Blake256(v)
			if err != nil {
				return nil, err
			}
			totalHash = append(totalHash, hashData...)
		}
		hashData, err := blake2b.Blake256(totalHash)
		if err != nil {
			return nil, err
		}
		hash = hashData[:10]
	}
	return hash, nil
}
