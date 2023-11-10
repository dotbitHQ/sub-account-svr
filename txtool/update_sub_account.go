package txtool

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"github.com/nervosnetwork/ckb-sdk-go/crypto/blake2b"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"time"
)

type ParamBuildUpdateSubAccountTx struct {
	TaskInfo              *tables.TableTaskInfo
	Account               *tables.TableAccountInfo
	AccountOutPoint       *types.OutPoint
	SubAccountOutpoint    *types.OutPoint
	SmtRecordInfoList     []tables.TableSmtRecordInfo
	Tree                  *smt.SmtServer
	BaseInfo              *BaseInfo
	SubAccountBuilderMap  map[string]*witness.SubAccountNew
	NewSubAccountPrice    uint64
	RenewSubAccountPrice  uint64
	BalanceDasLock        *types.Script
	BalanceDasType        *types.Script
	CommonFee             uint64
	SubAccountCellOutput  *types.CellOutput
	SubAccountOutputsData []byte
}

type ResultBuildUpdateSubAccountTx struct {
	DasTxBuilder          *txbuilder.DasTxBuilder
	SubAccountCellOutput  *types.CellOutput
	SubAccountOutputsData []byte
}

func (s *SubAccountTxTool) BuildUpdateSubAccountTx(p *ParamBuildUpdateSubAccountTx) (*ResultBuildUpdateSubAccountTx, error) {
	var res ResultBuildUpdateSubAccountTx
	txParams := &txbuilder.BuildTransactionParams{}

	// get sign_info
	signInfoResp, err := s.getSignInfo(p)
	if err != nil {
		return nil, err
	}

	// account price calculate
	subAccountPriceResp, err := s.getAccountPrice(p)
	if err != nil {
		return nil, err
	}

	// update smt get update smt response
	now := time.Now()
	updateSmtResp, err := s.updateSmt(p, signInfoResp, subAccountPriceResp)
	if err != nil {
		return nil, err
	}
	log.Info("UpdateSmt spend:", time.Since(now).Seconds())

	// cell deps
	s.cellDeps(txParams, p, updateSmtResp)

	// get sub_account cell output
	subAccCellOutputResp, err := s.subAccountCellInOutput(txParams, p, signInfoResp, subAccountPriceResp, updateSmtResp)
	if err != nil {
		return nil, err
	}
	res.SubAccountCellOutput = subAccCellOutputResp.subAccountCellOutput
	res.SubAccountOutputsData = subAccCellOutputResp.subAccountOutputsData

	// get balance normal or dp
	if err := s.getBalance(txParams, p, subAccountPriceResp, signInfoResp); err != nil {
		return nil, err
	}

	// account_cell and sub_account price rules witness
	if err := s.accountWitness(txParams, p); err != nil {
		return nil, err
	}

	//webauthn
	if err := s.webAuthn(txParams, p); err != nil {
		return nil, err
	}

	// build tx
	txBuilder := txbuilder.NewDasTxBuilderFromBase(s.TxBuilderBase, nil)
	if err := txBuilder.BuildTransaction(txParams); err != nil {
		return nil, fmt.Errorf("BuildTransaction err: %s", err.Error())
	}
	// header deps
	subTx, err := s.DasCore.Client().GetTransaction(s.Ctx, p.SubAccountOutpoint.TxHash)
	if err != nil {
		return nil, fmt.Errorf("GetTransaction err: %s", err.Error())
	}
	txBuilder.Transaction.HeaderDeps = append(txBuilder.Transaction.HeaderDeps, *subTx.TxStatus.BlockHash)

	// note: change fee
	sizeInBlock, _ := txBuilder.Transaction.SizeInBlock()
	changeCapacity := txBuilder.Transaction.Outputs[len(txBuilder.Transaction.Outputs)-1].Capacity
	changeCapacity = changeCapacity - sizeInBlock - 5000
	log.Infof("BuildCreateSubAccountTx txSize: %d", sizeInBlock)

	txBuilder.Transaction.Outputs[len(txBuilder.Transaction.Outputs)-1].Capacity = changeCapacity

	hash, err := txBuilder.Transaction.ComputeHash()
	if err != nil {
		return nil, fmt.Errorf("ComputeHash err: %s", err.Error())
	}

	log.Info("BuildUpdateSubAccountTx:", txBuilder.TxString(), hash.String())

	res.DasTxBuilder = txBuilder

	refOutpoint := common.OutPointStruct2String(p.SubAccountOutpoint)
	outpoint := common.OutPoint2String(hash.Hex(), 0)

	// update smt status
	if err := s.DbDao.UpdateSmtRecordOutpoint(p.TaskInfo.TaskId, refOutpoint, outpoint); err != nil {
		return nil, fmt.Errorf("UpdateSmtRecordOutpoint err: %s", err.Error())
	}
	return &res, nil
}

