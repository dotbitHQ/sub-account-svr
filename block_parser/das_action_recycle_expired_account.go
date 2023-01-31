package block_parser

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/witness"
	"time"
)

func (b *BlockParser) DasActionRecycleExpiredAccount(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version recycle expired account tx")
		return
	}
	log.Info("DasActionRecycleExpiredAccount:", req.BlockNumber, req.TxHash)

	builderMapOld, err := witness.AccountIdCellDataBuilderFromTx(req.Tx, common.DataTypeOld)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}
	builderMapNew, err := witness.AccountIdCellDataBuilderFromTx(req.Tx, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}
	var builder *witness.AccountCellDataBuilder
	for k, _ := range builderMapOld {
		if _, ok := builderMapNew[k]; !ok {
			builder = builderMapOld[k]
			break
		}
	}

	if builder != nil && builder.EnableSubAccount == 1 {
		tree := smt.NewSmtSrv(*b.SmtServerUrl, builder.AccountId)
		ok, err := tree.DeleteSmtWithTimeOut(time.Minute * 5)
		if err != nil {
			resp.Err = fmt.Errorf("Smt Drop err: %s ", err.Error())
			return
		} else if !ok {
			resp.Err = fmt.Errorf("Smt Drop fail: %v ", ok)
			return
		}
	}

	return
}
