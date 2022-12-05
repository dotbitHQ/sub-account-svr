package txtool

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
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
	balanceLock.Lock()
	defer balanceLock.Unlock()

	liveCells, total, err := s.DasCore.GetBalanceCells(&core.ParamGetBalanceCells{
		DasCache:          s.DasCache,
		LockScript:        p.dasLock,
		CapacityNeed:      p.needCapacity,
		CapacityForChange: common.DasLockWithBalanceTypeOccupiedCkb,
		SearchOrder:       indexer.SearchOrderAsc,
	})
	if err != nil {
		return 0, nil, fmt.Errorf("GetBalanceCells err: %s", err.Error())
	}

	var outpoints []string
	for _, v := range liveCells {
		outpoints = append(outpoints, common.OutPointStruct2String(v.OutPoint))
	}
	s.DasCache.AddOutPoint(outpoints)

	return total - p.needCapacity, liveCells, nil
}