type SignInfoResp struct {
	managerDasLock  *types.Script
	witnessSignInfo map[string][]byte
	memSmtResp      map[string]*smt.UpdateSmtOut
}

type AccountPriceResp struct {
	registerTotalYears  uint64
	renewTotalYears     uint64
	autoTotalCapacity   uint64
	manualTotalCapacity uint64
	subAccountPriceMap  map[string]uint64
}

type UpdateSmtResp struct {
	accountCharTypeMap map[common.AccountCharType]struct{}
	subAccountList     []*witness.SubAccountNew
}

type SubAccountCellOutputResp struct {
	subAccountCellOutput  *types.CellOutput
	subAccountOutputsData []byte
}

func (s *SubAccountTxTool) getSignInfo(p *ParamBuildUpdateSubAccountTx) (*SignInfoResp, error) {
	resp := &SignInfoResp{
		witnessSignInfo: make(map[string][]byte),
		memSmtResp:      make(map[string]*smt.UpdateSmtOut),
	}
	mintSignIdMap := make(map[string]string)
	memKvs := make(map[string][]smt.SmtKv)

	for _, v := range p.SmtRecordInfoList {
		if v.MintSignId == "" {
			continue
		}

		if mintSignId := mintSignIdMap[v.SubAction]; mintSignId == "" {
			mintSignIdMap[v.SubAction] = v.MintSignId
		} else {
			if mintSignId != v.MintSignId {
				return nil, errors.New("mint sign id is different")
			}
			continue
		}

		mintSignInfo, err := s.DbDao.GetMinSignInfo(v.MintSignId)
		if err != nil {
			return nil, fmt.Errorf("GetMinSignInfo err: %s", err.Error())
		}
		resp.witnessSignInfo[v.SubAction] = mintSignInfo.GenWitness()

		var listKeyValue []tables.MintSignInfoKeyValue
		err = json.Unmarshal([]byte(mintSignInfo.KeyValue), &listKeyValue)
		if err != nil {
			return nil, fmt.Errorf("KeyValue of table mint_sign_info is not a json string: %s", err.Error())
		}
		for _, kv := range listKeyValue {
			smtKey := smt.AccountIdToSmtH256(kv.Key)
			smtValue, err := blake2b.Blake256(common.Hex2Bytes(kv.Value))
			if err != nil {
				return nil, fmt.Errorf("blake2b.Blake256 err: %s", err.Error())
			}
			memKvs[v.SubAction] = append(memKvs[v.SubAction], smt.SmtKv{
				Key:   smtKey,
				Value: smtValue,
			})
		}

		resp.managerDasLock, _, err = s.DasCore.Daf().HexToScript(core.DasAddressHex{
			DasAlgorithmId: mintSignInfo.ChainType.ToDasAlgorithmId(true),
			AddressHex:     mintSignInfo.Address,
			ChainType:      mintSignInfo.ChainType,
		})
		if err != nil {
			return nil, fmt.Errorf("manager HexToScript err: %s", err.Error())
		}
	}

	signTree := smt.NewSmtSrv(p.Tree.GetSmtUrl(), "")
	opt := smt.SmtOpt{GetProof: true, GetRoot: true}
	for k, v := range memKvs {
		memRep, err := signTree.UpdateSmt(v, opt)
		if err != nil {
			return nil, fmt.Errorf("mintSignTree.Update err: %s", err.Error())
		}
		resp.memSmtResp[k] = memRep
	}
	return resp, nil
}

