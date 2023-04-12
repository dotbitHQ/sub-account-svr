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
	"github.com/thoas/go-funk"
	"gorm.io/gorm"
	"math"
	"net/http"
	"strings"
)

type RuleType int

const (
	RuleTypeConditions RuleType = 1
	RuleTypeWhitelist  RuleType = 2
)

var Ops = map[string]struct{}{"==": {}, ">": {}, ">=": {}, "<": {}, "<=": {}, "not": {}}
var Functions = map[string]struct{}{"include_chars": {}, "only_include_charset": {}}

type ReqPriceRuleUpdate struct {
	core.ChainTypeAddress
	Account string `json:"account" binding:"required"`
	List    Rules  `json:"list" binding:"required"`
}

type Rules []Rule

type Rule struct {
	Name       string      `json:"name" binding:"required"`
	Note       string      `json:"note"`
	Price      float64     `json:"price"`
	Type       RuleType    `json:"type" binding:"required;oneof=1 2"`
	Whitelist  []string    `json:"whitelist,omitempty"`
	Conditions []Condition `json:"conditions,omitempty"`
}

type Condition struct {
	VarName witness.VariableName `json:"var_name" binding:"required;oneof=account_chars account_length"`
	Op      string               `json:"op" binding:"required;oneof=include_chars only_include_charset == > >= < <= not"`
	Value   interface{}          `json:"value" binding:"required"`
}

