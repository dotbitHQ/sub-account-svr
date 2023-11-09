package txtool

import (
	"das_sub_account/tables"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
)

func (s *SubAccountTxTool) DoCheckCustomScriptHashNew(subAccountLiveCell *indexer.LiveCell, taskList []tables.TableTaskInfo) (string, bool) {
	// check custom script hash
	subAccDetail := witness.ConvertSubAccountCellOutputData(subAccountLiveCell.OutputData)
	customScripHash := ""
	if subAccDetail.HasCustomScriptArgs() {
		customScripHash = subAccDetail.ArgsAndConfigHash()
	}
	for _, v := range taskList {
		if v.CustomScripHash != customScripHash {
			return v.TaskId, false
		}
	}
	return "", true
}

type ResDoCheckContinue struct {
	Continue           bool
	BaseInfo           *BaseInfo
	SubAccountLiveCell *indexer.LiveCell
}

func (s *SubAccountTxTool) DoCheckBeforeBuildTx(parentAccountId string) (*ResDoCheckContinue, error) {
	var res ResDoCheckContinue

	// check (1,0)(0,2)(2,1)(3,?)
	if ok, err := s.CheckInProgressTask(parentAccountId); err != nil {
		return nil, fmt.Errorf("CheckInProgressTask err: %s", err.Error())
	} else if !ok {
		res.Continue = true
		return &res, fmt.Errorf("CheckInProgressTask: task in progress")
	}

	// base info
	baseInfo, err := s.GetBaseInfo()
	if err != nil {
		return nil, fmt.Errorf("GetBaseInfo err: %s", err.Error())
	}
	res.BaseInfo = baseInfo

	// sub account live cell
	subAccountLiveCell, err := s.CheckSubAccountLiveCell(baseInfo.ContractSubAcc, parentAccountId)
	if err != nil {
		if err == ErrTaskInProgress {
			res.Continue = true
			return &res, fmt.Errorf("CheckSubAccountLiveCell: task in progress")
		}
		return nil, fmt.Errorf("CheckSubAccountLiveCell err: %s", err.Error())
	}
	res.SubAccountLiveCell = subAccountLiveCell
	return &res, nil
}

type BaseInfo struct {
	ContractDas               *core.DasContractInfo
	ContractAcc               *core.DasContractInfo
	ContractSubAcc            *core.DasContractInfo
	ContractAS                *core.DasContractInfo
	HeightCell                *core.HeightCell
	TimeCell                  *core.TimeCell
	QuoteCell                 *core.QuoteCell
	ConfigCellSubAcc          *core.DasConfigCellInfo
	ConfigCellRecordNamespace *core.DasConfigCellInfo
	ConfigCellAcc             *core.DasConfigCellInfo
	ConfigCellBuilder         *witness.ConfigCellDataBuilder
	ConfigCellDPoint          *core.DasConfigCellInfo

	ConfigCellEmoji *core.DasConfigCellInfo
	ConfigCellDigit *core.DasConfigCellInfo
	ConfigCellEn    *core.DasConfigCellInfo
	ConfigCellJa    *core.DasConfigCellInfo
	ConfigCellRu    *core.DasConfigCellInfo
	ConfigCellTr    *core.DasConfigCellInfo
	ConfigCellVi    *core.DasConfigCellInfo
	ConfigCellTh    *core.DasConfigCellInfo
	ConfigCellKo    *core.DasConfigCellInfo
}

func (s *SubAccountTxTool) GetBaseInfo() (*BaseInfo, error) {
	var bi BaseInfo
	var err error

	bi.ContractDas, err = core.GetDasContractInfo(common.DasContractNameDispatchCellType)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}

	bi.ContractAcc, err = core.GetDasContractInfo(common.DasContractNameAccountCellType)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}

	bi.ContractSubAcc, err = core.GetDasContractInfo(common.DASContractNameSubAccountCellType)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}

	bi.ContractAS, err = core.GetDasContractInfo(common.DasContractNameAlwaysSuccess)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}

	bi.HeightCell, err = s.DasCore.GetHeightCell()
	if err != nil {
		return nil, fmt.Errorf("GetHeightCell err: %s", err.Error())
	}

	bi.TimeCell, err = s.DasCore.GetTimeCell()
	if err != nil {
		return nil, fmt.Errorf("GetTimeCell err: %s", err.Error())
	}

	bi.QuoteCell, err = s.DasCore.GetQuoteCell()
	if err != nil {
		return nil, fmt.Errorf("GetQuoteCell err: %s", err.Error())
	}

	bi.ConfigCellAcc, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsAccount)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}

	bi.ConfigCellSubAcc, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}

	//
	bi.ConfigCellRecordNamespace, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsRecordNamespace)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	// config cell builder
	bi.ConfigCellBuilder, err = s.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		return nil, fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}
	bi.ConfigCellDPoint, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsDPoint)
	if err != nil {
		return nil, fmt.Errorf("ConfigCellDPoint err: %s", err.Error())
	}

	// char
	bi.ConfigCellEmoji, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsCharSetEmoji)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	bi.ConfigCellDigit, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsCharSetDigit)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	bi.ConfigCellEn, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsCharSetEn)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	bi.ConfigCellJa, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsCharSetJa)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	bi.ConfigCellRu, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsCharSetRu)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	bi.ConfigCellTr, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsCharSetTr)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	bi.ConfigCellVi, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsCharSetVi)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	bi.ConfigCellKo, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsCharSetKo)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	bi.ConfigCellTh, err = core.GetDasConfigCellInfo(common.ConfigCellTypeArgsCharSetTh)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}

	return &bi, nil
}

