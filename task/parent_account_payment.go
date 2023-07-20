package task

import (
	"bytes"
	"das_sub_account/config"
	"das_sub_account/notify"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/shopspring/decimal"
	"math"
	"time"
)

func (t *SmtTask) RunParentAccountPayment() error {
	c := cron.New()
	// 0 30 10 5 * *
	if _, err := c.AddFunc("0 * * * * *", func() {
		if err := t.doParentAccountPayment(); err != nil {
			log.Error("doParentAccountPayment err:", err.Error())
		}
	}); err != nil {
		log.Error("RunParentAccountPayment err:", err.Error())
		return err
	}
	c.Start()
	return nil
}

func (t *SmtTask) doParentAccountPayment() error {
	now := time.Now()
	endTime := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	recordsNew, err := t.TxTool.StatisticsParentAccountPayment("", false, endTime)
	if err != nil {
		return err
	}

	if len(recordsNew) == 0 {
		return nil
	}

	buf := bytes.NewBufferString("")
	for _, v := range recordsNew {
		amount := v.Amount.DivRound(decimal.NewFromInt(int64(math.Pow10(int(v.Decimals)))), v.Decimals)
		buf.WriteString(fmt.Sprintf("-Account: %s\n", v.Account))
		buf.WriteString(fmt.Sprintf("-%s: %s Amount: %s \n\n", v.TokenId, v.Address, amount))
	}
	notify.SendLarkTextNotify(config.Cfg.Notify.LarkParentAccountPaymentKey, "OwnerPaymentInfo", buf.String())
	return nil
}
