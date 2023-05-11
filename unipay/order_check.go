package unipay

import (
	"das_sub_account/config"
	"das_sub_account/notify"
	"das_sub_account/tables"
	"fmt"
	"time"
)

func (t *ToolUniPay) RunOrderCheck() {
	tickerOrder := time.NewTicker(time.Minute * 5)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-tickerOrder.C:
				log.Info("RunOrderCheck start ...")
				if err := t.doOrderCheck(); err != nil {
					log.Error("doOrderCheck err:", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doOrderCheck", err.Error())
				}
				log.Info("RunOrderCheck end ...")
			case <-t.Ctx.Done():
				log.Info("RunOrderCheck done")
				t.Wg.Done()
				return
			}
		}
	}()
}

func (t *ToolUniPay) doOrderCheck() error {
	list, err := t.DbDao.GetNeedCheckOrderList()
	if err != nil {
		return fmt.Errorf("GetNeedCheckOrderList err: %s", err.Error())
	}
	for _, v := range list {
		switch v.ActionType {
		case tables.ActionTypeMint:
			acc, err := t.DbDao.GetAccountInfoByAccountId(v.AccountId)
			if err != nil {
				return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
			} else if acc.Id == 0 {
				continue
			}
			smtRecord, err := t.DbDao.GetSmtRecordByOrderId(v.OrderId)
			if err != nil {
				return fmt.Errorf("GetSmtRecordByOrderId err: %s", err.Error())
			}
			newStatus := tables.OrderStatusSuccess
			if smtRecord.Id == 0 || smtRecord.RecordType != tables.RecordTypeChain {
				newStatus = tables.OrderStatusFail
			}
			if err := t.DbDao.UpdateOrderStatus(v.OrderId, tables.OrderStatusDefault, newStatus); err != nil {
				return fmt.Errorf("UpdateOrderStatus err: %s[%s]", err.Error(), v.OrderId)
			}
		default:
			notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doOrderCheck", fmt.Sprintf("doOrderCheck unsupport action %d", v.ActionType))
		}
	}
	return nil
}
