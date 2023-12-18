package block_parser

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
)

func (b *BlockParser) DasActionApproval(req FuncTransactionHandleReq) (resp FuncTransactionHandleResp) {
	if isCV, err := isCurrentVersionTx(req.Tx, common.DasContractNameAccountCellType); err != nil {
		resp.Err = fmt.Errorf("isCurrentVersion err: %s", err.Error())
		return
	} else if !isCV {
		log.Warn("not current version create sub account tx")
		return
	}
	log.Info("DasActionApproval:", req.BlockNumber, req.TxHash)

	_, outpoint, err := b.getOutpoint(req, common.DasContractNameAccountCellType)
	if err != nil {
		resp.Err = fmt.Errorf("getOutpoint err: %s", err.Error())
		return
	}
	if err := b.DbDao.UpdatePendingStatusToConfirm(req.Action, outpoint, req.BlockNumber, uint64(req.BlockTimestamp)); err != nil {
		resp.Err = fmt.Errorf("UpdatePendingStatusToConfirm err: %s", err.Error())
		return
	}
	return
}