func (s *SubAccountTxTool) getAccountPrice(p *ParamBuildUpdateSubAccountTx) (*AccountPriceResp, error) {
	resp := &AccountPriceResp{
		subAccountPriceMap: make(map[string]uint64),
	}
	quote := p.BaseInfo.QuoteCell.Quote()

	for _, v := range p.SmtRecordInfoList {
		if v.SubAction != common.SubActionCreate &&
			v.SubAction != common.SubActionRenew {
			continue
		}

		var yearlyPrice uint64
		switch v.MintType {
		case tables.MintTypeDefault, tables.MintTypeManual:
			switch v.SubAction {
			case common.SubActionCreate:
				resp.registerTotalYears += v.RegisterYears
				yearlyPrice = p.NewSubAccountPrice
			case common.SubActionRenew:
				resp.renewTotalYears += v.RenewYears
				yearlyPrice = p.RenewSubAccountPrice
			}
		case tables.MintTypeAutoMint:
			subAccountCell, err := s.getSubAccountCell(v.ParentAccountId)
			if err != nil {
				return nil, err
			}
			subAccountTx, err := s.DasCore.Client().GetTransaction(s.Ctx, subAccountCell.OutPoint.TxHash)
			if err != nil {
				return nil, err
			}
			parentAccountInfo, err := s.DbDao.GetAccountInfoByAccountId(v.ParentAccountId)
			if err != nil {
				return nil, err
			}
			subAccountRule := witness.NewSubAccountRuleEntity(parentAccountInfo.Account)
			if err := subAccountRule.ParseFromTx(subAccountTx.Transaction, common.ActionDataTypeSubAccountPriceRules); err != nil {
				return nil, err
			}
			hit, idx, err := subAccountRule.Hit(v.Account)
			if err != nil {
				return nil, err
			}
			if !hit {
				return nil, fmt.Errorf("%s not hit any price rule", v.Account)
			}
			yearlyPrice = uint64(subAccountRule.Rules[idx].Price)
		}

		subAccCapacity := config.PriceToCKB(yearlyPrice, quote, v.RegisterYears+v.RenewYears)

		switch v.MintType {
		case tables.MintTypeDefault, tables.MintTypeManual:
			resp.manualTotalCapacity += subAccCapacity
		case tables.MintTypeAutoMint:
			resp.autoTotalCapacity += subAccCapacity
			resp.subAccountPriceMap[v.AccountId] = subAccCapacity
		}
	}
	return resp, nil
}

