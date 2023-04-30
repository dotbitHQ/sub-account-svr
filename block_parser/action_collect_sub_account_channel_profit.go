package block_parser

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
)

func (b *BlockParser) ActionCollectSubAccountChannelProfit(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DASContractNameSubAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		return
	}
	log.Info("ActionCollectSubAccountChannelProfit:", req.BlockNumber, req.TxHash)

	if err := b.DbDao.UpdateTxStatusByOutpoint(common.OutPoint2String(req.TxHash, 0), tables.TxStatusCommitted); err != nil {
		resp.Err = fmt.Errorf("UpdateTxStatusByOutpoint err: %s", err.Error())
	}
	return
}
