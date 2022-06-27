package block_parser

import (
	"das_sub_account/config"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
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

	builderMap, err := witness.AccountIdCellDataBuilderFromTx(req.Tx, common.DataTypeOld)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}
	var builder *witness.AccountCellDataBuilder
	for _, v := range builderMap {
		if v.Index == 1 {
			builder = v
		}
	}
	if builder != nil && builder.EnableSubAccount == 1 {
		if err = b.Mongo.Database(config.Cfg.DB.Mongo.SmtDatabase).Collection(builder.AccountId).Drop(b.Ctx); err != nil {
			resp.Err = fmt.Errorf("Mongo Drop err: %s ", err.Error())
			return
		}
	}

	return
}