func (s *SubAccountTxTool) updateSmt(p *ParamBuildUpdateSubAccountTx, signInfo *SignInfoResp, accountPrice *AccountPriceResp) (*UpdateSmtResp, error) {
	resp := &UpdateSmtResp{
		accountCharTypeMap: make(map[common.AccountCharType]struct{}),
		subAccountList:     make([]*witness.SubAccountNew, 0),
	}

	// smt record
	if err := s.DbDao.UpdateSmtStatus(p.TaskInfo.TaskId, tables.SmtStatusWriting); err != nil {
		return nil, fmt.Errorf("UpdateSmtStatus err: %s", err.Error())
	}
	var smtKv []smt.SmtKv
	subAccountIdMap := make(map[int]string)

	for i, v := range p.SmtRecordInfoList {
		log.Info("BuildUpdateSubAccountTx:", v.TaskId, len(p.SmtRecordInfoList), "-", i)

		var oldSubAccount *witness.SubAccountNew
		if v.SubAction != common.SubActionCreate {
			subAccountBuilder, ok := p.SubAccountBuilderMap[v.AccountId]
			if !ok {
				return nil, fmt.Errorf("SubAccountBuilderMap not exist: %s", v.AccountId)
			}
			oldSubAccount = subAccountBuilder
		}

		timeCellTimestamp := p.BaseInfo.TimeCell.Timestamp()
		subAccountData, subAccountNew, err := p.SmtRecordInfoList[i].GetCurrentSubAccountNew(s.DasCore, oldSubAccount, p.BaseInfo.ContractDas, timeCellTimestamp)
		if err != nil {
			return nil, fmt.Errorf("GetCurrentSubAccountNew err: %s", err.Error())
		}

		value, err := subAccountData.ToH256()
		if err != nil {
			return nil, err
		}
		smtKv = append(smtKv, smt.SmtKv{
			Key:   smt.AccountIdToSmtH256(v.AccountId),
			Value: value,
		})
		common.GetAccountCharType(resp.accountCharTypeMap, subAccountData.AccountCharSet)
		subAccountIdMap[len(resp.subAccountList)] = v.AccountId
		resp.subAccountList = append(resp.subAccountList, subAccountNew)

		switch v.SubAction {
		case common.SubActionCreate:
			if len(signInfo.witnessSignInfo[v.SubAction]) > 0 && v.MintSignId != "" {
				smtKey := smt.AccountIdToSmtH256(v.AccountId)
				signProof, ok := signInfo.memSmtResp[v.SubAction].Proofs[smtKey.String()]
				if !ok {
					return nil, fmt.Errorf("mintSignTree.MerkleProof err: proof is not found : %s", smtKey.String())
				}
				subAccountNew.EditValue = common.Hex2Bytes(signProof)
			}
			if v.MintType == tables.MintTypeAutoMint {
				subAccountNew.EditKey = common.EditKeyCustomRule
				subAccountNew.EditValue = s.ServerScript.Args
				subAccountPrice, ok := accountPrice.subAccountPriceMap[v.AccountId]
				if !ok {
					return nil, errors.New("data abnormal")
				}
				price := molecule.GoU64ToMoleculeU64(subAccountPrice)
				subAccountNew.EditValue = append(subAccountNew.EditValue, price.AsSlice()...)
			}
		case common.SubActionRenew:
			subAccountNew.EditKey = common.EditKeyManual
			if len(signInfo.witnessSignInfo[v.SubAction]) > 0 && v.MintSignId != "" {
				smtKey := smt.AccountIdToSmtH256(v.AccountId)
				renewSignProof, ok := signInfo.memSmtResp[v.SubAction].Proofs[smtKey.String()]
				if !ok {
					return nil, fmt.Errorf("mintSignTree.MerkleProof err: proof is not found : %s", smtKey.String())
				}
				subAccountNew.EditValue = append(subAccountNew.EditValue, common.Hex2Bytes(renewSignProof)...)
			}
			if v.MintType == tables.MintTypeAutoMint {
				subAccountNew.EditKey = common.EditKeyCustomRule
				subAccountNew.EditValue = append(subAccountNew.EditValue, s.ServerScript.Args...)
				subAccountPrice, ok := accountPrice.subAccountPriceMap[v.AccountId]
				if !ok {
					return nil, errors.New("data abnormal")
				}
				price := molecule.GoU64ToMoleculeU64(subAccountPrice)
				subAccountNew.EditValue = append(subAccountNew.EditValue, price.AsSlice()...)
			}
		}
	}

	if len(smtKv) > 0 {
		opt := smt.SmtOpt{GetProof: true, GetRoot: true}
		smtRes, err := p.Tree.UpdateMiddleSmt(smtKv, opt)
		if err != nil {
			return nil, fmt.Errorf("tree.Update err: %s", err.Error())
		}
		for i := range resp.subAccountList {
			key := common.Bytes2Hex(smt.AccountIdToSmtH256(subAccountIdMap[i]))
			pprof, ok := smtRes.Proofs[key]
			if !ok {
				return nil, fmt.Errorf("tree.MerkleProof Proof err: %s", smtRes.Proofs)
			}
			newRoot, ok := smtRes.Roots[key]
			if !ok {
				return nil, fmt.Errorf("tree.Roof err: %s", smtRes.Roots)
			}
			resp.subAccountList[i].Proof = common.Hex2Bytes(pprof)
			resp.subAccountList[i].NewRoot = newRoot
		}
	}
	return resp, nil
}

