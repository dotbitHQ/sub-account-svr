package task

import (
	"context"
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/dao"
	"das_sub_account/notify"
	"das_sub_account/txtool"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"sync"
	"time"
)

var log = logger.NewLogger("task", logger.LevelDebug)

type SmtTask struct {
	Ctx          context.Context
	Wg           *sync.WaitGroup
	DbDao        *dao.DbDao
	DasCore      *core.DasCore
	TxTool       *txtool.SubAccountTxTool
	RC           *cache.RedisCache
	MaxRetry     int
	SmtServerUrl string
}

// task_id='' -> task_id!=''
func (t *SmtTask) RunUpdateSubAccountTaskDistribution() {
	tickerDistribution := time.NewTicker(time.Minute)
	t.Wg.Add(1)
	go func() {
		defer http_api.RecoverPanic()
		for {
			select {
			case <-tickerDistribution.C:
				log.Debug("doUpdateDistribution start ...")
				if err := t.doUpdateDistribution(); err != nil {
					log.Error("doUpdateDistribution err:", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doUpdateDistribution", err.Error())
				}
				log.Debug("doUpdateDistribution end ...")
			case <-t.Ctx.Done():
				log.Debug("task doUpdateDistribution done")
				t.Wg.Done()
				return
			}
		}
	}()
}

// smt_status,tx_status: (2,1)->(3,3)
func (t *SmtTask) RunTaskCheckTx() {
	tickerCheckTx := time.NewTicker(time.Second * 15)
	t.Wg.Add(1)
	go func() {
		defer http_api.RecoverPanic()
		for {
			select {
			case <-tickerCheckTx.C:
				log.Debug("doCheckTx start ...")
				if err := t.doCheckTx(); err != nil {
					log.Error("doCheckTx err:", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doCheckTx", err.Error())
				}
				log.Debug("doCheckTx end ...")
			case <-t.Ctx.Done():
				log.Debug("task Check Tx done")
				t.Wg.Done()
				return
			}
		}
	}()
}

// smt_status,tx_status: (0,2)->(2,2)
func (t *SmtTask) RunTaskConfirmOtherTx() {
	tickerOther := time.NewTicker(time.Second * 7)
	t.Wg.Add(1)
	go func() {
		defer http_api.RecoverPanic()
		for {
			select {
			case <-tickerOther.C:
				log.Debug("doConfirmOtherTx start ...")
				if err := t.doConfirmOtherTx(); err != nil {
					log.Error("doConfirmOtherTx err:", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doConfirmOtherTx", err.Error())
				}
				log.Debug("doConfirmOtherTx end ...")
			case <-t.Ctx.Done():
				log.Debug("task confirm other tx done")
				t.Wg.Done()
				return
			}
		}
	}()
}

// smt_status,tx_status: (3,?)->(4,?)
func (t *SmtTask) RunTaskRollback() {
	tickerRollback := time.NewTicker(time.Second * 5)
	t.Wg.Add(1)
	go func() {
		defer http_api.RecoverPanic()
		for {
			select {
			case <-tickerRollback.C:
				log.Debug("doRollback start ...")
				if err := t.doRollback(); err != nil {
					log.Error("doRollback err:", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRollback", err.Error())
				}
				log.Debug("doRollback end ...")
			case <-t.Ctx.Done():
				log.Debug("task rollback done")
				t.Wg.Done()
				return
			}
		}
	}()
}

// update: smt_status,tx_status: (0,0)->(2,1)
func (t *SmtTask) RunUpdateSubAccountTask() {
	ticker := time.NewTicker(time.Second * 6)
	t.Wg.Add(1)
	go func() {
		defer http_api.RecoverPanic()
		for {
			select {
			case <-ticker.C:
				log.Debug("RunUpdateSubAccountTask start ...")
				if err := t.doBatchUpdateSubAccountTask(common.DasActionUpdateSubAccount); err != nil {
					log.Error("RunUpdateSubAccountTask err:", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "RunUpdateSubAccountTask", err.Error())
				}
				log.Debug("RunUpdateSubAccountTask end ...")
			case <-t.Ctx.Done():
				log.Debug("RunUpdateSubAccountTask done")
				t.Wg.Done()
				return
			}
		}
	}()
}
