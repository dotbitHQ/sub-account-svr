package block_parser

import (
	"das_sub_account/config"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/witness"
	"time"
)

func (b *BlockParser) DasActionRecycleExpiredAccount(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	res, err := b.DasCore.Client().GetTransaction(b.Ctx, req.Tx.Inputs[1].PreviousOutput.TxHash)
	if err != nil {
		resp.Err = fmt.Errorf("GetTransaction err: %s", err.Error())
		return
	}
	if isCV, err := isCurrentVersionTx(res.Transaction, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version recycle expired account tx")
		return
	}
	log.Info("DasActionRecycleExpiredAccount:", req.BlockNumber, req.TxHash)

	builder, err := witness.AccountCellDataBuilderFromTx(res.Transaction, common.DataTypeNew)
	if err != nil {
		resp.Err = fmt.Errorf("AccountCellDataBuilderFromTx err: %s", err.Error())
		return
	}
	builderConfig, err := b.DasCore.ConfigCellDataBuilderByTypeArgs(common.ConfigCellTypeArgsAccount)
	if err != nil {
		resp.Err = fmt.Errorf("ConfigCellDataBuilderByTypeArgs err: %s", err.Error())
		return
	}
	gracePeriod, err := builderConfig.ExpirationGracePeriod()
	if err != nil {
		resp.Err = fmt.Errorf("ExpirationGracePeriod err: %s", err.Error())
		return
	}

	if builder.Status != 0 {
		resp.Err = fmt.Errorf("ActionRecycleExpiredAccount: account is not normal status")
		return
	}
	if builder.ExpiredAt+uint64(gracePeriod) > uint64(time.Now().Unix()) {
		resp.Err = fmt.Errorf("ActionRecycleExpiredAccount: account has not expired yet")
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
