package api_code

import (
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/core"
)

type ChainTypeAddress struct {
	Type    string  `json:"type"` // blockchain
	KeyInfo KeyInfo `json:"key_info"`
}

type KeyInfo struct {
	CoinType CoinType `json:"coin_type"`
	ChainId  ChainId  `json:"chain_id"`
	Key      string   `json:"key"`
}

func (c *ChainTypeAddress) FormatChainTypeAddress(net common.DasNetType) (common.ChainType, string, error) {
	if c.Type != "blockchain" {
		return -1, "", fmt.Errorf("not support type[%s]", c.Type)
	}
	dasChainType := FormatCoinTypeToDasChainType(c.KeyInfo.CoinType)
	if dasChainType == -1 {
		dasChainType = FormatChainIdToDasChainType(net, c.KeyInfo.ChainId)
	}
	//if dasChainType == -1 {
	//	if strings.HasPrefix(c.KeyInfo.Key, "0x") && len(c.KeyInfo.Key) == 42 {
	//		dasChainType = common.ChainTypeEth
	//	} else if strings.HasPrefix(c.KeyInfo.Key, "0x") && len(c.KeyInfo.Key) == 66 {
	//		dasChainType = common.ChainTypeMixin
	//	}
	//}
	if dasChainType == -1 {
		return dasChainType, "", fmt.Errorf("not support coin type[%s]-chain id[%s]", c.KeyInfo.CoinType, c.KeyInfo.ChainId)
	}

	return dasChainType, core.FormatAddressToHex(dasChainType, c.KeyInfo.Key), nil
}

func FormatChainTypeAddress(net common.DasNetType, chainType common.ChainType, address string) ChainTypeAddress {
	var coinType CoinType
	switch chainType {
	case common.ChainTypeEth:
		coinType = CoinTypeEth
	case common.ChainTypeTron:
		coinType = CoinTypeTrx
	}

	var chainId ChainId
	if net == common.DasNetTypeMainNet {
		switch chainType {
		case common.ChainTypeEth:
			chainId = ChainIdEthMainNet
		}
	} else {
		switch chainType {
		case common.ChainTypeEth:
			chainId = ChainIdEthTestNet
		}
	}

	return ChainTypeAddress{
		Type: "blockchain",
		KeyInfo: KeyInfo{
			CoinType: coinType,
			ChainId:  chainId,
			Key:      core.FormatHexAddressToNormal(chainType, address),
		},
	}
}
