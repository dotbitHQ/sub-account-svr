package txtool

import (
	"bytes"
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
	var txParams txbuilder.BuildTransactionParams
	var res ResultBuildUpdateSubAccountTx

	var balanceDasLock *types.Script
	var balanceDasType *types.Script
	mintSignIdMap := make(map[string]string)
	witnessSignInfo := make(map[string][]byte)
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
		witnessSignInfo[v.SubAction] = mintSignInfo.GenWitness()

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
		balanceDasLock, balanceDasType, err = s.DasCore.Daf().HexToScript(core.DasAddressHex{
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
	smtMemoryRep := make(map[string]*smt.UpdateSmtOut)
	for k, v := range memKvs {
		memRep, err := signTree.UpdateSmt(v, opt)
		if err != nil {
			return nil, fmt.Errorf("mintSignTree.Update err: %s", err.Error())
		}
		smtMemoryRep[k] = memRep
	}

	registerTotalYears := uint64(0)
	renewTotalYears := uint64(0)
	autoTotalCapacity := uint64(0)
	subAccountPriceMap := make(map[string]uint64)
	for _, v := range p.SmtRecordInfoList {
		if v.SubAction != common.SubActionCreate &&
			v.SubAction != common.SubActionRenew {
			continue
		}

		switch v.MintType {
		case tables.MintTypeDefault, tables.MintTypeManual:
			switch v.SubAction {
			case common.SubActionCreate:
				registerTotalYears += v.RegisterYears
			case common.SubActionRenew:
				renewTotalYears += v.RenewYears
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

			quote := p.BaseInfo.QuoteCell.Quote()
			yearlyPrice := uint64(subAccountRule.Rules[idx].Price)
			subAccountPrice := uint64(0)
			years := uint64(0)
			switch v.SubAction {
			case common.SubActionCreate:
				years = v.RegisterYears
			case common.SubActionRenew:
				years = v.RenewYears
			}
			if yearlyPrice < quote {
				subAccountPrice = yearlyPrice * common.OneCkb / quote * years
			} else {
				subAccountPrice = yearlyPrice / quote * common.OneCkb * years
			}
			autoTotalCapacity += subAccountPrice
			subAccountPriceMap[v.AccountId] = subAccountPrice
		}
	}

	var err error
	var manualChange uint64
	manualBalanceLiveCells := make([]*indexer.LiveCell, 0)
	manualCapacity := p.NewSubAccountPrice*registerTotalYears + p.RenewSubAccountPrice*renewTotalYears
	if manualCapacity > 0 {
		needCapacity := manualCapacity
		if autoTotalCapacity == 0 {
			needCapacity += p.CommonFee
		}
		manualChange, manualBalanceLiveCells, err = s.GetBalanceCell(&ParamBalance{
			DasLock:      balanceDasLock,
			DasType:      balanceDasType,
			NeedCapacity: needCapacity,
		})
		if err != nil {
			log.Info("UpdateTaskStatusToRollbackWithBalanceErr:", p.TaskInfo.TaskId)
			_ = s.DbDao.UpdateTaskStatusToRollbackWithBalanceErr(p.TaskInfo.TaskId)
			return nil, fmt.Errorf("getBalanceCell err: %s", err.Error())
		}
		if autoTotalCapacity == 0 {
			manualChange += p.CommonFee
		}
	}

	var autoChange uint64
	autoBalanceLiveCells := make([]*indexer.LiveCell, 0)
	if autoTotalCapacity > 0 {
		autoChange, autoBalanceLiveCells, err = s.GetBalanceCell(&ParamBalance{
			DasLock:      p.BalanceDasLock,
			DasType:      p.BalanceDasType,
			NeedCapacity: autoTotalCapacity + p.CommonFee,
		})
		if err != nil {
			log.Info("UpdateTaskStatusToRollbackWithBalanceErr:", p.TaskInfo.TaskId)
			_ = s.DbDao.UpdateTaskStatusToRollbackWithBalanceErr(p.TaskInfo.TaskId)
			return nil, fmt.Errorf("getBalanceCell err: %s", err.Error())
		}
		autoChange += p.CommonFee
	}

	balanceLiveCells := make([]*indexer.LiveCell, 0)
	if manualChange == 0 && autoChange == 0 {
		_, balanceLiveCells, err = s.GetBalanceCell(&ParamBalance{
			DasLock:      p.BalanceDasLock,
			DasType:      p.BalanceDasType,
			NeedCapacity: common.OneCkb,
		})
		if err != nil {
			log.Info("UpdateTaskStatusToRollbackWithBalanceErr:", p.TaskInfo.TaskId)
			_ = s.DbDao.UpdateTaskStatusToRollbackWithBalanceErr(p.TaskInfo.TaskId)
			return nil, fmt.Errorf("getBalanceCell err: %s", err.Error())
		}
	}

	// update smt status
	if err := s.DbDao.UpdateSmtStatus(p.TaskInfo.TaskId, tables.SmtStatusWriting); err != nil {
		return nil, fmt.Errorf("UpdateSmtStatus err: %s", err.Error())
	}

	// smt record
	var accountCharTypeMap = make(map[common.AccountCharType]struct{})
	var subAccountList []*witness.SubAccountNew
	var smtKv []smt.SmtKv
	subAccountIdMap := make(map[int]string)
	time1 := time.Now()

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
		common.GetAccountCharType(accountCharTypeMap, subAccountData.AccountCharSet)
		subAccountIdMap[len(subAccountList)] = v.AccountId
		subAccountList = append(subAccountList, subAccountNew)

		switch v.SubAction {
		case common.SubActionCreate:
			if len(witnessSignInfo[v.SubAction]) > 0 && v.MintSignId != "" {
				smtKey := smt.AccountIdToSmtH256(v.AccountId)
				signProof, ok := smtMemoryRep[v.SubAction].Proofs[smtKey.String()]
				if !ok {
					return nil, fmt.Errorf("mintSignTree.MerkleProof err: proof is not found : %s", smtKey.String())
				}
				subAccountNew.EditValue = common.Hex2Bytes(signProof)
			}
			if v.MintType == tables.MintTypeAutoMint {
				subAccountNew.EditKey = common.EditKeyCustomRule
				subAccountNew.EditValue = s.ServerScript.Args
				subAccountPrice, ok := subAccountPriceMap[v.AccountId]
				if !ok {
					return nil, errors.New("data abnormal")
				}
				price := molecule.GoU64ToMoleculeU64(subAccountPrice)
				subAccountNew.EditValue = append(subAccountNew.EditValue, price.AsSlice()...)
			}
		case common.SubActionRenew:
			subAccountNew.EditKey = common.EditKeyManual
			if len(witnessSignInfo[v.SubAction]) > 0 && v.MintSignId != "" {
				smtKey := smt.AccountIdToSmtH256(v.AccountId)
				renewSignProof, ok := smtMemoryRep[v.SubAction].Proofs[smtKey.String()]
				if !ok {
					return nil, fmt.Errorf("mintSignTree.MerkleProof err: proof is not found : %s", smtKey.String())
				}
				subAccountNew.EditValue = append(subAccountNew.EditValue, common.Hex2Bytes(renewSignProof)...)
			}
			if v.MintType == tables.MintTypeAutoMint {
				subAccountNew.EditKey = common.EditKeyCustomRule
				subAccountNew.EditValue = append(subAccountNew.EditValue, s.ServerScript.Args...)
				subAccountPrice, ok := subAccountPriceMap[v.AccountId]
				if !ok {
					return nil, errors.New("data abnormal")
				}
				price := molecule.GoU64ToMoleculeU64(subAccountPrice)
				subAccountNew.EditValue = append(subAccountNew.EditValue, price.AsSlice()...)
			}
		}
	}

	if len(smtKv) > 0 {
		smtRes, err := p.Tree.UpdateMiddleSmt(smtKv, opt)
		if err != nil {
			return nil, fmt.Errorf("tree.Update err: %s", err.Error())
		}
		for i := range subAccountList {
			key := common.Bytes2Hex(smt.AccountIdToSmtH256(subAccountIdMap[i]))
			pprof, ok := smtRes.Proofs[key]
			if !ok {
				return nil, fmt.Errorf("tree.MerkleProof Proof err: %s", smtRes.Proofs)
			}
			newRoot, ok := smtRes.Roots[key]
			if !ok {
				return nil, fmt.Errorf("tree.Roof err: %s", smtRes.Roots)
			}
			subAccountList[i].Proof = common.Hex2Bytes(pprof)
			subAccountList[i].NewRoot = newRoot
		}
	}

	log.Info("SmtRecordInfoList spend:", time.Since(time1).Seconds())
	// inputs
	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: p.SubAccountOutpoint,
	})

	// get balance cell
	for _, v := range manualBalanceLiveCells {
		txParams.Inputs = append(txParams.Inputs, &types.CellInput{
			PreviousOutput: v.OutPoint,
		})
	}
	for _, v := range autoBalanceLiveCells {
		txParams.Inputs = append(txParams.Inputs, &types.CellInput{
			PreviousOutput: v.OutPoint,
		})
	}
	for _, v := range balanceLiveCells {
		txParams.Inputs = append(txParams.Inputs, &types.CellInput{
			PreviousOutput: v.OutPoint,
		})
	}

	// outputs
	res.SubAccountCellOutput = &types.CellOutput{
		Capacity: p.SubAccountCellOutput.Capacity + manualCapacity + autoTotalCapacity,
		Lock:     p.SubAccountCellOutput.Lock,
		Type:     p.SubAccountCellOutput.Type,
	}
	txParams.Outputs = append(txParams.Outputs, res.SubAccountCellOutput) // sub_account
	// root+profit
	subDataDetail := witness.ConvertSubAccountCellOutputData(p.SubAccountOutputsData)
	if len(subAccountList) > 0 {
		subDataDetail.SmtRoot = subAccountList[len(subAccountList)-1].NewRoot
	}
	subDataDetail.DasProfit += manualCapacity + autoTotalCapacity
	res.SubAccountOutputsData = witness.BuildSubAccountCellOutputData(subDataDetail)
	txParams.OutputsData = append(txParams.OutputsData, res.SubAccountOutputsData) // smt root

	// change
	if manualChange > 0 {
		txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
			Capacity: manualChange,
			Lock:     balanceDasLock,
			Type:     balanceDasType,
		})
		txParams.OutputsData = append(txParams.OutputsData, []byte{})
	}
	if autoChange > 0 {
		base := 1000 * common.OneCkb
		splitList, err := core.SplitOutputCell2(autoChange, base, 10, p.BalanceDasLock, p.BalanceDasType, indexer.SearchOrderAsc)
		if err != nil {
			return nil, fmt.Errorf("SplitOutputCell2 err: %s", err.Error())
		}
		for i := range splitList {
			txParams.Outputs = append(txParams.Outputs, splitList[i])
			txParams.OutputsData = append(txParams.OutputsData, []byte{})
		}
	}
	for _, v := range balanceLiveCells {
		txParams.Outputs = append(txParams.Outputs, v.Output)
		txParams.OutputsData = append(txParams.OutputsData, []byte{})
	}

	// witness
	actionWitness, err := witness.GenActionDataWitnessV2(common.DasActionUpdateSubAccount, nil, "")
	if err != nil {
		return nil, fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	// sign info
	for _, v := range witnessSignInfo {
		txParams.Witnesses = append(txParams.Witnesses, v)
	}

	// smt
	smtWitnessList, _ := getSubAccountWitness(subAccountList)
	for _, v := range smtWitnessList {
		txParams.Witnesses = append(txParams.Witnesses, v)
	}

	// account cell witness
	accTx, err := s.DasCore.Client().GetTransaction(s.Ctx, p.AccountOutPoint.TxHash)
	if err != nil {
		return nil, fmt.Errorf("GetTransaction acc tx err: %s", err.Error())
	}
	accBuilderMap, err := witness.AccountIdCellDataBuilderFromTx(accTx.Transaction, common.DataTypeNew)
	if err != nil {
		return nil, fmt.Errorf("AccountIdCellDataBuilderFromTx err: %s", err.Error())
	}
	accBuilder, ok := accBuilderMap[p.Account.AccountId]
	if !ok {
		return nil, fmt.Errorf("accBuilderMap is nil: %s", p.Account.AccountId)
	}
	accWitness, _, _ := accBuilder.GenWitness(&witness.AccountCellParam{
		OldIndex: 0,
		Action:   common.DasActionUpdateSubAccount,
	})
	txParams.Witnesses = append(txParams.Witnesses, accWitness)

	// sub_account_cell custom rule witness
	subAccountTx, err := s.DasCore.Client().GetTransaction(s.Ctx, p.SubAccountOutpoint.TxHash)
	if err != nil {
		return nil, err
	}

	rulesSize := 0
	if err := witness.GetWitnessDataFromTx(subAccountTx.Transaction, func(actionDataType common.ActionDataType, dataBys []byte, index int) (bool, error) {
		if actionDataType == common.ActionDataTypeSubAccountPriceRules || actionDataType == common.ActionDataTypeSubAccountPreservedRules {
			witnessBytes := witness.GenDasDataWitnessWithByte(actionDataType, dataBys)
			rulesSize += len(witnessBytes)
			txParams.Witnesses = append(txParams.Witnesses, witnessBytes)
		}
		return true, nil
	}); err != nil {
		return nil, err
	}

	txParams.CellDeps = append(txParams.CellDeps,
		&types.CellDep{
			OutPoint: p.AccountOutPoint,
			DepType:  types.DepTypeCode,
		},
		p.BaseInfo.ContractDas.ToCellDep(),
		p.BaseInfo.ContractAcc.ToCellDep(),
		p.BaseInfo.ContractSubAcc.ToCellDep(),
		p.BaseInfo.QuoteCell.ToCellDep(),
		p.BaseInfo.HeightCell.ToCellDep(),
		p.BaseInfo.TimeCell.ToCellDep(),
		p.BaseInfo.ConfigCellAcc.ToCellDep(),
		p.BaseInfo.ConfigCellSubAcc.ToCellDep(),
		p.BaseInfo.ConfigCellRecordNamespace.ToCellDep(),
	)
	for k := range accountCharTypeMap {
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

	//webauthn
	keyListMap := make(map[string]bool, 0)
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
			return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
		}

		txParams.CellDeps = append(txParams.CellDeps, keyListConfigCellContract.ToCellDep())
		cell, err := s.DasCore.GetKeyListCell(lockArgs)
		if err != nil {
			return nil, fmt.Errorf("GetKeyListCell(webauthn keyListCell) : %s", err.Error())
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
				return nil, fmt.Errorf("GetTransaction err : %s", err.Error())
			}
			webAuthnKeyListConfigBuilder, err := witness.WebAuthnKeyListDataBuilderFromTx(keyListConfigTx.Transaction, common.DataTypeNew)
			if err != nil {
				return nil, fmt.Errorf("WebAuthnKeyListDataBuilderFromTx err : %s", err.Error())
			}

			webAuthnKeyListConfigBuilder.DataEntityOpt.AsSlice()
			tmp := webAuthnKeyListConfigBuilder.DeviceKeyListCellData.AsSlice()
			keyListWitness := witness.GenDasDataWitnessWithByte(common.ActionDataTypeKeyListCfgCellData, tmp)
			txParams.OtherWitnesses = append(txParams.OtherWitnesses, keyListWitness)
			keyListMap[cell.OutPoint.TxHash.Hex()] = true
		}

	}

	// build tx
	txBuilder := txbuilder.NewDasTxBuilderFromBase(s.TxBuilderBase, nil)
	if err := txBuilder.BuildTransaction(&txParams); err != nil {
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
	log.Infof("BuildCreateSubAccountTx txSize: %d rulesSize: %d", sizeInBlock, rulesSize)

	txBuilder.Transaction.Outputs[len(txBuilder.Transaction.Outputs)-1].Capacity = changeCapacity

	hash, err := txBuilder.Transaction.ComputeHash()
	if err != nil {
		return nil, fmt.Errorf("ComputeHash err: %s", err.Error())
	}

	log.Info("BuildUpdateSubAccountTx:", txBuilder.TxString(), hash.String())

	// new tx outpoint
	res.DasTxBuilder = txBuilder
	subAccountOutpoint := &types.OutPoint{
		TxHash: hash,
		Index:  0,
	}

	// update smt status
	if err := s.DbDao.UpdateSmtRecordOutpoint(p.TaskInfo.TaskId, common.OutPointStruct2String(p.SubAccountOutpoint), common.OutPointStruct2String(subAccountOutpoint)); err != nil {
		return nil, fmt.Errorf("UpdateSmtRecordOutpoint err: %s", err.Error())
	}

	return &res, nil
}

