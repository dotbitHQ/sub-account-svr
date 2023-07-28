package example

import (
	"fmt"
	"github.com/shopspring/decimal"
	"testing"
)

func TestAmount(t *testing.T) {
	//tokenID, price, decimals := tables.TokenIdErc20USDT, 1.0, int32(6)
	//usdAmount := decimal.NewFromFloat(5.52)
	//decPrice := decimal.NewFromFloat(price)
	//dec := decimal.New(1, decimals)
	//amount := usdAmount.Mul(dec).Div(decPrice).Ceil()
	//fmt.Println(amount)
	//amount = handle.RoundAmount(amount, tokenID)
	//fmt.Println(amount)

	amount := decimal.NewFromInt(10)
	premiumPercentage := decimal.NewFromFloat(0.036)
	premiumBase := decimal.NewFromFloat(0.52)
	amount = amount.Mul(premiumPercentage.Add(decimal.NewFromInt(1))).Add(premiumBase.Mul(decimal.NewFromInt(100)))
	fmt.Println(amount)
	amount = decimal.NewFromInt(amount.Ceil().IntPart())
	fmt.Println(amount)
}