func (h *HttpHandle) PriceRuleUpdate(ctx *gin.Context) {
	var (
		funcName = "PriceRuleUpdate"
		clientIp = GetClientIp(ctx)
		req      ReqPriceRuleUpdate
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

	if err = h.doPriceRuleUpdate(&req, &apiResp); err != nil {
		log.Error("doConfigAutoMintUpdate err:", err.Error(), funcName, clientIp)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doPriceRuleUpdate(req *ReqPriceRuleUpdate, apiResp *api_code.ApiResp) error {
	// req params check
	if err := h.reqCheck(req, apiResp); err != nil {
		return err
	}

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

	if err := h.checkAuth(address, req.Account, apiResp); err != nil {
		return err
	}
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))

	txParams, whiteListMap, err := h.rulesTxAssemble(common.ActionDataTypeSubAccountPriceRules, req, apiResp)
	if err != nil {
		return err
	}

	// build tx
	txBuilder := txbuilder.NewDasTxBuilderFromBase(h.TxBuilderBase, nil)
	if err := txBuilder.BuildTransaction(txParams); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "build tx error")
		return err
	}

	sizeInBlock, _ := txBuilder.Transaction.SizeInBlock()
	changeCapacity := txBuilder.Transaction.Outputs[len(txBuilder.Transaction.Outputs)-1].Capacity
	changeCapacity = changeCapacity - sizeInBlock - 5000
	log.Info("BuildCreateSubAccountTx change fee:", sizeInBlock)

	txBuilder.Transaction.Outputs[len(txBuilder.Transaction.Outputs)-1].Capacity = changeCapacity

	txHash, err := txBuilder.Transaction.ComputeHash()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "build tx error")
		return err
	}
	log.Info("BuildUpdateSubAccountTx:", txBuilder.TxString(), txHash.String())

	signKey, signList, err := h.buildTx(&paramBuildTx{
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
			Delete(tables.RuleWhitelist{}).Error; err != nil {
			return err
		}
		for accountId, whiteList := range whiteListMap {
			if err := tx.Create(tables.RuleWhitelist{
				TxHash:          txHash.String(),
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
		return err
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) rulesTxAssemble(inputActionDataType common.ActionDataType, req *ReqPriceRuleUpdate, apiResp *api_code.ApiResp) (*txbuilder.BuildTransactionParams, map[string]Whitelist, error) {
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
	subAccountCellDetail.AutoDistribution = witness.AutoDistributionEnable

	ruleEntity, whiteListMap, err := req.ParseToSubAccountRule()
	if err != nil {
		return nil, nil, err
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

	totalRules, err := ruleEntity.GenWitnessDataWithRuleData(inputActionDataType, [][]byte{rules.AsSlice()})
	if err != nil {
		return nil, nil, err
	}

	hash, err := blake2b.Blake256(totalRules[0])
	if err != nil {
		return nil, nil, err
	}

	switch inputActionDataType {
	case common.ActionDataTypeSubAccountPriceRules:
		subAccountCellDetail.PriceRulesHash = hash[:10]
	case common.ActionDataTypeSubAccountPreservedRules:
		subAccountCellDetail.PreservedRulesHash = hash[:10]
	}

	newSubAccountCellOutputData := witness.BuildSubAccountCellOutputData(subAccountCellDetail)
	txParams.OutputsData = append(txParams.OutputsData, newSubAccountCellOutputData)

	for _, v := range balanceLiveCells {
		txParams.Outputs = append(txParams.Outputs, v.Output)
		txParams.OutputsData = append(txParams.OutputsData, v.OutputData)
	}

	rulesResult, err := ruleEntity.GenWitnessDataWithRuleData(inputActionDataType, ruleData)
	if err != nil {
		return nil, nil, err
	}
	for _, v := range rulesResult {
		txParams.Witnesses = append(txParams.Witnesses, v)
	}

	if err := witness.GetWitnessDataFromTx(subAccountTx.Transaction, func(actionDataType common.ActionDataType, dataBys []byte) (bool, error) {
		if actionDataType != inputActionDataType {
			txParams.Witnesses = append(txParams.Witnesses, dataBys)
		}
		return true, nil
	}); err != nil {
		return nil, nil, err
	}
	return txParams, whiteListMap, nil
}

func (h *HttpHandle) reqCheck(req *ReqPriceRuleUpdate, apiResp *api_code.ApiResp) error {
	for _, v := range req.List {
		if v.Type == RuleTypeConditions && len(v.Conditions) == 0 {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
			return fmt.Errorf("params invalid")
		}

		if v.Type == RuleTypeWhitelist && len(v.Whitelist) == 0 {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
			return fmt.Errorf("params invalid")
		}

		if v.Price > 0 && float64(int(v.Price*100))/100 != v.Price {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
			return fmt.Errorf("params invalid")
		}

		switch v.Type {
		case RuleTypeConditions:
			for _, vv := range v.Conditions {
				if _, ok := Functions[vv.Op]; !ok && vv.VarName == witness.AccountChars {
					apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
					return fmt.Errorf("params invalid")
				}

				if vv.Op == string(witness.FunctionOnlyIncludeCharset) {
					charsetName := gconv.String(vv.Value)
					if _, ok := common.AccountCharTypeNameMap[charsetName]; !ok {
						err := fmt.Errorf("charset %s not support", vv.Value)
						apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
						return err
					}
				}

				if vv.VarName == witness.AccountLength {
					if _, ok := Ops[vv.Op]; !ok {
						apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
						return fmt.Errorf("params invalid")
					}
				}
			}
		case RuleTypeWhitelist:
			if len(v.Whitelist) == 0 {
				apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
				return fmt.Errorf("params invalid")
			}
		}
	}
	return nil
}

type Whitelist struct {
	Index   int    `json:"index"`
	Account string `json:"account"`
}

func (r *ReqPriceRuleUpdate) ParseToSubAccountRule() (*witness.SubAccountRuleEntity, map[string]Whitelist, error) {
	whiteList := make(map[string]Whitelist, 0)
	ruleEntity := witness.NewSubAccountRuleEntity(r.Account)
	for i := 0; i < len(r.List); i++ {
		ruleReq := r.List[i]
		rule := witness.SubAccountRule{
			Index: uint32(i),
			Name:  ruleReq.Name,
			Note:  ruleReq.Note,
			Price: uint64(math.Pow10(6) * ruleReq.Price),
		}
		switch ruleReq.Type {
		case RuleTypeConditions:
			for _, v := range ruleReq.Conditions {

				if funk.Contains(Functions, v.Op) {
					rule.Ast.Type = witness.Function
					rule.Ast.Expression.Name = v.Op
					rule.Ast.Expression.Expressions = append(rule.Ast.Expression.Expressions, witness.AstExpression{
						Type: witness.Variable,
						Expression: witness.ExpressionEntity{
							Name: string(v.VarName),
						},
					})

					if v.Op == string(witness.FunctionIncludeCharts) {
						rule.Ast.Expression.Expressions = append(rule.Ast.Expression.Expressions, witness.AstExpression{
							Type: witness.Value,
							Expression: witness.ExpressionEntity{
								ValueType: witness.StringArray,
								Value:     gconv.Strings(v.Value),
							},
						})
					}
					if v.Op == string(witness.FunctionOnlyIncludeCharset) {
						charsetType := common.AccountCharTypeNameMap[gconv.String(v.Value)]
						rule.Ast.Expression.Expressions = append(rule.Ast.Expression.Expressions, witness.AstExpression{
							Type: witness.Value,
							Expression: witness.ExpressionEntity{
								ValueType: witness.Charset,
								Value:     charsetType,
							},
						})
					}
					continue
				}

				rule.Ast.Type = witness.Operator
				rule.Ast.Expression.Symbol = witness.SymbolType(v.Op)
				rule.Ast.Expression.Expressions = append(rule.Ast.Expression.Expressions, witness.AstExpression{
					Type: witness.Variable,
					Expression: witness.ExpressionEntity{
						Name: string(v.VarName),
					},
				})
				rule.Ast.Expression.Expressions = append(rule.Ast.Expression.Expressions, witness.AstExpression{
					Type: witness.Value,
					Expression: witness.ExpressionEntity{
						ValueType: witness.Uint8,
						Value:     gconv.Uint8(v.Value),
					},
				})
			}
		case RuleTypeWhitelist:
			rule.Ast.Type = witness.Function
			rule.Ast.Expression.Name = string(witness.FunctionInList)
			rule.Ast.Expression.Expressions = append(rule.Ast.Expression.Expressions, witness.AstExpression{
				Type: witness.Variable,
				Expression: witness.ExpressionEntity{
					Name: string(witness.Account),
				},
			})

			subAccountIds := make([]string, 0)
			for _, v := range ruleReq.Whitelist {
				subAccount := strings.Split(strings.TrimSpace(v), ".")[0]
				subAccount = subAccount + "." + r.Account
				subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(subAccount))
				subAccountIds = append(subAccountIds, subAccountId)
				whiteList[subAccountId] = Whitelist{
					Index:   i,
					Account: subAccount,
				}
			}
			rule.Ast.Expression.Expressions = append(rule.Ast.Expression.Expressions, witness.AstExpression{
				Type: witness.Value,
				Expression: witness.ExpressionEntity{
					ValueType: witness.BinaryArray,
					Value:     subAccountIds,
				},
			})
		}
		ruleEntity.Rules = append(ruleEntity.Rules, rule)
	}
	return ruleEntity, whiteList, nil
}
