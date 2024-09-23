package tables

import (
	"github.com/dotbitHQ/das-lib/common"
	"github.com/shopspring/decimal"
	"time"
)

type TTokenPriceInfo struct {
	Id            uint64          `json:"id" gorm:"column:id; primaryKey; type:bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '';"`
	TokenId       TokenId         `json:"token_id" gorm:"column:token_id; uniqueIndex:uk_token_id; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	CoinType      common.CoinType `json:"coin_type" gorm:"column:coin_type; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	Contract      string          `json:"contact" gorm:"column:contract; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	Name          string          `json:"name" gorm:"column:name; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	Symbol        string          `json:"symbol" gorm:"column:symbol; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	Decimals      int32           `json:"decimals" gorm:"column:decimals; type:smallint(6) NOT NULL DEFAULT '0' COMMENT '';"`
	Price         decimal.Decimal `json:"price" gorm:"column:price; type:decimal(50, 8) NOT NULL DEFAULT '0.00000000' COMMENT '';"`
	Logo          string          `json:"logo" gorm:"column:logo; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	LastUpdatedAt int64           `json:"last_updated_at" gorm:"column:last_updated_at; type:bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '';"`
	Icon          string          `json:"icon" gorm:"column:icon; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	DisplayName   string          `json:"display_name" gorm:"column:display_name; type:varchar(255) NOT NULL DEFAULT '' COMMENT '';"`
	CreatedAt     time.Time       `json:"created_at" gorm:"column:created_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '';"`
	UpdatedAt     time.Time       `json:"updated_at" gorm:"column:updated_at; type:timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '';"`
}

func (t *TTokenPriceInfo) TableName() string {
	return "t_token_price_info"
}

func (t *TTokenPriceInfo) FormatTokenId() TokenId {
	if t.TokenId == TokenIdCkb {
		return TokenIdCkbDas
	}
	return t.TokenId
}

type TokenId string

const (
	TokenIdCkb       TokenId = "ckb_ckb"
	TokenIdCkbDas    TokenId = "ckb_das"
	TokenIdEth       TokenId = "eth_eth"
	TokenIdErc20USDT TokenId = "eth_erc20_usdt"
	TokenIdTrx       TokenId = "tron_trx"
	TokenIdTrc20USDT TokenId = "tron_trc20_usdt"
	TokenIdBnb       TokenId = "bsc_bnb"
	TokenIdBep20USDT TokenId = "bsc_bep20_usdt"
	//TokenIdMatic     TokenId = "polygon_matic"
	TokenIdPOL       TokenId = "polygon_pol"
	TokenIdDoge      TokenId = "doge_doge"
	TokenIdStripeUSD TokenId = "stripe_usd"
	TokenIdDp        TokenId = "did_point"
)
