package block_parser

import (
	"das_sub_account/config"
	"das_sub_account/notify"
	"das_sub_account/tables"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/core"
)

func (b *BlockParser) DasActionCreateSubAccount(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DASContractNameSubAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version create sub account tx")
		return
	}
	log.Info("DasActionCreateSubAccount:", req.BlockNumber, req.TxHash)

	// check sub-account config custom-script-args or not
	contractSub, err := core.GetDasContractInfo(common.DASContractNameSubAccountCellType)
	if err != nil {
		resp.Err = fmt.Errorf("GetDasContractInfo err: %s", err.Error())
		return
	}
	var parentAccountId, outpoint, refOutpoint string
	for i, v := range req.Tx.Outputs {
		if v.Type != nil && contractSub.IsSameTypeId(v.Type.CodeHash) {
			parentAccountId = common.Bytes2Hex(v.Type.Args)
			outpoint = common.OutPoint2String(req.TxHash, uint(i))
		}
	}
	for _, v := range req.Tx.Inputs {
		res, err := b.DasCore.Client().GetTransaction(b.Ctx, v.PreviousOutput.TxHash)
		if err != nil {
			resp.Err = fmt.Errorf("GetTransaction err: %s", err.Error())
			return
		}
		tmp := res.Transaction.Outputs[v.PreviousOutput.Index]
		if tmp.Type != nil && contractSub.IsSameTypeId(tmp.Type.CodeHash) {
			refOutpoint = common.OutPointStruct2String(v.PreviousOutput)
			break
		}
	}

	// get sub account
	taskInfo, smtRecordList, err := getTaskAndSmtRecords(&req, parentAccountId, refOutpoint, outpoint)
	if err != nil {
		resp.Err = fmt.Errorf("getTaskAndSmtRecords err: %s", err.Error())
		return
	}

	// get self task
	selfTask, err := b.DbDao.GetTaskByRefOutpointAndOutpoint(refOutpoint, outpoint)
	if err != nil {
		resp.Err = fmt.Errorf("GetTaskByRefOutpointAndOutpoint err: %s", err.Error())
		return
	}

	// add task and smt records
	if selfTask.TaskId != "" {
		// maybe rollback
		if err := b.DbDao.UpdateToChainTask(selfTask.TaskId, req.BlockNumber); err != nil {
			resp.Err = fmt.Errorf("UpdateToChainTask err: %s", err.Error())
			return
		}
	} else {
		if err := b.DbDao.CreateChainTask(taskInfo, smtRecordList, selfTask.TaskId); err != nil {
			resp.Err = fmt.Errorf("CreateChainTask err: %s", err.Error())
			return
		}
	}

	doNotify(smtRecordList)

	return
}

func doNotify(smtRecordList []tables.TableSmtRecordInfo) {
	content := ""
	count := 0
	var contentList []string
	for _, v := range smtRecordList {
		account := v.Account
		registerYears := uint64(1)
		registerYears = v.RegisterYears

		content += fmt.Sprintf(`%s registered for %d year(s)
`, account, registerYears)
		count++
		if count == 30 {
			contentList = append(contentList, content)
			content = ""
			count = 0
		}
	}
	if content != "" {
		contentList = append(contentList, content)
	}
	go func() {
		for _, v := range contentList {
			notify.SendLarkTextNotify(config.Cfg.Notify.LarkCreateSubAccountKey, "", v)
		}
	}()
}
