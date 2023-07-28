package example

import (
	"das_sub_account/http_server/handle"
	"das_sub_account/tables"
	"fmt"
	"github.com/shopspring/decimal"
	"testing"
)

func TestAmount(t *testing.T) {
	tokenID, price, decimals := tables.TokenIdErc20USDT, 1.0, int32(6)
	usdAmount := decimal.NewFromFloat(5.52)
	decPrice := decimal.NewFromFloat(price)
	dec := decimal.New(1, decimals)
	amount := usdAmount.Mul(dec).Div(decPrice).Ceil()
	fmt.Println(amount)
	amount = handle.RoundAmount(amount, tokenID)
	fmt.Println(amount)
}
