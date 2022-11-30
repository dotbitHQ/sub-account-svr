package txtool

import (
	"context"
	"das_sub_account/dao"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/dascache"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/mylog"
	"go.mongodb.org/mongo-driver/mongo"
)

var log = mylog.NewLogger("txtool", mylog.LevelDebug)

type SubAccountTxTool struct {
	Ctx           context.Context
	Mongo         *mongo.Client
	DbDao         *dao.DbDao
	DasCore       *core.DasCore
	DasCache      *dascache.DasCache
	ServerScript  *types.Script
	TxBuilderBase *txbuilder.DasTxBuilderBase
}

type ParamBuildTxs struct {
	TaskList             []tables.TableTaskInfo
	TaskMap              map[string][]tables.TableSmtRecordInfo
	Account              *tables.TableAccountInfo // parent account
	SubAccountLiveCell   *indexer.LiveCell
	Tree                 *smt.SparseMerkleTree
	BaseInfo             *BaseInfo
	BalanceDasLock       *types.Script
	BalanceDasType       *types.Script
	SubAccountIds        []string
	SubAccountValueMap   map[string]string
	SubAccountBuilderMap map[string]*witness.SubAccountNew
}

type ResultBuildTxs struct {
	IsCustomScript   bool
	DasTxBuilderList []*txbuilder.DasTxBuilder
}

