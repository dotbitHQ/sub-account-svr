package txtool

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"sync"
)

type ParamBalance struct {
	DasLock      *types.Script
	DasType      *types.Script
	NeedCapacity uint64
}

var balanceLock sync.Mutex

func (s *SubAccountTxTool) GetBalanceCell(p *ParamBalance) (uint64, []*indexer.LiveCell, error) {
	if p.NeedCapacity == 0 {
		return 0, nil, nil
	}
	balanceLock.Lock()
	defer balanceLock.Unlock()

	liveCells, total, err := s.DasCore.GetBalanceCells(&core.ParamGetBalanceCells{
		DasCache:          s.DasCache,
		LockScript:        p.DasLock,
		CapacityNeed:      p.NeedCapacity,
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

	return total - p.NeedCapacity, liveCells, nil
}
