package task

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/nervosnetwork/ckb-sdk-go/types"
)

func (t *SmtTask) doCheckTx() error {
	list, err := t.DbDao.GetNeedDoCheckTxTaskList()
	if err != nil {
		return fmt.Errorf("GetNeedDoCheckTxTaskList err: %s", err.Error())
	}
	var rollbackList []uint64
	var mapRejected = make(map[string]struct{})
	for _, v := range list {
		log.Info(v.TaskId, v.RefOutpoint, v.Outpoint)

		// check with rejected outpoint map
		if _, ok := mapRejected[v.RefOutpoint]; ok {
			rollbackList = append(rollbackList, v.Id)
			mapRejected[v.Outpoint] = struct{}{}
			continue
		}

		outpoint := common.String2OutPointStruct(v.Outpoint)
		res, err := t.DasCore.Client().GetTransaction(t.Ctx, outpoint.TxHash)
		if err != nil {
			return fmt.Errorf("GetTransaction err: %s", err.Error())
		}

		if res.TxStatus.Status == types.TransactionStatusRejected {
			rollbackList = append(rollbackList, v.Id)
			mapRejected[v.Outpoint] = struct{}{}
		}
	}
	if err := t.DbDao.UpdateTaskStatusToRejected(rollbackList); err != nil {
		return fmt.Errorf("UpdateTaskStatusToRejected err: %s", err.Error())
	}
	return nil
}