func (s *SubAccountTxTool) subAccountCellInOutput(txParams *txbuilder.BuildTransactionParams, p *ParamBuildUpdateSubAccountTx, signInfo *SignInfoResp, accountPrice *AccountPriceResp, updateSmtResp *UpdateSmtResp) (*SubAccountCellOutputResp, error) {
	resp := &SubAccountCellOutputResp{}

	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: p.SubAccountOutpoint,
	})
	resp.subAccountCellOutput = &types.CellOutput{
		Capacity: p.SubAccountCellOutput.Capacity + accountPrice.manualTotalCapacity + accountPrice.autoTotalCapacity,
		Lock:     p.SubAccountCellOutput.Lock,
		Type:     p.SubAccountCellOutput.Type,
	}
	txParams.Outputs = append(txParams.Outputs, resp.subAccountCellOutput)

	subDataDetail := witness.ConvertSubAccountCellOutputData(p.SubAccountOutputsData)
	if len(updateSmtResp.subAccountList) > 0 {
		subDataDetail.SmtRoot = updateSmtResp.subAccountList[len(updateSmtResp.subAccountList)-1].NewRoot
	}
	subDataDetail.DasProfit += accountPrice.manualTotalCapacity + accountPrice.autoTotalCapacity
	resp.subAccountOutputsData = witness.BuildSubAccountCellOutputData(subDataDetail)
	txParams.OutputsData = append(txParams.OutputsData, resp.subAccountOutputsData)

	// witness
	actionWitness, err := witness.GenActionDataWitnessV2(common.DasActionUpdateSubAccount, nil, "")
	if err != nil {
		return nil, fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)
	for _, v := range signInfo.witnessSignInfo {
		txParams.Witnesses = append(txParams.Witnesses, v)
	}
	smtWitnessList, _ := getSubAccountWitness(updateSmtResp.subAccountList)
	for _, v := range smtWitnessList {
		txParams.Witnesses = append(txParams.Witnesses, v)
	}
	return resp, nil
}

