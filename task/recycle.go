package task

import (
	"das_sub_account/config"
	"das_sub_account/notify"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"time"
)

func (t *SmtTask) RunRecycleSubAccount() {
	tickerRecycle := time.NewTicker(time.Minute * 10)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-tickerRecycle.C:
				log.Info("RunRecycleSubAccount start ...")
				if err := t.recycleSubAccount(); err != nil {
					log.Error("recycleSubAccount err:", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "recycleSubAccount", err.Error())
				}
				log.Info("RunRecycleSubAccount end ...")
			case <-t.Ctx.Done():
				log.Info("RunRecycleSubAccount task done")
				t.Wg.Done()
				return
			}
		}
	}()
}

func (t *SmtTask) recycleSubAccount() error {
	if !config.Cfg.Server.RecycleSwitch {
		return nil
	}
	accConfigCell, err := t.DasCore.ConfigCellDataBuilderByTypeArgs(common.ConfigCellTypeArgsAccount)
	if err != nil {
		return fmt.Errorf("recycleSubAccount err: %s", err.Error())
	}
	expirationGracePeriod, err := accConfigCell.ExpirationGracePeriod()
	if err != nil {
		return fmt.Errorf("ExpirationGracePeriod err: %s", err.Error())
	}
	timeCell, err := t.DasCore.GetTimeCell()
	if err != nil {
		return fmt.Errorf("GetTimeCell err: %s", err.Error())
	}
	timestamp := timeCell.Timestamp()
	log.Info("recycleSubAccount:", timestamp, expirationGracePeriod)
	timestamp = timestamp - int64(expirationGracePeriod)
	if timestamp <= 0 {
		return fmt.Errorf("timestamp is 0")
	}
	// get need to recycle sub-account list
	list, err := t.DbDao.GetNeedToRecycleList(timestamp)
	if err != nil {
		return fmt.Errorf("GetNeedToRecycleList err: %s", err.Error())
	}

	// check recycle pending
	var smtRecordList []tables.TableSmtRecordInfo
	for _, v := range list {
		smtRecord, err := t.DbDao.GetRecycleSmtRecord(v.AccountId)
		if err != nil {
			return fmt.Errorf("GetRecycleSmtRecord err: %s", err.Error())
		} else if smtRecord.Id != 0 {
			continue
		}
		tmpSmtRecord := tables.TableSmtRecordInfo{
			SvrName:         "",
			AccountId:       v.AccountId,
			Nonce:           v.Nonce + 1,
			RecordType:      tables.RecordTypeDefault,
			MintType:        tables.MintTypeDefault,
			Action:          common.DasActionUpdateSubAccount,
			ParentAccountId: v.ParentAccountId,
			Account:         v.Account,
			Timestamp:       time.Now().UnixMilli(),
			SubAction:       common.SubActionRecycle,
		}
		smtRecordList = append(smtRecordList, tmpSmtRecord)
	}
	if len(smtRecordList) == 0 {
		return nil
	}
	if err := t.DbDao.CreateRecycleSmtRecordList(smtRecordList); err != nil {
		return fmt.Errorf("CreateRecycleSmtRecordList err: %s", err.Error())
	}
	return nil
}
