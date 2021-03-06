package task

import (
	"das_sub_account/config"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"time"
)

func (t *SmtTask) doCheckError() error {
	list, err := t.DbDao.GetNeedDoCheckErrorTaskList()
	if err != nil {
		return fmt.Errorf("GetNeedDoCheckErrorTaskList err: %s", err.Error())
	}
	var needRollbackIds []uint64
	for _, v := range list {
		// timestamp > 3min
		timestamp := time.Now().Add(-time.Minute*3).UnixNano() / 1e6
		if config.Cfg.Server.Net != common.DasNetTypeMainNet {
			timestamp = time.Now().Add(-time.Minute).UnixNano() / 1e6
		}
		if v.Timestamp < timestamp {
			needRollbackIds = append(needRollbackIds, v.Id)
		}
	}
	if len(needRollbackIds) > 0 {
		if err := t.DbDao.UpdateTaskStatusToRollback(needRollbackIds); err != nil {
			return fmt.Errorf("UpdateTaskStatusToRollback err: %s", err.Error())
		}
	}
	return nil
}
