package block_parser

import (
	"das_sub_account/config"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/witness"
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

	builder, err := witness.AccountCellDataBuilderFromTx(req.Tx, common.DataTypeOld)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}
	if builder.EnableSubAccount == 1 {
		if err = b.Mongo.Database(config.Cfg.DB.Mongo.SmtDatabase).Collection(builder.AccountId).Drop(b.Ctx); err != nil {
			resp.Err = fmt.Errorf("Mongo Drop err: %s ", err.Error())
			return
		}
	}

	return
}
