package block_parser

import (
	"das_sub_account/dao"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/nervosnetwork/ckb-sdk-go/types"
)

func (b *BlockParser) registerTransactionHandle() {
	b.mapTransactionHandle = make(map[string]FuncTransactionHandle)
	b.mapTransactionHandle[common.DasActionConfig] = b.DasActionConfig

	b.mapTransactionHandle[common.DasActionEnableSubAccount] = b.DasActionEnableSubAccount
	b.mapTransactionHandle[common.DasActionCreateSubAccount] = b.DasActionCreateSubAccount
	b.mapTransactionHandle[common.DasActionEditSubAccount] = b.DasActionEditSubAccount
	b.mapTransactionHandle[common.DasActionRenewSubAccount] = b.DasActionRenewSubAccount     // todo
	b.mapTransactionHandle[common.DasActionRecycleSubAccount] = b.DasActionRecycleSubAccount // todo
	b.mapTransactionHandle[common.DasActionRecycleExpiredAccount] = b.DasActionRecycleExpiredAccount
	b.mapTransactionHandle[common.DasActionLockSubAccountForCrossChain] = b.DasActionRecycleSubAccount   // todo
	b.mapTransactionHandle[common.DasActionUnlockSubAccountForCrossChain] = b.DasActionRecycleSubAccount // todo
}

func isCurrentVersionTx(tx *types.Transaction, name common.DasContractName) (bool, error) {
	contract, err := core.GetDasContractInfo(name)
	if err != nil {
		return false, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	isCV := false
	for _, v := range tx.Outputs {
		if v.Type == nil {
			continue
		}
		if contract.IsSameTypeId(v.Type.CodeHash) {
			isCV = true
			break
		}
	}
	return isCV, nil
}

type FuncTransactionHandleReq struct {
	DbDao          *dao.DbDao
	Tx             *types.Transaction
	TxHash         string
	BlockNumber    uint64
	BlockTimestamp int64
	Action         common.DasAction
}

type FuncTransactionHandleResp struct {
	ActionName string
	Err        error
}

type FuncTransactionHandle func(FuncTransactionHandleReq) FuncTransactionHandleResp
