package txtool

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/core"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"sync"
)

type paramBalance struct {
	taskInfo     *tables.TableTaskInfo
	dasLock      *types.Script
	dasType      *types.Script
	needCapacity uint64
}

var balanceLock sync.Mutex

func (s *SubAccountTxTool) getBalanceCell(p *paramBalance) (uint64, []*indexer.LiveCell, error) {
	if p.needCapacity == 0 {
		return 0, nil, nil
	}
	needCapacity := p.needCapacity
	if p.taskInfo.TaskType == tables.TaskTypeDelegate {
		balanceLock.Lock()
		defer balanceLock.Unlock()
		needCapacity += 400 * common.OneCkb
	}

	liveCells, total, err := core.GetSatisfiedCapacityLiveCellWithOrder(s.DasCore.Client(), s.DasCache, p.dasLock, p.dasType, needCapacity, common.DasLockWithBalanceTypeOccupiedCkb, indexer.SearchOrderAsc)
	if err != nil {
		return 0, nil, fmt.Errorf("GetSatisfiedCapacityLiveCellWithOrder err: %s", err.Error())
	}
	if p.taskInfo.TaskType == tables.TaskTypeDelegate {
		var outpoints []string
		for _, v := range liveCells {
			outpoints = append(outpoints, common.OutPointStruct2String(v.OutPoint))
		}
		s.DasCache.AddOutPoint(outpoints)
	}
	return total - p.needCapacity, liveCells, nil
}
