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

func (s *SubAccountTxTool) BuildTxsForUpdateSubAccount(p *ParamBuildTxs) (*ResultBuildTxs, error) {
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
	// build txs
	for i, task := range p.TaskList {
		records, ok := p.TaskMap[task.TaskId]
		if !ok {
			continue
		}
		switch task.Action {
		case common.DasActionUpdateSubAccount:
			var resUpdate *ResultBuildUpdateSubAccountTx
			var err error
			if res.IsCustomScript {
				resUpdate, err = s.BuildUpdateSubAccountTxForCustomScript(&ParamBuildUpdateSubAccountTx{
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
					return nil, fmt.Errorf("BuildUpdateSubAccountTxForCustomScript err: %s", err.Error())
				}
			} else {
				resUpdate, err = s.BuildUpdateSubAccountTx(&ParamBuildUpdateSubAccountTx{
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
			}
			res.DasTxBuilderList = append(res.DasTxBuilderList, resUpdate.DasTxBuilder)
		default:
			return nil, fmt.Errorf("not exist action [%s]", task.Action)
		}
	}

	return &res, nil
}
