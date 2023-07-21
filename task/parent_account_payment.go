package task

import (
	"bytes"
	"das_sub_account/config"
	"das_sub_account/notify"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"math"
	"time"
)

func (t *SmtTask) RunParentAccountPayment() error {
	secondParser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.DowOptional | cron.Descriptor)
	c := cron.New(cron.WithParser(secondParser), cron.WithChain())
	if _, err := c.AddFunc("0 30 10 5 * ?", func() {
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
		log.Error("doParentAccountPayment StatisticsParentAccountPayment err:", err.Error())
		return err
	}
	log.Infof("doParentAccountPayment recordsNew: %s", toolib.JsonString(recordsNew))

	if len(recordsNew) == 0 {
		return nil
	}

	buf := bytes.NewBufferString("")
	for _, v := range recordsNew {
		for _, record := range v {
			amount := record.Amount.DivRound(decimal.NewFromInt(int64(math.Pow10(int(record.Decimals)))), record.Decimals)
			buf.WriteString(fmt.Sprintf("-Account: %s\n", record.Account))
			buf.WriteString(fmt.Sprintf("-%s: %s Amount: %s \n\n", record.TokenId, record.Address, amount))
		}
		buf.WriteString("\n")
	}
	notify.SendLarkTextNotify(config.Cfg.Notify.LarkParentAccountPaymentKey, "OwnerPaymentInfo", buf.String())
	return nil
}
