package unipay

import (
	"context"
	"das_sub_account/dao"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"sync"
)

var (
	log = logger.NewLogger("unipay", logger.LevelDebug)
)

type ToolUniPay struct {
	Ctx     context.Context
	Wg      *sync.WaitGroup
	DbDao   *dao.DbDao
	DasCore *core.DasCore
}
