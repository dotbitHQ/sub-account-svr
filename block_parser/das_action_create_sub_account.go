package block_parser

import (
	"das_sub_account/config"
	"das_sub_account/notify"
	"das_sub_account/tables"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
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

	// get parentAccountId, refOutpoint, outpoint
	parentAccountId := common.Bytes2Hex(req.Tx.Outputs[1].Type.Args)
	refOutpoint := common.OutPointStruct2String(req.Tx.Inputs[1].PreviousOutput)
	outpoint := common.OutPoint2String(req.TxHash, 1)

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
