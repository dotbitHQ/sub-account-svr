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
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"github.com/thoas/go-funk"
	"math"
	"net/http"
	"strings"
)

type RuleType int

const (
	RuleTypeConditions RuleType = 1
	RuleTypeWhitelist  RuleType = 2
)

var Ops = []string{"==", ">", ">=", "<", "<=", "not"}
var Functions = []string{"include_chars", "only_include_charset"}

type ReqPriceRuleUpdate struct {
	core.ChainTypeAddress
	Account string   `json:"account" binding:"required"`
	List    ReqRules `json:"list" binding:"required"`
}

type ReqRules []ReqRule

type ReqRule struct {
	Name       string         `json:"name" binding:"required"`
	Note       string         `json:"note"`
	Price      float64        `json:"price"`
	Type       RuleType       `json:"type" binding:"required;oneof=1 2"`
	Whitelist  []string       `json:"whitelist"`
	Conditions []ReqCondition `json:"conditions"`
}

type ReqCondition struct {
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

	baseInfo, err := h.TxTool.GetBaseInfo()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "server error")
		return err
	}

	accountInfo, err := h.DbDao.GetAccountInfoByAccountId(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "internal error")
		return fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
	}
	if accountInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account no exist")
		return fmt.Errorf("account no exist")
	}
	accountOutpoint := common.String2OutPointStruct(accountInfo.Outpoint)
	accountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, accountOutpoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "server error")
		return err
	}

	subAccountCell, err := h.getSubAccountCell(baseInfo.ContractSubAcc, parentAccountId)
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
		baseInfo.ContractSubAcc.ToCellDep(),
		baseInfo.TimeCell.ToCellDep(),
		baseInfo.HeightCell.ToCellDep(),
		baseInfo.ConfigCellAcc.ToCellDep(),
		baseInfo.ConfigCellSubAcc.ToCellDep(),
	)

	dasLock, _, err := h.DasCore.Daf().HexToScript(*res)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "HexToArgs err")
		return err
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
		return fmt.Errorf("GetBalanceCells err: %s", err.Error())
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

	priceRuleWitnessData, err := req.ParseToSubAccountRule()
	if err != nil {
		return err
	}
	totalBytes := make([]byte, 0)
	for i := 0; i < len(priceRuleWitnessData); i++ {
		totalBytes = append(totalBytes, priceRuleWitnessData[i]...)
		txParams.Witnesses = append(txParams.Witnesses, priceRuleWitnessData[i])
	}
	hash, err := blake2b.Blake256(totalBytes)
	if err != nil {
		return err
	}
	subAccountCellDetail.PriceRulesHash = hash[:10]

	newSubAccountCellOutputData := witness.BuildSubAccountCellOutputData(subAccountCellDetail)
	txParams.OutputsData = append(txParams.OutputsData, newSubAccountCellOutputData)

	for _, v := range balanceLiveCells {
		txParams.Outputs = append(txParams.Outputs, v.Output)
		txParams.OutputsData = append(txParams.OutputsData, v.OutputData)
	}

	if err := witness.GetWitnessDataFromTx(subAccountTx.Transaction, func(actionDataType common.ActionDataType, dataBys []byte) (bool, error) {
		if actionDataType == common.DasActionSubAccountPreservedRule {
			txParams.Witnesses = append(txParams.Witnesses, dataBys)
		}
		return true, nil
	}); err != nil {
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

	if err := h.DbDao.CreatePriceConfig(tables.PriceConfig{
		Account:   req.Account,
		AccountId: common.Bytes2Hex(common.GetAccountIdByAccount(req.Account)),
		Action:    tables.PriceConfigActionAutoMintSwitch,
		TxHash:    txHash.String(),
	}); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return fmt.Errorf("CreatePriceConfig err: %s", err.Error())
	}
	apiResp.ApiRespOK(resp)
	return nil
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
				if vv.VarName == witness.Account && vv.Op != string(witness.FunctionIncludeCharts) {
					apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
					return fmt.Errorf("params invalid")
				}

				if vv.Value == witness.AccountChars && vv.Op != string(witness.FunctionOnlyIncludeCharset) {
					apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
					return fmt.Errorf("params invalid")
				}

				if vv.Value == witness.AccountLength {
					find := false
					for _, op := range Ops {
						if vv.Op == op {
							find = true
							break
						}
					}
					if !find {
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

func (r *ReqPriceRuleUpdate) ParseToSubAccountRule() ([][]byte, error) {
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
					rule.Ast.Name = v.Op
					rule.Ast.Expressions = append(rule.Ast.Expressions, witness.ExpressionEntity{
						Type: witness.Variable,
						Name: string(v.VarName),
					})

					if v.Op == string(witness.FunctionIncludeCharts) {
						rule.Ast.Expressions = append(rule.Ast.Expressions, witness.ExpressionEntity{
							Type:      witness.Value,
							ValueType: witness.StringArray,
							Value:     gconv.Strings(v.Value),
						})
					}
					if v.Op == string(witness.FunctionOnlyIncludeCharset) {
						rule.Ast.Expressions = append(rule.Ast.Expressions, witness.ExpressionEntity{
							Type:      witness.Value,
							ValueType: witness.Charset,
							Value:     gconv.String(v.Value),
						})
					}
				} else {
					rule.Ast.Type = witness.Operator
					rule.Ast.Symbol = witness.SymbolType(v.Op)
					rule.Ast.Expressions = append(rule.Ast.Expressions, witness.ExpressionEntity{
						Type: witness.Variable,
						Name: string(v.VarName),
					})
					rule.Ast.Expressions = append(rule.Ast.Expressions, witness.ExpressionEntity{
						Type:      witness.Value,
						ValueType: witness.Uint8,
						Value:     gconv.Uint8(v.Value),
					})
				}
			}
		case RuleTypeWhitelist:
			rule.Ast.Type = witness.Function
			rule.Ast.Name = string(witness.FunctionInList)
			rule.Ast.Expressions = append(rule.Ast.Expressions, witness.ExpressionEntity{
				Type: witness.Variable,
				Name: string(witness.Account),
			})

			subAccountIds := make([]string, 0)
			for _, v := range ruleReq.Whitelist {
				subAccount := strings.Split(strings.TrimSpace(v), ".")[0]
				subAccount = subAccount + "." + r.Account
				subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(subAccount))
				subAccountIds = append(subAccountIds, subAccountId)
			}
			rule.Ast.Expressions = append(rule.Ast.Expressions, witness.ExpressionEntity{
				Type:      witness.Value,
				ValueType: witness.BinaryArray,
				Value:     subAccountIds,
			})
		}
		ruleEntity.Rules = append(ruleEntity.Rules, rule)
	}
	return ruleEntity.GenWitnessData(common.DasActionSubAccountPriceRule)
}
