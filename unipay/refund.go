package unipay

import (
	"context"
	"das_sub_account/config"
	"das_sub_account/dao"
	"das_sub_account/notify"
	"fmt"
	"github.com/scorpiotzh/mylog"
	"sync"
	"time"
)

var (
	log = mylog.NewLogger("unipay", mylog.LevelDebug)
)

type ToolRefund struct {
	Ctx   context.Context
	Wg    *sync.WaitGroup
	DbDao *dao.DbDao
}

func (t *ToolRefund) RunRefund() {
	if !config.Cfg.Server.RefundSwitch {
		return
	}

	tickerRefund := time.NewTicker(time.Minute * 10)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-tickerRefund.C:
				log.Info("doRefund start")
				if err := t.doRefund(); err != nil {
					log.Error("doRefund err: %s", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRefund", err.Error())
				}
				log.Info("doRefund end")
			case <-t.Ctx.Done():
				log.Info("RunRefund done")
				t.Wg.Done()
				return
			}
		}
	}()
}

func (t *ToolRefund) doRefund() error {
	//get payment list
	list, err := t.DbDao.GetUnRefundList()
	if err != nil {
		return fmt.Errorf("GetUnRefundList err: %s", err.Error())
	}

	//call unipay to refund
	var req ReqOrderRefund
	req.BusinessId = BusinessIdAutoSubAccount
	var ids []uint64
	for _, v := range list {
		ids = append(ids, v.Id)
		req.RefundList = append(req.RefundList, RefundInfo{
			OrderId: v.OrderId,
			PayHash: v.PayHash,
		})
	}

	_, err = RefundOrder(req)
	if err != nil {
		return fmt.Errorf("RefundOrder err: %s", err.Error())
	}

	if err = t.DbDao.UpdateRefundStatusToRefundIng(ids); err != nil {
		return fmt.Errorf("UpdateRefundStatusToRefundIng err: %s", err.Error())
	}

	return nil
}
