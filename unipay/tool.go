package unipay

import (
	"context"
	"das_sub_account/dao"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/scorpiotzh/mylog"
	"sync"
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
