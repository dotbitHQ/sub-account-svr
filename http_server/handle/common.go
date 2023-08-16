package handle

import (
	"crypto/md5"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"time"
)

type Pagination struct {
	Page int `json:"page"`
	Size int `json:"size"`
}

func (p Pagination) GetLimit() int {
	if p.Size < 1 || p.Size > 100 {
		return 100
	}
	return p.Size
}

func (p Pagination) GetOffset() int {
	page := p.Page
	if p.Page < 1 {
		page = 1
	}
	size := p.GetLimit()
	return (page - 1) * size
}

// ======

type SignInfo struct {
	//SignKey  string               `json:"sign_key"`  // sign tx key
	SignList []txbuilder.SignData `json:"sign_list"` // sign list
	//MMJson   *common.MMJsonObj    `json:"mm_json"`   // 712 mmjson
}

type SignInfoList struct {
	Action      common.DasAction     `json:"action"`
	SubAction   common.SubAction     `json:"sub_action"`
	SignKey     string               `json:"sign_key"`
	SignAddress string               `json:"sign_address"`
	List        []SignInfo           `json:"list,omitempty"`
	SignList    []txbuilder.SignData `json:"sign_list,omitempty"` // sign list
	MMJson      *common.MMJsonObj    `json:"mm_json,omitempty"`   // 712 mmjson
}

// =========

type SignInfoCache struct {
	ChainType common.ChainType                   `json:"chain_type"`
	Address   string                             `json:"address"`
	Action    string                             `json:"action"`
	SubAction string                             `json:"sub_action"`
	Account   string                             `json:"account"`
	Capacity  uint64                             `json:"capacity"`
	BuilderTx *txbuilder.DasTxBuilderTransaction `json:"builder_tx"`
}

func (s *SignInfoCache) SignKey() string {
	key := fmt.Sprintf("%d%s%s%d", s.ChainType, s.Address, s.Action, time.Now().UnixNano())
	return fmt.Sprintf("%x", md5.Sum([]byte(key)))
}

// =======

type SignInfoCacheList struct {
	Action        string                               `json:"action"`
	Account       string                               `json:"account"`
	TaskIdList    []string                             `json:"task_id_list"`
	BuilderTxList []*txbuilder.DasTxBuilderTransaction `json:"builder_tx_list"`
}

func (s *SignInfoCacheList) SignKey() string {
	key := fmt.Sprintf("%s%s%d", s.Account, s.Action, time.Now().UnixNano())
	return fmt.Sprintf("%x", md5.Sum([]byte(key)))
}
