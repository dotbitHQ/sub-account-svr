package tables

import (
	"github.com/shopspring/decimal"
	"time"
)

type TTokenPriceInfo struct {
	Id            uint64          `gorm:"column:id;AUTO_INCREMENT;comment:自增主键" json:"id"`
	TokenId       TokenId         `gorm:"column:token_id;type:varchar(255);NOT NULL" json:"token_id"`
	ChainType     int             `gorm:"column:chain_type;type:smallint(6);default:0;NOT NULL" json:"chain_type"`
	Contract      string          `gorm:"column:contract;type:varchar(255);NOT NULL" json:"contract"`
	Symbol        string          `gorm:"column:symbol;type:varchar(255);comment:the symbol of token;NOT NULL" json:"symbol"`
	Decimals      int32           `gorm:"column:decimals;type:smallint(6);default:0;NOT NULL" json:"decimals"`
	Price         decimal.Decimal `gorm:"column:price;type:decimal(50,8);default:0.00000000;NOT NULL" json:"price"`
	Logo          string          `gorm:"column:logo;type:varchar(255);NOT NULL" json:"logo"`
	LastUpdatedAt uint64          `gorm:"column:last_updated_at;type:bigint(20) unsigned;default:0;NOT NULL" json:"last_updated_at"`
	CreatedAt     time.Time       `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP;NOT NULL" json:"updated_at"`
}

func (m *TTokenPriceInfo) TableName() string {
	return "t_token_price_info"
}

type TokenId string

const (
	TokenIdCkb       TokenId = "ckb_ckb"
	TokenIdEth       TokenId = "eth_eth"
	TokenIdErc20USDT TokenId = "eth_erc20_usdt"
	TokenIdTrx       TokenId = "tron_trx"
	TokenIdBnb       TokenId = "bsc_bnb"
	TokenIdBep20USDT TokenId = "bsc_bep20_usdt"
	TokenIdMatic     TokenId = "polygon_matic"
	TokenIdDoge      TokenId = "doge_doge"
)
