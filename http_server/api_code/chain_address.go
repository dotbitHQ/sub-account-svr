package api_code

import (
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
)

//
//type ChainTypeAddress struct {
//	Type    string  `json:"type"` // blockchain
//	KeyInfo KeyInfo `json:"key_info"`
//}
//
//type KeyInfo struct {
//	CoinType CoinType `json:"coin_type"`
//	ChainId  ChainId  `json:"chain_id"`
//	Key      string   `json:"key"`
//}
//
//func (c *ChainTypeAddress) FormatChainTypeAddress(net common.DasNetType) (*core.DasAddressHex, error) {
//	if c.Type != "blockchain" {
//		return nil, fmt.Errorf("not support type[%s]", c.Type)
//	}
//	dasChainType := FormatCoinTypeToDasChainType(c.KeyInfo.CoinType)
//	if dasChainType == -1 {
//		dasChainType = FormatChainIdToDasChainType(net, c.KeyInfo.ChainId)
//	}
//	if dasChainType == -1 {
//		return nil, fmt.Errorf("not support coin type[%s]-chain id[%s]", c.KeyInfo.CoinType, c.KeyInfo.ChainId)
//	}
//
//	daf := core.DasAddressFormat{DasNetType: net}
//	addrHex, err := daf.NormalToHex(core.DasAddressNormal{
//		ChainType:     dasChainType,
//		AddressNormal: c.KeyInfo.Key,
//		Is712:         true,
//	})
//	if err != nil {
//		return nil, fmt.Errorf("address NormalToHex err")
//	}
//
//	return &addrHex, nil
//}

func FormatChainTypeAddress(net common.DasNetType, chainType common.ChainType, key string) core.ChainTypeAddress {
	var coinType common.CoinType
	switch chainType {
	case common.ChainTypeEth:
		coinType = common.CoinTypeEth
	case common.ChainTypeTron:
		coinType = common.CoinTypeTrx
	}

	var chainId common.ChainId
	if net == common.DasNetTypeMainNet {
		switch chainType {
		case common.ChainTypeEth:
			chainId = common.ChainIdEthMainNet
		}
	} else {
		switch chainType {
		case common.ChainTypeEth:
			chainId = common.ChainIdEthTestNet
		}
	}

	return core.ChainTypeAddress{
		Type: "blockchain",
		KeyInfo: core.KeyInfo{
			CoinType: coinType,
			ChainId:  chainId,
			Key:      key,
		},
	}
}
