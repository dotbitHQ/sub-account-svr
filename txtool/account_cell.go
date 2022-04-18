package txtool

import (
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/types"
)

func (s *SubAccountTxTool) getAccountByOutpoint(accOutpoint *types.OutPoint, accountId string) (*types.CellOutput, []byte, []byte, error) {
	tx, err := s.DasCore.Client().GetTransaction(s.Ctx, accOutpoint.TxHash)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("GetTransaction err: %s", err.Error())
	}
	mapAcc, err := witness.AccountIdCellDataBuilderFromTx(tx.Transaction, common.DataTypeNew)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("AccountIdCellDataBuilderFromTx err: %s", err.Error())
	}
	if item, ok := mapAcc[accountId]; !ok {
		return nil, nil, nil, fmt.Errorf("not exist acc builder: %s", accountId)
	} else {
		accWitness, accData, err := item.GenWitness(&witness.AccountCellParam{
			OldIndex: 0,
			NewIndex: 0,
			Action:   common.DasActionCreateSubAccount,
		})
		accData = append(accData, tx.Transaction.OutputsData[item.Index][32:]...)
		accCellOutput := types.CellOutput{
			Capacity: tx.Transaction.Outputs[accOutpoint.Index].Capacity,
			Lock:     tx.Transaction.Outputs[accOutpoint.Index].Lock,
			Type:     tx.Transaction.Outputs[accOutpoint.Index].Type,
		}
		return &accCellOutput, accWitness, accData, err
	}
}
