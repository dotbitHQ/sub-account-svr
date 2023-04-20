package unipay

import (
	"context"
	"das_sub_account/config"
	"das_sub_account/dao"
	"das_sub_account/notify"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/scorpiotzh/mylog"
	"sync"
	"time"
)

var (
	log = mylog.NewLogger("unipay", mylog.LevelDebug)
)

type ToolUniPay struct {
	Ctx     context.Context
	Wg      *sync.WaitGroup
	DbDao   *dao.DbDao
	DasCore *core.DasCore
}

func (t *ToolUniPay) RunUniPay() {
	tickerRefund := time.NewTicker(time.Minute * 10)
	tickerSearchStatus := time.NewTicker(time.Minute * 5)

	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-tickerRefund.C:
				log.Info("doRefund start")
				if err := t.doRefund(); err != nil {
					log.Errorf("doRefund err: %s", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRefund", err.Error())
				}
				log.Info("doRefund end")
			case <-tickerSearchStatus.C:
				log.Info("doConfirmStatus start")
				if err := t.doConfirmStatus(); err != nil {
					log.Errorf("doConfirmStatus err: %s", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doConfirmStatus", err.Error())
				}
				log.Info("doConfirmStatus end")
			case <-t.Ctx.Done():
				log.Info("RunRefund done")
				t.Wg.Done()
				return
			}
		}
	}()
}
