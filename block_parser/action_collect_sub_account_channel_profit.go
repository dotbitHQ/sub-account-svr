package block_parser

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"gorm.io/gorm"
)

func (b *BlockParser) ActionCollectSubAccountChannelProfit(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DASContractNameSubAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		return
	}
	log.Info("ActionCollectSubAccountChannelProfit:", req.BlockNumber, req.TxHash)

	outpoint := common.OutPoint2String(req.TxHash, 0)
	resp.Err = b.DbDao.Transaction(func(tx *gorm.DB) error {
		return tx.Model(&tables.TableTaskInfo{}).Where("outpoint=?", outpoint).Updates(map[string]interface{}{
			"tx_status":    tables.TxStatusCommitted,
			"block_number": req.BlockNumber,
		}).Error
	})
	return
}