// check (1,0)(2,1)(0,2)(3,?)
func (s *SubAccountTxTool) CheckInProgressTask(parentAccountId string) (bool, error) {
	list, err := s.DbDao.GetTaskInProgress(parentAccountId)
	if err != nil {
		return false, fmt.Errorf("GetTaskInProgress err: %s", err.Error())
	}
	if len(list) > 0 {
		log.Warn("GetTaskInProgress (1,0)(2,1)(0,2)(3,?):", len(list))
		return false, nil
	}
	return true, nil
}

var (
	ErrTaskInProgress = errors.New("task in progress")
)

func (s *SubAccountTxTool) CheckSubAccountLiveCell(contractSubAcc *core.DasContractInfo, parentAccountId string) (*indexer.LiveCell, error) {
	searchKey := indexer.SearchKey{
		Script:     contractSubAcc.ToScript(common.Hex2Bytes(parentAccountId)),
		ScriptType: indexer.ScriptTypeType,
		ArgsLen:    0,
		Filter:     nil,
	}
	subAccLiveCells, err := s.DasCore.Client().GetCells(s.Ctx, &searchKey, indexer.SearchOrderDesc, 1, "")
	if err != nil {
		return nil, fmt.Errorf("GetCells err: %s", err.Error())
	}
	if subLen := len(subAccLiveCells.Objects); subLen != 1 {
		return nil, fmt.Errorf("sub account outpoint len: %d", subLen)
	}

	outpoint := common.OutPointStruct2String(subAccLiveCells.Objects[0].OutPoint)
	task, err := s.DbDao.GetTaskByOutpointWithParentAccountId(parentAccountId, outpoint)
	if err != nil {
		return nil, fmt.Errorf("GetTaskByOutpointWithParentAccountId err: %s", err.Error())
	} else if task.Id == 0 {
		log.Warn("not exist outpoint:", outpoint)
		if pbn, err := s.DbDao.GetParserBlockNumber(); err == nil && pbn.Id > 0 && pbn.BlockNumber > subAccLiveCells.Objects[0].BlockNumber {
			log.Warn("GetParserBlockNumber:", pbn.BlockNumber, subAccLiveCells.Objects[0].BlockNumber)
			return subAccLiveCells.Objects[0], nil
		}
		return nil, ErrTaskInProgress
	}
	return subAccLiveCells.Objects[0], nil
}

func (s *SubAccountTxTool) CheckSubAccountLiveCellForConfirm(contractSubAcc *core.DasContractInfo, parentAccountId string) (*indexer.LiveCell, error) {
	searchKey := indexer.SearchKey{
		Script:     contractSubAcc.ToScript(common.Hex2Bytes(parentAccountId)),
		ScriptType: indexer.ScriptTypeType,
		ArgsLen:    0,
		Filter:     nil,
	}
	subAccLiveCells, err := s.DasCore.Client().GetCells(s.Ctx, &searchKey, indexer.SearchOrderDesc, 1, "")
	if err != nil {
		return nil, fmt.Errorf("GetCells err: %s", err.Error())
	}
	if subLen := len(subAccLiveCells.Objects); subLen != 1 {
		return nil, fmt.Errorf("sub account outpoint len: %d", subLen)
	}

	outpoint := common.OutPointStruct2String(subAccLiveCells.Objects[0].OutPoint)
	task, err := s.DbDao.GetTaskByOutpointWithParentAccountIdForConfirm(parentAccountId, outpoint)
	if err != nil {
		return nil, fmt.Errorf("GetTaskByOutpointWithParentAccountIdForConfirm err: %s", err.Error())
	} else if task.Id == 0 {
		log.Warn("not exist outpoint:", outpoint)
		return nil, ErrTaskInProgress
	}
	return subAccLiveCells.Objects[0], nil
}
