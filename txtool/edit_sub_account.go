package txtool

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/core"
	"github.com/DeAccountSystems/das-lib/smt"
	"github.com/DeAccountSystems/das-lib/txbuilder"
	"github.com/DeAccountSystems/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
)

type ParamBuildEditSubAccountTx struct {
	TaskInfo              *tables.TableTaskInfo
	Account               *tables.TableAccountInfo
	AccountOutPoint       *types.OutPoint
	SubAccountOutpoint    *types.OutPoint
	SmtRecordInfoList     []tables.TableSmtRecordInfo
	Tree                  *smt.SparseMerkleTree
	SubAccountCellOutput  *types.CellOutput
	SubAccountOutputsData []byte
	CommonFee             uint64
	BaseInfo              *BaseInfo
	SubAccountBuilderMap  map[string]*witness.SubAccountBuilder
}

type ResultBuildEditSubAccountTx struct {
	DasTxBuilder          *txbuilder.DasTxBuilder
	SubAccountOutpoint    *types.OutPoint
	SubAccountOutputsData []byte
	SubAccountCellOutput  *types.CellOutput
}

func (s *SubAccountTxTool) BuildEditSubAccountTx(p *ParamBuildEditSubAccountTx) (*ResultBuildEditSubAccountTx, error) {
	var txParams txbuilder.BuildTransactionParams

	// update smt status
	if err := s.DbDao.UpdateSmtStatus(p.TaskInfo.TaskId, tables.SmtStatusWriting); err != nil {
		return nil, fmt.Errorf("UpdateSmtStatus err: %s", err.Error())
	}

	// update smt,get root and proof
	var res ResultBuildEditSubAccountTx
	var subAccountParamList []*witness.SubAccountParam
	for i, v := range p.SmtRecordInfoList {
		// update smt,get root and proof
		subAccountBuilder, ok := p.SubAccountBuilderMap[v.AccountId]
		if !ok {
			return nil, fmt.Errorf("SubAccountBuilderMap not exist: %s", v.AccountId)
		}
		newSubAccount, subAccountParam, err := p.SmtRecordInfoList[i].GetCurrentSubAccount(subAccountBuilder.CurrentSubAccount, p.BaseInfo.ContractDas, 0)
		if err != nil {
			return nil, fmt.Errorf("GetCurrentSubAccount err: %s", err.Error())
		} else {
			key := smt.AccountIdToSmtH256(v.AccountId)
			value := newSubAccount.ToH256()
			log.Info("BuildEditSubAccountTx:", v.AccountId)
			log.Info("BuildEditSubAccountTx key:", common.Bytes2Hex(key))
			log.Info("BuildEditSubAccountTx value:", common.Bytes2Hex(value))
			log.Info("BuildEditSubAccountTx sub account:", toolib.JsonString(newSubAccount))
			if root, err := p.Tree.Root(); err != nil {
				return nil, fmt.Errorf("tree.Root err: %s", err.Error())
			} else {
				log.Info("PrevRoot:", v.AccountId, common.Bytes2Hex(root))
				subAccountParam.PrevRoot = root
			}
			if err := p.Tree.Update(key, value); err != nil {
				return nil, fmt.Errorf("tree.Update err: %s", err.Error())
			}
			if proof, err := p.Tree.MerkleProof([]smt.H256{key}, []smt.H256{value}); err != nil {
				return nil, fmt.Errorf("tree.MerkleProof err: %s", err.Error())
			} else {
				subAccountParam.Proof = *proof
				log.Info("Proof:", v.AccountId, common.Bytes2Hex(*proof))
			}
			if root, err := p.Tree.Root(); err != nil {
				return nil, fmt.Errorf("tree.Root err: %s", err.Error())
			} else {
				log.Info("CurrentRoot:", v.AccountId, common.Bytes2Hex(root))
				subAccountParam.CurrentRoot = root
			}
		}
		subAccountParamList = append(subAccountParamList, subAccountParam)
	}

	// update smt status
	//if err := s.DbDao.UpdateSmtStatus(p.TaskInfo.TaskId, tables.SmtStatusWriteComplete); err != nil {
	//	return nil, fmt.Errorf("UpdateSmtStatus err: %s", err.Error())
	//}

	// inputs
	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: p.SubAccountOutpoint,
	})
	// outputs
	commonFee := p.CommonFee
	res.SubAccountCellOutput = &types.CellOutput{
		Capacity: p.SubAccountCellOutput.Capacity - commonFee,
		Lock:     p.SubAccountCellOutput.Lock,
		Type:     p.SubAccountCellOutput.Type,
	}

	txParams.Outputs = append(txParams.Outputs, res.SubAccountCellOutput)
	// root+profit
	_, profit := witness.ConvertSubAccountCellOutputData(p.SubAccountOutputsData)
	log.Info("ConvertSubAccountCellOutputData:", profit, common.Bytes2Hex(p.SubAccountOutputsData))
	res.SubAccountOutputsData = witness.BuildSubAccountCellOutputData(subAccountParamList[len(subAccountParamList)-1].CurrentRoot, profit)
	txParams.OutputsData = append(txParams.OutputsData, res.SubAccountOutputsData) // smt root

	// witness
	actionWitness, err := witness.GenActionDataWitness(common.DasActionEditSubAccount, nil)
	if err != nil {
		return nil, fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)
	smtWitnessList, _ := getSubAccountWitness(subAccountParamList)
	for _, v := range smtWitnessList {
		txParams.Witnesses = append(txParams.Witnesses, v) // smt witness
	}

	// so
	soEd25519, _ := core.GetDasSoScript(common.SoScriptTypeEd25519)
	soEth, _ := core.GetDasSoScript(common.SoScriptTypeEth)
	soTron, _ := core.GetDasSoScript(common.SoScriptTypeTron)

	// cell deps
	txParams.CellDeps = append(txParams.CellDeps,
		p.BaseInfo.ConfigCellAcc.ToCellDep(),
		p.BaseInfo.ContractDas.ToCellDep(),
		p.BaseInfo.ContractSubAcc.ToCellDep(),
		p.BaseInfo.HeightCell.ToCellDep(),
		p.BaseInfo.TimeCell.ToCellDep(),
		p.BaseInfo.ConfigCellSubAcc.ToCellDep(),
		p.BaseInfo.ConfigCellRecordNamespace.ToCellDep(),
		&types.CellDep{
			OutPoint: p.AccountOutPoint,
			DepType:  types.DepTypeCode,
		},
		soEd25519.ToCellDep(),
		soEth.ToCellDep(),
		soTron.ToCellDep(),
	)

	// build tx
	txBuilder := txbuilder.NewDasTxBuilderFromBase(s.TxBuilderBase, nil)
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
	sizeInBlock += 1e3
	if sizeInBlock < 1e4 {
		res.SubAccountCellOutput.Capacity += commonFee - 1e4
	} else {
		res.SubAccountCellOutput.Capacity += commonFee - sizeInBlock
	}

	log.Info("BuildEditSubAccountTx change fee:", sizeInBlock+1e3)

	hash, err := txBuilder.Transaction.ComputeHash()
	if err != nil {
		return nil, fmt.Errorf("ComputeHash err: %s", err.Error())
	}

	log.Info("BuildEditSubAccountTx:", txBuilder.TxString(), hash.String())

	// new tx outpoint
	res.DasTxBuilder = txBuilder
	res.SubAccountOutpoint = &types.OutPoint{
		TxHash: hash,
		Index:  0,
	}

	// update smt status
	if err := s.DbDao.UpdateSmtRecordOutpoint(p.TaskInfo.TaskId, common.OutPointStruct2String(p.SubAccountOutpoint), common.OutPointStruct2String(res.SubAccountOutpoint)); err != nil {
		return nil, fmt.Errorf("UpdateSmtRecordOutpoint err: %s", err.Error())
	}

	return &res, nil
}
