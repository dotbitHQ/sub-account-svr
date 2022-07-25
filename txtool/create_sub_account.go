package txtool

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/types"
)

type ParamBuildCreateSubAccountTx struct {
	TaskInfo              *tables.TableTaskInfo
	SmtRecordInfoList     []tables.TableSmtRecordInfo
	BaseInfo              *BaseInfo
	Tree                  *smt.SparseMerkleTree
	AccountOutPoint       *types.OutPoint
	SubAccountOutpoint    *types.OutPoint
	NewSubAccountPrice    uint64
	CommonFee             uint64
	AccountCellOutput     *types.CellOutput
	AccountCellData       []byte
	AccountCellWitness    []byte
	SubAccountCellOutput  *types.CellOutput
	SubAccountOutputsData []byte
	BalanceDasLock        *types.Script
	BalanceDasType        *types.Script
}

type ResultBuildCreateSubAccountTx struct {
	DasTxBuilder          *txbuilder.DasTxBuilder
	AccountOutPoint       *types.OutPoint
	SubAccountOutpoint    *types.OutPoint
	SubAccountOutputsData []byte
	SubAccountCellOutput  *types.CellOutput
}

func (s *SubAccountTxTool) BuildCreateSubAccountTx(p *ParamBuildCreateSubAccountTx) (*ResultBuildCreateSubAccountTx, error) {
	var res ResultBuildCreateSubAccountTx
	var txParams txbuilder.BuildTransactionParams
	timeCellTimestamp := p.BaseInfo.TimeCell.Timestamp()

	// get balance cell
	totalYears := uint64(0)
	for _, v := range p.SmtRecordInfoList {
		totalYears += v.RegisterYears
	}
	registerCapacity := p.NewSubAccountPrice * totalYears
	change, balanceLiveCells, err := s.getBalanceCell(&paramBalance{
		taskInfo:     p.TaskInfo,
		dasLock:      p.BalanceDasLock,
		dasType:      p.BalanceDasType,
		needCapacity: p.CommonFee + registerCapacity,
	})
	if err != nil {
		return nil, fmt.Errorf("getBalanceCell err: %s", err.Error())
	}

	// create task and records
	if p.TaskInfo.Id == 0 {
		p.TaskInfo.SmtStatus = tables.SmtStatusWriting
		if err := s.DbDao.CreateTaskWithRecords(p.TaskInfo, p.SmtRecordInfoList); err != nil {
			return nil, fmt.Errorf("CreateTaskWithRecords err: %s", err.Error())
		}
	} else {
		// update smt status
		if err := s.DbDao.UpdateSmtStatus(p.TaskInfo.TaskId, tables.SmtStatusWriting); err != nil {
			return nil, fmt.Errorf("UpdateSmtStatus err: %s", err.Error())
		}
	}

	// update smt,get root and proof
	var accountCharTypeMap = make(map[common.AccountCharType]struct{})
	var subAccountParamList []*witness.SubAccountParam
	for i, v := range p.SmtRecordInfoList {
		// update smt,get root and proof
		newSubAccount, subAccountParam, err := p.SmtRecordInfoList[i].GetCurrentSubAccount(nil, p.BaseInfo.ContractDas, timeCellTimestamp)
		if err != nil {
			return nil, fmt.Errorf("CreateAccountInfo err: %s", err.Error())
		} else {
			key := smt.AccountIdToSmtH256(v.AccountId)
			value := newSubAccount.ToH256()
			log.Info("BuildCreateSubAccountTx:", v.AccountId)
			log.Info("BuildCreateSubAccountTx key:", common.Bytes2Hex(key))
			log.Info("BuildCreateSubAccountTx value:", common.Bytes2Hex(value))

			log.Info("Tree.Root")
			if root, err := p.Tree.Root(); err != nil {
				return nil, fmt.Errorf("tree.Root err: %s", err.Error())
			} else {
				log.Info("PrevRoot:", v.AccountId, common.Bytes2Hex(root))
				subAccountParam.PrevRoot = root
			}
			log.Info("Tree.Update")
			if err := p.Tree.Update(key, value); err != nil {
				return nil, fmt.Errorf("tree.Update err: %s", err.Error())
			}
			log.Info("Tree.MerkleProof")
			if proof, err := p.Tree.MerkleProof([]smt.H256{key}, []smt.H256{value}); err != nil {
				return nil, fmt.Errorf("tree.MerkleProof err: %s", err.Error())
			} else {
				subAccountParam.Proof = *proof
				log.Info("Proof:", v.AccountId, common.Bytes2Hex(*proof))
			}
			log.Info("Tree.Root")
			if root, err := p.Tree.Root(); err != nil {
				return nil, fmt.Errorf("tree.Root err: %s", err.Error())
			} else {
				log.Info("CurrentRoot:", v.AccountId, common.Bytes2Hex(root))
				subAccountParam.CurrentRoot = root
			}
		}
		accountCharTypeMap = common.GetAccountCharType(newSubAccount.AccountCharSet)
		subAccountParamList = append(subAccountParamList, subAccountParam)
	}

	// update smt status
	//if err := s.DbDao.UpdateSmtStatus(p.TaskInfo.TaskId, tables.SmtStatusWriteComplete); err != nil {
	//	return nil, fmt.Errorf("UpdateSmtStatus err: %s", err.Error())
	//}

	// inputs
	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: p.AccountOutPoint,
	})
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
	txParams.Outputs = append(txParams.Outputs, p.AccountCellOutput) // account
	txParams.OutputsData = append(txParams.OutputsData, p.AccountCellData)

	res.SubAccountCellOutput = &types.CellOutput{
		Capacity: p.SubAccountCellOutput.Capacity + registerCapacity,
		Lock:     p.SubAccountCellOutput.Lock,
		Type:     p.SubAccountCellOutput.Type,
	}
	txParams.Outputs = append(txParams.Outputs, res.SubAccountCellOutput) // sub account
	// root+profit
	subDataDetail := witness.ConvertSubAccountCellOutputData(p.SubAccountOutputsData)
	subDataDetail.SmtRoot = subAccountParamList[len(subAccountParamList)-1].CurrentRoot
	subDataDetail.DasProfit = subDataDetail.DasProfit + registerCapacity
	res.SubAccountOutputsData = witness.BuildSubAccountCellOutputData(subDataDetail)
	txParams.OutputsData = append(txParams.OutputsData, res.SubAccountOutputsData) // smt root

	// change
	if change > 0 {
		changeList, _ := core.SplitOutputCell(change, 200*common.OneCkb, 2, p.BalanceDasLock, p.BalanceDasType)
		for _, cell := range changeList {
			txParams.Outputs = append(txParams.Outputs, cell)
			txParams.OutputsData = append(txParams.OutputsData, []byte{})
		}
		//txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		//	Capacity: change,
		//	Lock:     p.BalanceDasLock,
		//	Type:     p.BalanceDasType,
		//})
		//txParams.OutputsData = append(txParams.OutputsData, []byte{})
	}

	// witness
	actionWitness, err := witness.GenActionDataWitnessV2(common.DasActionCreateSubAccount, nil, common.ParamManager)
	if err != nil {
		return nil, fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)
	txParams.Witnesses = append(txParams.Witnesses, p.AccountCellWitness) // account

	smtWitnessList, _ := getSubAccountWitness(subAccountParamList)
	for _, v := range smtWitnessList {
		txParams.Witnesses = append(txParams.Witnesses, v) // smt witness
	}

	txParams.CellDeps = append(txParams.CellDeps,
		p.BaseInfo.ContractDas.ToCellDep(),
		p.BaseInfo.ContractAcc.ToCellDep(),
		p.BaseInfo.ContractSubAcc.ToCellDep(),
		p.BaseInfo.HeightCell.ToCellDep(),
		p.BaseInfo.TimeCell.ToCellDep(),
		p.BaseInfo.ConfigCellAcc.ToCellDep(),
		p.BaseInfo.ConfigCellSubAcc.ToCellDep(),
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
		}
	}

	// build tx
	txBuilder := txbuilder.NewDasTxBuilderFromBase(s.TxBuilderBase, nil)

	accountOutPoint := common.OutPointStruct2String(p.AccountOutPoint)
	txBuilder.MapInputsCell[accountOutPoint] = &types.CellWithStatus{
		Cell: &types.CellInfo{
			Data:   nil,
			Output: p.AccountCellOutput,
		},
		Status: "",
	}

	subAccountOutpoint := common.OutPointStruct2String(p.SubAccountOutpoint)
	txBuilder.MapInputsCell[subAccountOutpoint] = &types.CellWithStatus{
		Cell: &types.CellInfo{
			Data:   nil,
			Output: p.SubAccountCellOutput,
		},
		Status: "",
	}

	if err := txBuilder.BuildTransaction(&txParams); err != nil {
		return nil, fmt.Errorf("BuildTransaction err: %s", err.Error())
	}

	// note: change fee
	sizeInBlock, _ := txBuilder.Transaction.SizeInBlock()
	changeCapacity := txBuilder.Transaction.Outputs[len(txBuilder.Transaction.Outputs)-1].Capacity
	changeCapacity += p.CommonFee - sizeInBlock - 5000
	log.Info("BuildCreateSubAccountTx change fee:", sizeInBlock)

	txBuilder.Transaction.Outputs[len(txBuilder.Transaction.Outputs)-1].Capacity = changeCapacity

	hash, err := txBuilder.Transaction.ComputeHash()
	if err != nil {
		return nil, fmt.Errorf("ComputeHash err: %s", err.Error())
	}

	log.Info("BuildCreateSubAccountTx:", txBuilder.TxString(), hash.String())

	// new tx outpoint
	res.DasTxBuilder = txBuilder
	res.AccountOutPoint = &types.OutPoint{
		TxHash: hash,
		Index:  0,
	}
	res.SubAccountOutpoint = &types.OutPoint{
		TxHash: hash,
		Index:  1,
	}
	log.Info("BuildCreateSubAccountTx:", p.SubAccountCellOutput.Capacity)

	// update smt status
	if err := s.DbDao.UpdateSmtRecordOutpoint(p.TaskInfo.TaskId, common.OutPointStruct2String(p.SubAccountOutpoint), common.OutPointStruct2String(res.SubAccountOutpoint)); err != nil {
		return nil, fmt.Errorf("UpdateSmtRecordOutpoint err: %s", err.Error())
	}

	return &res, nil
}