func (s *SubAccountTxTool) BuildUpdateSubAccountTxForCustomScript(p *ParamBuildUpdateSubAccountTx) (*ResultBuildUpdateSubAccountTx, error) {
	var txParams txbuilder.BuildTransactionParams
	var res ResultBuildUpdateSubAccountTx

	customScriptCell, err := s.DasCore.GetCustomScriptLiveCell(p.SubAccountOutputsData)
	if err != nil {
		return nil, fmt.Errorf("GetCustomScriptLiveCell err: %s", err.Error())
	}
	// custom-script-witness
	customScriptInfo, err := s.DbDao.GetCustomScriptInfo(p.TaskInfo.ParentAccountId)
	if err != nil {
		return nil, fmt.Errorf("GetCustomScriptInfo err: %s", err.Error())
	}
	csiOutpoint := common.String2OutPointStruct(customScriptInfo.Outpoint)
	resTx, err := s.DasCore.Client().GetTransaction(s.Ctx, csiOutpoint.TxHash)
	if err != nil {
		return nil, fmt.Errorf("GetTransaction err: %s", err.Error())
	}

	customScriptConfigWit, _, err := witness.ConvertCustomScriptConfigByTx(resTx.Transaction)
	if err != nil {
		return nil, fmt.Errorf("ConvertCustomScriptConfigByTx err: %s", err.Error())
	}
	txParams.OtherWitnesses = append(txParams.OtherWitnesses, customScriptConfigWit)

	// get price
	builderConfigCellSub, err := s.DasCore.ConfigCellDataBuilderByTypeArgs(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		return nil, fmt.Errorf("ConfigCellDataBuilderByTypeArgs err: %s", err.Error())
	}
	newRate, err := molecule.Bytes2GoU32(builderConfigCellSub.ConfigCellSubAccount.NewSubAccountCustomPriceDasProfitRate().RawData())
	if err != nil {
		return nil, fmt.Errorf("NewSubAccountCustomPriceDasProfitRate err: %s", err.Error())
	}

	//newRate := uint32(2000)
	resPrice, err := GetCustomScriptMintTotalCapacity(&ParamCustomScriptMintTotalCapacity{
		Action: p.TaskInfo.Action,
		//PriceApi:                              &PriceApiDefault{},
		PriceApi: &PriceApiConfig{
			DasCore: s.DasCore,
			DbDao:   s.DbDao,
		},
		MintList:                              p.SmtRecordInfoList,
		Quote:                                 p.BaseInfo.QuoteCell.Quote(),
		NewSubAccountCustomPriceDasProfitRate: newRate,
		MinPriceCkb:                           p.NewSubAccountPrice,
	})
	if err != nil {
		return nil, fmt.Errorf("GetCustomScriptMintTotalCapacity err: %s", err.Error())
	}
	registerCapacity := resPrice.DasCapacity + resPrice.OwnerCapacity

	var change uint64
	var balanceLiveCells []*indexer.LiveCell
	if registerCapacity > 0 {
		change, balanceLiveCells, err = s.GetBalanceCell(&ParamBalance{
			DasLock:      p.BalanceDasLock,
			DasType:      p.BalanceDasType,
			NeedCapacity: p.CommonFee + registerCapacity,
		})
		if err != nil {
			return nil, fmt.Errorf("getBalanceCell err: %s", err.Error())
		}
	}

	// update smt status
	if err := s.DbDao.UpdateSmtStatus(p.TaskInfo.TaskId, tables.SmtStatusWriting); err != nil {
		return nil, fmt.Errorf("UpdateSmtStatus err: %s", err.Error())
	}

	// smt record
	var accountCharTypeMap = make(map[common.AccountCharType]struct{})
	var subAccountNewList []*witness.SubAccountNew
	opt := smt.SmtOpt{GetProof: true, GetRoot: true}
	for i, v := range p.SmtRecordInfoList {
		log.Info("BuildUpdateSubAccountTxForCustomScript:", v.TaskId, len(p.SmtRecordInfoList), "-", i)
		// update smt,get root and proof
		if v.SubAction == common.SubActionCreate {
			timeCellTimestamp := p.BaseInfo.TimeCell.Timestamp()
			subAccountData, subAccountNew, err := p.SmtRecordInfoList[i].GetCurrentSubAccountNew(s.DasCore, nil, p.BaseInfo.ContractDas, timeCellTimestamp)
			if err != nil {
				return nil, fmt.Errorf("CreateAccountInfo err: %s", err.Error())
			}
			key := smt.AccountIdToSmtH256(v.AccountId)
			value, err := subAccountData.ToH256()
			if err != nil {
				return nil, err
			}
			var smtKvTemp []smt.SmtKv
			smtKvTemp = append(smtKvTemp, smt.SmtKv{
				Key:   key,
				Value: value,
			})
			if res, err := p.Tree.UpdateSmt(smtKvTemp, opt); err != nil {
				return nil, fmt.Errorf("tree.Update err: %s", err.Error())
			} else if _, ok := res.Proofs[common.Bytes2Hex(key)]; !ok {
				return nil, fmt.Errorf("tree.MerkleProof Proof err: %s", res.Proofs)
			} else {
				subAccountNew.Proof = common.Hex2Bytes(res.Proofs[common.Bytes2Hex(key)])
				subAccountNew.NewRoot = res.Root
			}
			subAccountNew.EditKey = common.EditKeyCustomScript

			common.GetAccountCharType(accountCharTypeMap, subAccountData.AccountCharSet)
			subAccountNewList = append(subAccountNewList, subAccountNew)
		} else {
			subAccountBuilder, ok := p.SubAccountBuilderMap[v.AccountId]
			if !ok {
				return nil, fmt.Errorf("SubAccountBuilderMap not exist: %s", v.AccountId)
			}
			subAccountData, subAccountNew, err := p.SmtRecordInfoList[i].GetCurrentSubAccountNew(s.DasCore, subAccountBuilder, p.BaseInfo.ContractDas, 0)
			if err != nil {
				return nil, fmt.Errorf("GetCurrentSubAccount err: %s", err.Error())
			}
			key := smt.AccountIdToSmtH256(v.AccountId)
			value, err := subAccountData.ToH256()
			if err != nil {
				return nil, err
			}
			var smtKvTemp []smt.SmtKv
			smtKvTemp = append(smtKvTemp, smt.SmtKv{
				Key:   key,
				Value: value,
			})
			if res, err := p.Tree.UpdateSmt(smtKvTemp, opt); err != nil {
				return nil, fmt.Errorf("tree.Update err: %s", err.Error())
			} else if _, ok := res.Proofs[common.Bytes2Hex(key)]; !ok {
				return nil, fmt.Errorf("tree.MerkleProof Proof err: %s", res.Proofs)
			} else {
				subAccountNew.Proof = common.Hex2Bytes(res.Proofs[common.Bytes2Hex(key)])
				subAccountNew.NewRoot = res.Root
			}
			subAccountNewList = append(subAccountNewList, subAccountNew)
		}
	}

	// inputs
	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: p.SubAccountOutpoint,
	})

	// get balance cell
	for _, v := range balanceLiveCells {
		txParams.Inputs = append(txParams.Inputs, &types.CellInput{
			PreviousOutput: v.OutPoint,
		})
	}

	// outputs
	res.SubAccountCellOutput = &types.CellOutput{
		Capacity: p.SubAccountCellOutput.Capacity + registerCapacity,
		Lock:     p.SubAccountCellOutput.Lock,
		Type:     p.SubAccountCellOutput.Type,
	}
	txParams.Outputs = append(txParams.Outputs, res.SubAccountCellOutput) // sub_account
	// root+profit
	subDataDetail := witness.ConvertSubAccountCellOutputData(p.SubAccountOutputsData)
	subDataDetail.SmtRoot = subAccountNewList[len(subAccountNewList)-1].NewRoot
	subDataDetail.DasProfit += resPrice.DasCapacity
	subDataDetail.OwnerProfit += resPrice.OwnerCapacity
	res.SubAccountOutputsData = witness.BuildSubAccountCellOutputData(subDataDetail)
	txParams.OutputsData = append(txParams.OutputsData, res.SubAccountOutputsData) // smt root

	// change
	if change > 0 {
		txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
			Capacity: change,
			Lock:     p.BalanceDasLock,
			Type:     p.BalanceDasType,
		})
		txParams.OutputsData = append(txParams.OutputsData, []byte{})
	}

	// witness
	actionWitness, err := witness.GenActionDataWitnessV2(common.DasActionUpdateSubAccount, nil, "")
	if err != nil {
		return nil, fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	// smt
	smtWitnessList, _ := getSubAccountWitness(subAccountNewList)
	for _, v := range smtWitnessList {
		txParams.Witnesses = append(txParams.Witnesses, v) // smt witness
	}

	// account cell witness
	accTx, err := s.DasCore.Client().GetTransaction(s.Ctx, p.AccountOutPoint.TxHash)
	if err != nil {
		return nil, fmt.Errorf("GetTransaction acc tx err: %s", err.Error())
	}
	accBuilderMap, err := witness.AccountIdCellDataBuilderFromTx(accTx.Transaction, common.DataTypeNew)
	if err != nil {
		return nil, fmt.Errorf("AccountIdCellDataBuilderFromTx err: %s", err.Error())
	}
	accBuilder, ok := accBuilderMap[p.Account.AccountId]
	if !ok {
		return nil, fmt.Errorf("accBuilderMap is nil: %s", p.Account.AccountId)
	}
	accWitness, _, _ := accBuilder.GenWitness(&witness.AccountCellParam{
		OldIndex: 0,
		Action:   common.DasActionUpdateSubAccount,
	})
	txParams.Witnesses = append(txParams.Witnesses, accWitness)

	// cell deps
	// so
	soEd25519, _ := core.GetDasSoScript(common.SoScriptTypeEd25519)
	soEth, _ := core.GetDasSoScript(common.SoScriptTypeEth)
	soTron, _ := core.GetDasSoScript(common.SoScriptTypeTron)

	txParams.CellDeps = append(txParams.CellDeps,
		&types.CellDep{
			OutPoint: p.AccountOutPoint,
			DepType:  types.DepTypeCode,
		},
		&types.CellDep{
			OutPoint: customScriptCell.OutPoint,
			DepType:  types.DepTypeCode,
		},
		p.BaseInfo.ContractDas.ToCellDep(),
		p.BaseInfo.ContractAcc.ToCellDep(),
		p.BaseInfo.ContractSubAcc.ToCellDep(),
		p.BaseInfo.HeightCell.ToCellDep(),
		p.BaseInfo.TimeCell.ToCellDep(),
		p.BaseInfo.QuoteCell.ToCellDep(),
		p.BaseInfo.ConfigCellAcc.ToCellDep(),
		p.BaseInfo.ConfigCellSubAcc.ToCellDep(),
		p.BaseInfo.ConfigCellRecordNamespace.ToCellDep(),
		soEd25519.ToCellDep(),
		soEth.ToCellDep(),
		soTron.ToCellDep(),
	)
	for k := range accountCharTypeMap {
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

	// build tx
	txBuilder := txbuilder.NewDasTxBuilderFromBase(s.TxBuilderBase, nil)
	if err := txBuilder.BuildTransaction(&txParams); err != nil {
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
	log.Info("BuildCreateSubAccountTx change fee:", sizeInBlock)

	txBuilder.Transaction.Outputs[len(txBuilder.Transaction.Outputs)-1].Capacity = changeCapacity

	hash, err := txBuilder.Transaction.ComputeHash()
	if err != nil {
		return nil, fmt.Errorf("ComputeHash err: %s", err.Error())
	}

	log.Info("BuildUpdateSubAccountTx:", txBuilder.TxString(), hash.String())

	// new tx outpoint
	res.DasTxBuilder = txBuilder
	subAccountOutpoint := &types.OutPoint{
		TxHash: hash,
		Index:  0,
	}

	// update smt status
	if err := s.DbDao.UpdateSmtRecordOutpoint(p.TaskInfo.TaskId, common.OutPointStruct2String(p.SubAccountOutpoint), common.OutPointStruct2String(subAccountOutpoint)); err != nil {
		return nil, fmt.Errorf("UpdateSmtRecordOutpoint err: %s", err.Error())
	}

	return &res, nil
}

func (s *SubAccountTxTool) isCustomScript(data []byte) bool {
	subDataDetail := witness.ConvertSubAccountCellOutputData(data)
	customScriptArgs := make([]byte, 32)
	if len(subDataDetail.CustomScriptArgs) == 0 || bytes.Compare(subDataDetail.CustomScriptArgs, customScriptArgs) == 0 {
		return false
	}
	return true
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