func (s *SubAccountTxTool) getBalance(txParams *txbuilder.BuildTransactionParams, p *ParamBuildUpdateSubAccountTx, accountPrice *AccountPriceResp, signInfoResp *SignInfoResp) error {
	manualPrice := p.NewSubAccountPrice*accountPrice.registerTotalYears + p.RenewSubAccountPrice*accountPrice.renewTotalYears
	if manualPrice > 0 {
		// manager payment dp
		manualDpLiveCells, manualTotalAmount, manualTotalCapacity, err := s.DasCore.GetDpCells(&core.ParamGetDpCells{
			DasCache:    s.DasCache,
			LockScript:  signInfoResp.managerDasLock,
			AmountNeed:  manualPrice,
			SearchOrder: indexer.SearchOrderAsc,
		})
		for _, v := range manualDpLiveCells {
			txParams.Inputs = append(txParams.Inputs, &types.CellInput{
				PreviousOutput: v.OutPoint,
			})
		}

		transferWhiteList, err := address.Parse(config.Cfg.Das.Dp.TransferWhiteList)
		if err != nil {
			return fmt.Errorf("parse dp recycle whitelist err: %s", err.Error())
		}
		capacityWhitelist, err := address.Parse(config.Cfg.Das.Dp.CapacityWhitelist)
		if err != nil {
			return fmt.Errorf("parse dp capacity whitelist err: %s", err.Error())
		}

		// split dp
		dpOutputCells, dpOutputData, replenishNormal, err := s.DasCore.SplitDPCell(&core.ParamSplitDPCell{
			FromLock:           signInfoResp.managerDasLock,
			ToLock:             transferWhiteList.Script,
			DPLiveCell:         manualDpLiveCells,
			DPLiveCellCapacity: manualTotalCapacity,
			DPTotalAmount:      manualTotalAmount,
			DPTransferAmount:   manualPrice,
			DPSplitCount:       10,
			DPSplitAmount:      25 * common.UsdRateBase,
			NormalCellLock:     capacityWhitelist.Script,
		})
		if err != nil {
			return fmt.Errorf("SplitDPCell err: %s", err.Error())
		}
		txParams.Outputs = append(txParams.Outputs, dpOutputCells...)
		txParams.OutputsData = append(txParams.OutputsData, dpOutputData...)

		// provider normal cell
		manualChange, manualBalanceLiveCells, err := s.GetBalanceCell(&ParamBalance{
			DasLock:      p.BalanceDasLock,
			DasType:      p.BalanceDasType,
			NeedCapacity: accountPrice.manualTotalCapacity + replenishNormal + p.CommonFee,
		})
		if err != nil {
			log.Info("UpdateTaskStatusToRollbackWithBalanceErr:", p.TaskInfo.TaskId)
			_ = s.DbDao.UpdateTaskStatusToRollbackWithBalanceErr(p.TaskInfo.TaskId)
			return fmt.Errorf("getBalanceCell err: %s", err.Error())
		}
		manualChange += p.CommonFee
		// balance input
		for _, v := range manualBalanceLiveCells {
			txParams.Inputs = append(txParams.Inputs, &types.CellInput{
				PreviousOutput: v.OutPoint,
			})
		}
		// split out balance
		if manualChange > 0 {
			base := 1000 * common.OneCkb
			splitList, err := core.SplitOutputCell2(manualChange, base, 10, p.BalanceDasLock, p.BalanceDasType, indexer.SearchOrderAsc)
			if err != nil {
				return fmt.Errorf("SplitOutputCell2 err: %s", err.Error())
			}
			for i := range splitList {
				txParams.Outputs = append(txParams.Outputs, splitList[i])
				txParams.OutputsData = append(txParams.OutputsData, []byte{})
			}
		}
	}

	if accountPrice.autoTotalCapacity > 0 {
		needCapacity := accountPrice.autoTotalCapacity
		if manualPrice == 0 {
			needCapacity += p.CommonFee
		}
		autoChange, autoBalanceLiveCells, err := s.GetBalanceCell(&ParamBalance{
			DasLock:      p.BalanceDasLock,
			DasType:      p.BalanceDasType,
			NeedCapacity: needCapacity,
		})
		if err != nil {
			log.Info("UpdateTaskStatusToRollbackWithBalanceErr:", p.TaskInfo.TaskId)
			_ = s.DbDao.UpdateTaskStatusToRollbackWithBalanceErr(p.TaskInfo.TaskId)
			return fmt.Errorf("getBalanceCell err: %s", err.Error())
		}
		if manualPrice == 0 {
			autoChange += p.CommonFee
		}
		for _, v := range autoBalanceLiveCells {
			txParams.Inputs = append(txParams.Inputs, &types.CellInput{
				PreviousOutput: v.OutPoint,
			})
		}
		if autoChange > 0 {
			base := 1000 * common.OneCkb
			splitList, err := core.SplitOutputCell2(autoChange, base, 10, p.BalanceDasLock, p.BalanceDasType, indexer.SearchOrderAsc)
			if err != nil {
				return fmt.Errorf("SplitOutputCell2 err: %s", err.Error())
			}
			for i := range splitList {
				txParams.Outputs = append(txParams.Outputs, splitList[i])
				txParams.OutputsData = append(txParams.OutputsData, []byte{})
			}
		}
	}
	return nil
}

