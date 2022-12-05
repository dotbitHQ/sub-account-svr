package txtool

import (
	"das_sub_account/tables"
	"encoding/json"
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
)

type ParamBuildUpdateSubAccountTx struct {
	TaskInfo              *tables.TableTaskInfo
	Account               *tables.TableAccountInfo
	AccountOutPoint       *types.OutPoint
	SubAccountOutpoint    *types.OutPoint
	SmtRecordInfoList     []tables.TableSmtRecordInfo
	Tree                  *smt.SparseMerkleTree
	BaseInfo              *BaseInfo
	SubAccountBuilderMap  map[string]*witness.SubAccountNew
	NewSubAccountPrice    uint64
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

	balanceDasLock := p.BalanceDasLock
	balanceDasType := p.BalanceDasType
	// get mint sign info
	var witnessMintSignInfo []byte
	mintSignTree := smt.NewSparseMerkleTree(nil)
	for _, v := range p.SmtRecordInfoList {
		if v.SubAction != common.SubActionCreate {
			continue
		}
		if v.MintSignId != "" {
			mintSignInfo, err := s.DbDao.GetMinSignInfo(v.MintSignId)
			if err != nil {
				return nil, fmt.Errorf("GetMinSignInfo err: %s", err.Error())
			}
			witnessMintSignInfo = mintSignInfo.GenWitness()
			var listKeyValue []tables.MintSignInfoKeyValue
			_ = json.Unmarshal([]byte(mintSignInfo.KeyValue), &listKeyValue)
			for _, kv := range listKeyValue {
				smtKey := smt.AccountIdToSmtH256(kv.Key)
				smtValue, err := blake2b.Blake256(common.Hex2Bytes(kv.Value))
				if err != nil {
					return nil, fmt.Errorf("blake2b.Blake256 err: %s", err.Error())
				}
				if err = mintSignTree.Update(smtKey, smtValue); err != nil {
					return nil, fmt.Errorf("mintSignTree.Update err: %s", err.Error())
				}
			}
			balanceDasLock, balanceDasType, err = s.DasCore.Daf().HexToScript(core.DasAddressHex{
				DasAlgorithmId: mintSignInfo.ChainType.ToDasAlgorithmId(true),
				AddressHex:     mintSignInfo.Address,
				IsMulti:        false,
				ChainType:      mintSignInfo.ChainType,
			})
			if err != nil {
				return nil, fmt.Errorf("manager HexToScript err: %s", err.Error())
			}

			break
		}
	}

	// get balance cell
	totalYears := uint64(0)
	for _, v := range p.SmtRecordInfoList {
		if v.SubAction == common.SubActionCreate {
			totalYears += v.RegisterYears
		}
	}
	registerCapacity := p.NewSubAccountPrice * totalYears
	var change uint64
	var balanceLiveCells []*indexer.LiveCell
	var err error
	if registerCapacity > 0 {
		change, balanceLiveCells, err = s.getBalanceCell(&paramBalance{
			taskInfo:     p.TaskInfo,
			dasLock:      balanceDasLock,
			dasType:      balanceDasType,
			needCapacity: p.CommonFee + registerCapacity,
		})
		if err != nil {
			return nil, fmt.Errorf("getBalanceCell err: %s", err.Error())
		}
		change += p.CommonFee
	}

	// update smt status
	if err := s.DbDao.UpdateSmtStatus(p.TaskInfo.TaskId, tables.SmtStatusWriting); err != nil {
		return nil, fmt.Errorf("UpdateSmtStatus err: %s", err.Error())
	}

	// smt record
	var accountCharTypeMap = make(map[common.AccountCharType]struct{})
	var subAccountNewList []*witness.SubAccountNew
	for i, v := range p.SmtRecordInfoList {
		// update smt,get root and proof
		if v.SubAction == common.SubActionCreate {
			timeCellTimestamp := p.BaseInfo.TimeCell.Timestamp()
			subAccountData, subAccountNew, err := p.SmtRecordInfoList[i].GetCurrentSubAccountNew(nil, p.BaseInfo.ContractDas, timeCellTimestamp)
			if err != nil {
				return nil, fmt.Errorf("CreateAccountInfo err: %s", err.Error())
			}
			if len(witnessMintSignInfo) > 0 {
				smtKey := smt.AccountIdToSmtH256(v.AccountId)
				smtValue, err := blake2b.Blake256(common.Hex2Bytes(v.RegisterArgs))
				if err != nil {
					return nil, fmt.Errorf("blake2b.Blake256 err: %s", err.Error())
				}
				mintSignProof, err := mintSignTree.MerkleProof([]smt.H256{smtKey}, []smt.H256{smtValue})
				if err != nil {
					return nil, fmt.Errorf("mintSignTree.MerkleProof err: %s", err.Error())
				}
				subAccountNew.EditValue = *mintSignProof
			}
			key := smt.AccountIdToSmtH256(v.AccountId)
			value := subAccountData.ToH256()
			if err := p.Tree.Update(key, value); err != nil {
				return nil, fmt.Errorf("tree.Update err: %s", err.Error())
			}
			if proof, err := p.Tree.MerkleProof([]smt.H256{key}, []smt.H256{value}); err != nil {
				return nil, fmt.Errorf("tree.MerkleProof err: %s", err.Error())
			} else {
				subAccountNew.Proof = *proof
			}
			if root, err := p.Tree.Root(); err != nil {
				return nil, fmt.Errorf("tree.Root err: %s", err.Error())
			} else {
				subAccountNew.NewRoot = root
			}
			common.GetAccountCharType(accountCharTypeMap, subAccountData.AccountCharSet)
			subAccountNewList = append(subAccountNewList, subAccountNew)
		} else {
			subAccountBuilder, ok := p.SubAccountBuilderMap[v.AccountId]
			if !ok {
				return nil, fmt.Errorf("SubAccountBuilderMap not exist: %s", v.AccountId)
			}
			subAccountData, subAccountNew, err := p.SmtRecordInfoList[i].GetCurrentSubAccountNew(subAccountBuilder.CurrentSubAccountData, p.BaseInfo.ContractDas, 0)
			if err != nil {
				return nil, fmt.Errorf("GetCurrentSubAccount err: %s", err.Error())
			} else {
				key := smt.AccountIdToSmtH256(v.AccountId)
				value := subAccountData.ToH256()
				if err := p.Tree.Update(key, value); err != nil {
					return nil, fmt.Errorf("tree.Update err: %s", err.Error())
				}
				if proof, err := p.Tree.MerkleProof([]smt.H256{key}, []smt.H256{value}); err != nil {
					return nil, fmt.Errorf("tree.MerkleProof err: %s", err.Error())
				} else {
					subAccountNew.Proof = *proof
				}
				if root, err := p.Tree.Root(); err != nil {
					return nil, fmt.Errorf("tree.Root err: %s", err.Error())
				} else {
					subAccountNew.NewRoot = root
				}
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
	txParams.Outputs = append(txParams.Outputs, res.SubAccountCellOutput) // sub account
	// root+profit
	subDataDetail := witness.ConvertSubAccountCellOutputData(p.SubAccountOutputsData)
	subDataDetail.SmtRoot = subAccountNewList[len(subAccountNewList)-1].NewRoot
	subDataDetail.DasProfit = subDataDetail.DasProfit + registerCapacity
	res.SubAccountOutputsData = witness.BuildSubAccountCellOutputData(subDataDetail)
	txParams.OutputsData = append(txParams.OutputsData, res.SubAccountOutputsData) // smt root

	// change
	if change > 0 {
		txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
			Capacity: change,
			Lock:     balanceDasLock,
			Type:     balanceDasType,
		})
		txParams.OutputsData = append(txParams.OutputsData, []byte{})
	}

	// witness
	actionWitness, err := witness.GenActionDataWitnessV2(common.DasActionUpdateSubAccount, nil, "")
	if err != nil {
		return nil, fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	// mint sign info
	if len(witnessMintSignInfo) > 0 {
		txParams.Witnesses = append(txParams.Witnesses, witnessMintSignInfo)
	}
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

	// so
	soEd25519, _ := core.GetDasSoScript(common.SoScriptTypeEd25519)
	soEth, _ := core.GetDasSoScript(common.SoScriptTypeEth)
	soTron, _ := core.GetDasSoScript(common.SoScriptTypeTron)

	txParams.CellDeps = append(txParams.CellDeps,
		&types.CellDep{
			OutPoint: p.AccountOutPoint,
			DepType:  types.DepTypeCode,
		},
		p.BaseInfo.ContractDas.ToCellDep(),
		p.BaseInfo.ContractAcc.ToCellDep(),
		p.BaseInfo.ContractSubAcc.ToCellDep(),
		p.BaseInfo.HeightCell.ToCellDep(),
		p.BaseInfo.TimeCell.ToCellDep(),
		p.BaseInfo.ConfigCellAcc.ToCellDep(),
		p.BaseInfo.ConfigCellSubAcc.ToCellDep(),
		p.BaseInfo.ConfigCellRecordNamespace.ToCellDep(),
		soEd25519.ToCellDep(),
		soEth.ToCellDep(),
		soTron.ToCellDep(),
	)
	for k, _ := range accountCharTypeMap {
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
		Action: common.DasActionCreateSubAccount,
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
	change, balanceLiveCells, err := s.getBalanceCell(&paramBalance{
		taskInfo:     p.TaskInfo,
		dasLock:      p.BalanceDasLock,
		dasType:      p.BalanceDasType,
		needCapacity: p.CommonFee + registerCapacity,
	})
	if err != nil {
		return nil, fmt.Errorf("getBalanceCell err: %s", err.Error())
	}

	// update smt status
	if err := s.DbDao.UpdateSmtStatus(p.TaskInfo.TaskId, tables.SmtStatusWriting); err != nil {
		return nil, fmt.Errorf("UpdateSmtStatus err: %s", err.Error())
	}

	// smt record
	var accountCharTypeMap = make(map[common.AccountCharType]struct{})
	var subAccountNewList []*witness.SubAccountNew
	for i, v := range p.SmtRecordInfoList {
		// update smt,get root and proof
		if v.SubAction == common.SubActionCreate {
			timeCellTimestamp := p.BaseInfo.TimeCell.Timestamp()
			subAccountData, subAccountNew, err := p.SmtRecordInfoList[i].GetCurrentSubAccountNew(nil, p.BaseInfo.ContractDas, timeCellTimestamp)
			if err != nil {
				return nil, fmt.Errorf("CreateAccountInfo err: %s", err.Error())
			}
			key := smt.AccountIdToSmtH256(v.AccountId)
			value := subAccountData.ToH256()
			if err := p.Tree.Update(key, value); err != nil {
				return nil, fmt.Errorf("tree.Update err: %s", err.Error())
			}
			if proof, err := p.Tree.MerkleProof([]smt.H256{key}, []smt.H256{value}); err != nil {
				return nil, fmt.Errorf("tree.MerkleProof err: %s", err.Error())
			} else {
				subAccountNew.Proof = *proof
			}
			if root, err := p.Tree.Root(); err != nil {
				return nil, fmt.Errorf("tree.Root err: %s", err.Error())
			} else {
				subAccountNew.NewRoot = root
			}
			common.GetAccountCharType(accountCharTypeMap, subAccountData.AccountCharSet)
			subAccountNewList = append(subAccountNewList, subAccountNew)
		} else {
			subAccountBuilder, ok := p.SubAccountBuilderMap[v.AccountId]
			if !ok {
				return nil, fmt.Errorf("SubAccountBuilderMap not exist: %s", v.AccountId)
			}
			subAccountData, subAccountNew, err := p.SmtRecordInfoList[i].GetCurrentSubAccountNew(subAccountBuilder.CurrentSubAccountData, p.BaseInfo.ContractDas, 0)
			if err != nil {
				return nil, fmt.Errorf("GetCurrentSubAccount err: %s", err.Error())
			}
			key := smt.AccountIdToSmtH256(v.AccountId)
			value := subAccountData.ToH256()
			if err := p.Tree.Update(key, value); err != nil {
				return nil, fmt.Errorf("tree.Update err: %s", err.Error())
			}
			if proof, err := p.Tree.MerkleProof([]smt.H256{key}, []smt.H256{value}); err != nil {
				return nil, fmt.Errorf("tree.MerkleProof err: %s", err.Error())
			} else {
				subAccountNew.Proof = *proof
			}
			if root, err := p.Tree.Root(); err != nil {
				return nil, fmt.Errorf("tree.Root err: %s", err.Error())
			} else {
				subAccountNew.NewRoot = root
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
	txParams.Outputs = append(txParams.Outputs, res.SubAccountCellOutput) // sub account
	// root+profit
	subDataDetail := witness.ConvertSubAccountCellOutputData(p.SubAccountOutputsData)
	subDataDetail.SmtRoot = subAccountNewList[len(subAccountNewList)-1].NewRoot
	subDataDetail.DasProfit = subDataDetail.DasProfit + registerCapacity
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
		p.BaseInfo.ConfigCellAcc.ToCellDep(),
		p.BaseInfo.ConfigCellSubAcc.ToCellDep(),
		p.BaseInfo.ConfigCellRecordNamespace.ToCellDep(),
	)
	for k, _ := range accountCharTypeMap {
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
	//// header deps
	//subTx, err := s.DasCore.Client().GetTransaction(s.Ctx, p.SubAccountOutpoint.TxHash)
	//if err != nil {
	//	return nil, fmt.Errorf("GetTransaction err: %s", err.Error())
	//}
	//txBuilder.Transaction.HeaderDeps = append(txBuilder.Transaction.HeaderDeps, *subTx.TxStatus.BlockHash)

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
