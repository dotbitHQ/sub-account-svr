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
	"github.com/scorpiotzh/mylog"
	"go.mongodb.org/mongo-driver/mongo"
	"sync"
	"time"
)

var log = mylog.NewLogger("task", mylog.LevelDebug)

type SmtTask struct {
	Ctx      context.Context
	Wg       *sync.WaitGroup
	DbDao    *dao.DbDao
	DasCore  *core.DasCore
	Mongo    *mongo.Client
	TxTool   *txtool.SubAccountTxTool
	RC       *cache.RedisCache
	MaxRetry int
}

// task_id='' -> task_id!=''
func (t *SmtTask) RunUpdateSubAccountTaskDistribution() {
	tickerDistribution := time.NewTicker(time.Minute)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-tickerDistribution.C:
				log.Info("doUpdateDistribution start ...")
				if err := t.doUpdateDistribution(); err != nil {
					log.Error("doUpdateDistribution err:", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doUpdateDistribution", err.Error())
				}
				log.Info("doUpdateDistribution end ...")
			case <-t.Ctx.Done():
				log.Info("task doUpdateDistribution done")
				t.Wg.Done()
				return
			}
		}
	}()
}

//func (t *SmtTask) RunTaskDistribution() {
//	tickerDistribution := time.NewTicker(time.Minute * 3)
//	t.Wg.Add(1)
//	go func() {
//		for {
//			select {
//			case <-tickerDistribution.C:
//				log.Info("doDistribution start ...")
//				if err := t.doDistribution(); err != nil {
//					log.Error("doDistribution err:", err.Error())
//					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doDistribution", err.Error())
//				}
//				log.Info("doDistribution end ...")
//			case <-t.Ctx.Done():
//				log.Info("task Distribution done")
//				t.Wg.Done()
//				return
//			}
//		}
//	}()
//}

//func (t *SmtTask) RunMintTaskDistribution() {
//	tickerDistribution := time.NewTicker(time.Minute * 3)
//	t.Wg.Add(1)
//	go func() {
//		for {
//			select {
//			case <-tickerDistribution.C:
//				log.Info("RunMintTaskDistribution start ...")
//				if err := t.doMintDistribution(); err != nil {
//					log.Error("doMintDistribution err:", err.Error())
//					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doMintDistribution", err.Error())
//				}
//				log.Info("RunMintTaskDistribution end ...")
//			case <-t.Ctx.Done():
//				log.Info("task RunMintTaskDistribution done")
//				t.Wg.Done()
//				return
//			}
//		}
//	}()
//}

// smt_status,tx_status: (2,1)->(3,3)
func (t *SmtTask) RunTaskCheckTx() {
	tickerCheckTx := time.NewTicker(time.Second * 15)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-tickerCheckTx.C:
				log.Info("doCheckTx start ...")
				if err := t.doCheckTx(); err != nil {
					log.Error("doCheckTx err:", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doCheckTx", err.Error())
				}
				log.Info("doCheckTx end ...")
			case <-t.Ctx.Done():
				log.Info("task Check Tx done")
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
		for {
			select {
			case <-tickerOther.C:
				log.Info("doConfirmOtherTx start ...")
				if err := t.doConfirmOtherTx(); err != nil {
					log.Error("doConfirmOtherTx err:", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doConfirmOtherTx", err.Error())
				}
				log.Info("doConfirmOtherTx end ...")
			case <-t.Ctx.Done():
				log.Info("task confirm other tx done")
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
		for {
			select {
			case <-tickerRollback.C:
				log.Info("doRollback start ...")
				if err := t.doRollback(); err != nil {
					log.Error("doRollback err:", err.Error())
					notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "doRollback", err.Error())
				}
				log.Info("doRollback end ...")
			case <-t.Ctx.Done():
				log.Info("task rollback done")
				t.Wg.Done()
				return
			}
		}
	}()
}

// edit: smt_status,tx_status: (0,0)->(2,1)
func (t *SmtTask) RunEditSubAccountTask() {
	ticker := time.NewTicker(time.Second * 6)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Info("RunEditSubAccountTask start ...")
				if !config.Cfg.Das.IsEditTaskClosed {
					if err := t.doTask(common.DasActionEditSubAccount); err != nil {
						log.Error("RunEditSubAccountTask err:", err.Error())
						notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "RunEditSubAccountTask", err.Error())
					}
				}
				log.Info("RunEditSubAccountTask end ...")
			case <-t.Ctx.Done():
				log.Info("RunEditSubAccountTask done")
				t.Wg.Done()
				return
			}
		}
	}()
}

// create: smt_status,tx_status: (0,0)->(2,1)
func (t *SmtTask) RunCreateSubAccountTask() {
	ticker := time.NewTicker(time.Second * 7)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Info("RunCreateSubAccountTask start ...")
				if !config.Cfg.Das.IsCreateTaskClosed && t.TxTool.ServerScript != nil {
					if err := t.doTask(common.DasActionCreateSubAccount); err != nil {
						log.Error("RunCreateSubAccountTask err:", err.Error())
						notify.SendLarkTextNotify(config.Cfg.Notify.LarkErrorKey, "RunCreateSubAccountTask", err.Error())
					}
				}
				log.Info("RunCreateSubAccountTask end ...")
			case <-t.Ctx.Done():
				log.Info("RunCreateSubAccountTask done")
				t.Wg.Done()
				return
			}
		}
	}()
}

// smt_status,tx_status: (1,0) check user create sub account
func (t *SmtTask) RunCheckError() {
	tickerError := time.NewTicker(time.Second * 10)
	t.Wg.Add(1)
	go func() {
		for {
			select {
			case <-tickerError.C:
				log.Info("doCheckError start ...")
				if err := t.doCheckError(); err != nil {
					log.Error("doCheckError err:", err.Error())
				}
				log.Info("doCheckError end ...")
			case <-t.Ctx.Done():
				log.Info("task check error done")
				t.Wg.Done()
				return
			}
		}
	}()
}