func (s *SubAccountTxTool) accountWitness(txParams *txbuilder.BuildTransactionParams, p *ParamBuildUpdateSubAccountTx) error {
	// account cell witness
	accTx, err := s.DasCore.Client().GetTransaction(s.Ctx, p.AccountOutPoint.TxHash)
	if err != nil {
		return fmt.Errorf("GetTransaction acc tx err: %s", err.Error())
	}
	accBuilderMap, err := witness.AccountIdCellDataBuilderFromTx(accTx.Transaction, common.DataTypeNew)
	if err != nil {
		return fmt.Errorf("AccountIdCellDataBuilderFromTx err: %s", err.Error())
	}
	accBuilder, ok := accBuilderMap[p.Account.AccountId]
	if !ok {
		return fmt.Errorf("accBuilderMap is nil: %s", p.Account.AccountId)
	}
	accWitness, _, _ := accBuilder.GenWitness(&witness.AccountCellParam{
		Action: common.DasActionUpdateSubAccount,
	})
	txParams.Witnesses = append(txParams.Witnesses, accWitness)

	// sub_account_cell custom rule witness
	subAccountTx, err := s.DasCore.Client().GetTransaction(s.Ctx, p.SubAccountOutpoint.TxHash)
	if err != nil {
		return err
	}

	if err := witness.GetWitnessDataFromTx(subAccountTx.Transaction, func(actionDataType common.ActionDataType, dataBys []byte, index int) (bool, error) {
		if actionDataType == common.ActionDataTypeSubAccountPriceRules || actionDataType == common.ActionDataTypeSubAccountPreservedRules {
			witnessBytes := witness.GenDasDataWitnessWithByte(actionDataType, dataBys)
			txParams.Witnesses = append(txParams.Witnesses, witnessBytes)
		}
		return true, nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *SubAccountTxTool) cellDeps(txParams *txbuilder.BuildTransactionParams, p *ParamBuildUpdateSubAccountTx, updateSmtResp *UpdateSmtResp) {
	txParams.CellDeps = append(txParams.CellDeps,
		&types.CellDep{
			OutPoint: p.AccountOutPoint,
			DepType:  types.DepTypeCode,
		},
		p.BaseInfo.QuoteCell.ToCellDep(),
		p.BaseInfo.ContractDas.ToCellDep(),
		p.BaseInfo.ContractAcc.ToCellDep(),
		p.BaseInfo.ContractSubAcc.ToCellDep(),
		p.BaseInfo.HeightCell.ToCellDep(),
		p.BaseInfo.TimeCell.ToCellDep(),
		p.BaseInfo.ConfigCellAcc.ToCellDep(),
		p.BaseInfo.ConfigCellSubAcc.ToCellDep(),
		p.BaseInfo.ConfigCellRecordNamespace.ToCellDep(),
		p.BaseInfo.ConfigCellDPoint.ToCellDep(),
	)
	for k := range updateSmtResp.accountCharTypeMap {
		switch k {
		case common.AccountCharTypeEmoji:
			txParams.CellDeps = append(txParams.CellDeps, p.BaseInfo.ConfigCellEmoji.ToCellDep())
		case common.AccountCharTypeDigit:
			txParams.CellDeps = append(txParams.CellDeps, p.BaseInfo.ConfigCellDigit.ToCellDep())
		case common.AccountCharTypeEn:
			txParams.CellDeps = append(txParams.CellDeps, p.BaseInfo.ConfigCellEn.ToCellDep())
		case common.AccountCharTypeJa:
			txParams.CellDeps = append(txParams.CellDeps, p.BaseInfo.ConfigCellJa.ToCellDep())
		case common.AccountCharTypeRu:
			txParams.CellDeps = append(txParams.CellDeps, p.BaseInfo.ConfigCellRu.ToCellDep())
		case common.AccountCharTypeTr:
			txParams.CellDeps = append(txParams.CellDeps, p.BaseInfo.ConfigCellTr.ToCellDep())
		case common.AccountCharTypeVi:
			txParams.CellDeps = append(txParams.CellDeps, p.BaseInfo.ConfigCellVi.ToCellDep())
		case common.AccountCharTypeKo:
			txParams.CellDeps = append(txParams.CellDeps, p.BaseInfo.ConfigCellKo.ToCellDep())
		case common.AccountCharTypeTh:
			txParams.CellDeps = append(txParams.CellDeps, p.BaseInfo.ConfigCellTh.ToCellDep())
		}
	}
}

func (s *SubAccountTxTool) webAuthn(txParams *txbuilder.BuildTransactionParams, p *ParamBuildUpdateSubAccountTx) error {
	keyListMap := make(map[string]bool)
	for _, v := range p.SmtRecordInfoList {
		if v.LoginAddress == v.SignAddress || v.LoginChainType != common.ChainTypeWebauthn {
			continue
		}
		addrHex := core.DasAddressHex{
			DasAlgorithmId:    common.DasAlgorithmIdWebauthn,
			DasSubAlgorithmId: common.DasWebauthnSubAlgorithmIdES256,
			AddressHex:        v.LoginAddress,
			AddressPayload:    common.Hex2Bytes(v.LoginAddress),
			ChainType:         common.ChainTypeWebauthn,
		}
		lockArgs, err := s.DasCore.Daf().HexToArgs(addrHex, addrHex)
		keyListConfigCellContract, err := core.GetDasContractInfo(common.DasKeyListCellType)
		if err != nil {
			return fmt.Errorf("GetDasContractInfo err: %s", err.Error())
		}

		txParams.CellDeps = append(txParams.CellDeps, keyListConfigCellContract.ToCellDep())
		cell, err := s.DasCore.GetKeyListCell(lockArgs)
		if err != nil {
			return fmt.Errorf("GetKeyListCell(webauthn keyListCell) : %s", err.Error())
		}
		//has enable authorize
		if cell != nil {
			if _, ok := keyListMap[cell.OutPoint.TxHash.Hex()]; ok {
				continue
			}
			txParams.CellDeps = append(txParams.CellDeps, &types.CellDep{
				OutPoint: cell.OutPoint,
				DepType:  types.DepTypeCode,
			})
			//2. add master device keyList witness
			keyListConfigTx, err := s.DasCore.Client().GetTransaction(s.Ctx, cell.OutPoint.TxHash)
			if err != nil {
				return fmt.Errorf("GetTransaction err : %s", err.Error())
			}
			webAuthnKeyListConfigBuilder, err := witness.WebAuthnKeyListDataBuilderFromTx(keyListConfigTx.Transaction, common.DataTypeNew)
			if err != nil {
				return fmt.Errorf("WebAuthnKeyListDataBuilderFromTx err : %s", err.Error())
			}
			webAuthnKeyListConfigBuilder.DataEntityOpt.AsSlice()
			tmp := webAuthnKeyListConfigBuilder.DeviceKeyListCellData.AsSlice()
			keyListWitness := witness.GenDasDataWitnessWithByte(common.ActionDataTypeKeyListCfgCellData, tmp)
			txParams.OtherWitnesses = append(txParams.OtherWitnesses, keyListWitness)
			keyListMap[cell.OutPoint.TxHash.Hex()] = true
		}
	}
	return nil
}

func (s *SubAccountTxTool) getSubAccountCell(parentAccountId string) (*indexer.LiveCell, error) {
	baseInfo, err := s.GetBaseInfo()
	if err != nil {
		return nil, err
	}
	searchKey := indexer.SearchKey{
		Script:     baseInfo.ContractSubAcc.ToScript(common.Hex2Bytes(parentAccountId)),
		ScriptType: indexer.ScriptTypeType,
		ArgsLen:    0,
		Filter:     nil,
	}
	liveCell, err := s.DasCore.Client().GetCells(s.Ctx, &searchKey, indexer.SearchOrderDesc, 1, "")
	if err != nil {
		return nil, fmt.Errorf("GetCells err: %s", err.Error())
	}
	if subLen := len(liveCell.Objects); subLen != 1 {
		return nil, fmt.Errorf("sub account outpoint len: %d", subLen)
	}
	return liveCell.Objects[0], nil
}