func (s *SubAccountTxTool) BuildTxs(p *ParamBuildTxs) (*ResultBuildTxs, error) {
	var res ResultBuildTxs

	newSubAccountPrice, _ := molecule.Bytes2GoU64(p.BaseInfo.ConfigCellBuilder.ConfigCellSubAccount.NewSubAccountPrice().RawData())
	commonFee, _ := molecule.Bytes2GoU64(p.BaseInfo.ConfigCellBuilder.ConfigCellSubAccount.CommonFee().RawData())

	// outpoint
	accountOutPoint := common.String2OutPointStruct(p.Account.Outpoint)
	subAccountOutpoint := p.SubAccountLiveCell.OutPoint

	// account
	res.IsCustomScript = s.isCustomScript(p.SubAccountLiveCell.OutputData)
	subAccountCellOutput := p.SubAccountLiveCell.Output
	subAccountOutputsData := p.SubAccountLiveCell.OutputData
	accOutpoint := common.String2OutPointStruct(p.Account.Outpoint)
	accountCellOutput, accountCellWitness, accountCellData, err := s.getAccountByOutpoint(accOutpoint, p.Account.AccountId, res.IsCustomScript)
	if err != nil {
		return nil, fmt.Errorf("getAccountByOutpoint err: %s", err.Error())
	}

	// build txs
	for i, task := range p.TaskList {
		records, ok := p.TaskMap[task.TaskId]
		if !ok {
			continue
		}
		switch task.Action {
		case common.DasActionCreateSubAccount:
			var resCreate *ResultBuildCreateSubAccountTx
			if res.IsCustomScript {
				resCreate, err = s.BuildCreateSubAccountTxByScript(&ParamBuildCreateSubAccountTx{
					TaskInfo:              &p.TaskList[i],
					SmtRecordInfoList:     records,
					BaseInfo:              p.BaseInfo,
					Tree:                  p.Tree,
					AccountOutPoint:       accountOutPoint,
					SubAccountOutpoint:    subAccountOutpoint,
					NewSubAccountPrice:    newSubAccountPrice,
					CommonFee:             commonFee,
					AccountCellOutput:     accountCellOutput,
					AccountCellData:       accountCellData,
					AccountCellWitness:    accountCellWitness,
					SubAccountCellOutput:  subAccountCellOutput,
					SubAccountOutputsData: subAccountOutputsData,
					BalanceDasLock:        p.BalanceDasLock,
					BalanceDasType:        p.BalanceDasType,
				})
				if err != nil {
					return nil, fmt.Errorf("BuildCreateSubAccountTx err: %s", err.Error())
				}
			} else {
				resCreate, err = s.BuildCreateSubAccountTx(&ParamBuildCreateSubAccountTx{
					TaskInfo:              &p.TaskList[i],
					SmtRecordInfoList:     records,
					BaseInfo:              p.BaseInfo,
					Tree:                  p.Tree,
					AccountOutPoint:       accountOutPoint,
					SubAccountOutpoint:    subAccountOutpoint,
					NewSubAccountPrice:    newSubAccountPrice,
					CommonFee:             commonFee,
					AccountCellOutput:     accountCellOutput,
					AccountCellData:       accountCellData,
					AccountCellWitness:    accountCellWitness,
					SubAccountCellOutput:  subAccountCellOutput,
					SubAccountOutputsData: subAccountOutputsData,
					BalanceDasLock:        p.BalanceDasLock,
					BalanceDasType:        p.BalanceDasType,
				})
				if err != nil {
					return nil, fmt.Errorf("BuildCreateSubAccountTx err: %s", err.Error())
				}
			}

			accountOutPoint = resCreate.AccountOutPoint
			subAccountOutpoint = resCreate.SubAccountOutpoint
			subAccountOutputsData = resCreate.SubAccountOutputsData
			subAccountCellOutput = resCreate.SubAccountCellOutput
			res.DasTxBuilderList = append(res.DasTxBuilderList, resCreate.DasTxBuilder)
		case common.DasActionEditSubAccount:
			resEdit, err := s.BuildEditSubAccountTx(&ParamBuildEditSubAccountTx{
				TaskInfo:              &p.TaskList[i],
				SmtRecordInfoList:     records,
				BaseInfo:              p.BaseInfo,
				Tree:                  p.Tree,
				Account:               p.Account,
				AccountOutPoint:       accountOutPoint,
				SubAccountOutpoint:    subAccountOutpoint,
				SubAccountCellOutput:  subAccountCellOutput,
				CommonFee:             commonFee,
				SubAccountBuilderMap:  p.SubAccountBuilderMap,
				SubAccountOutputsData: subAccountOutputsData,
			})
			if err != nil {
				return nil, fmt.Errorf("BuildEditSubAccountTx err: %s", err.Error())
			}
			subAccountOutpoint = resEdit.SubAccountOutpoint
			subAccountOutputsData = resEdit.SubAccountOutputsData
			subAccountCellOutput = resEdit.SubAccountCellOutput
			res.DasTxBuilderList = append(res.DasTxBuilderList, resEdit.DasTxBuilder)
		case common.DasActionUpdateSubAccount:
			resUpdate, err := s.BuildUpdateSubAccountTx(&ParamBuildUpdateSubAccountTx{
				TaskInfo:              &p.TaskList[i],
				Account:               p.Account,
				AccountOutPoint:       accountOutPoint,
				SubAccountOutpoint:    subAccountOutpoint,
				SmtRecordInfoList:     records,
				Tree:                  p.Tree,
				BaseInfo:              p.BaseInfo,
				SubAccountBuilderMap:  p.SubAccountBuilderMap,
				NewSubAccountPrice:    newSubAccountPrice,
				BalanceDasLock:        p.BalanceDasLock,
				BalanceDasType:        p.BalanceDasType,
				CommonFee:             commonFee,
				SubAccountCellOutput:  subAccountCellOutput,
				SubAccountOutputsData: subAccountOutputsData,
			})
			if err != nil {
				return nil, fmt.Errorf("BuildUpdateSubAccountTx err: %s", err.Error())
			}
			res.DasTxBuilderList = append(res.DasTxBuilderList, resUpdate.DasTxBuilder)
		default:
			return nil, fmt.Errorf("not exist action [%s]", task.Action)
		}
	}

	return &res, nil
}

func (s *SubAccountTxTool) RollbackSmtRecords(tree *smt.SparseMerkleTree, subAccountIds []string, subAccountValueMap map[string]string) error {
	// rollback
	for _, v := range subAccountIds {
		key := smt.AccountIdToSmtH256(v)
		value := smt.H256Zero()
		if subAccountValue, ok := subAccountValueMap[v]; ok {
			value = common.Hex2Bytes(subAccountValue)
		}
		if err := tree.Update(key, value); err != nil {
			return fmt.Errorf("tree.Update err: %s", err.Error())
		}
	}
	return nil
}
