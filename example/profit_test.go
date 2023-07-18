package example

import (
	"das_sub_account/config"
	"das_sub_account/dao"
	"das_sub_account/http_server/handle"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/shopspring/decimal"
	"math"
	"strings"
	"testing"
	"time"
)

func TestProfit(t *testing.T) {
	addrDB := ""
	user := ""
	password := ""
	dbMysql := config.DbMysql{
		Addr:        addrDB,
		User:        user,
		Password:    password,
		DbName:      "sub_account_db",
		MaxOpenConn: 100,
		MaxIdleConn: 100,
	}
	parserMysql := config.DbMysql{
		Addr:        addrDB,
		User:        user,
		Password:    password,
		DbName:      "das_database",
		MaxOpenConn: 100,
		MaxIdleConn: 100,
	}
	dbDao, err := dao.NewDbDao(dbMysql, parserMysql)
	if err != nil {
		t.Fatal(err)
	}

	end := time.Now().UnixMilli()
	account := ""
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(account))
	list, err := dbDao.FindOrderByPayment(end, accountId)
	if err != nil {
		t.Fatal(err)
	}

	tokens, err := dbDao.FindTokens()
	if err != nil {
		t.Fatal(err)
		return
	}

	records := make(map[string]*handle.CsvRecord)
	for _, v := range list {
		token, ok := tokens[v.TokenId]
		if !ok {
			t.Fatalf("token_id: %s no exist", v.TokenId)
			return
		}

		recordKey := v.ParentAccountId + v.TokenId
		csvRecord, ok := records[recordKey]
		if !ok {
			accounts := strings.Split(v.Account, ".")
			acc := accounts[len(accounts)-2] + "." + accounts[len(accounts)-1]
			csvRecord = &handle.CsvRecord{}
			csvRecord.Account = acc
			csvRecord.AccountId = v.ParentAccountId
			csvRecord.TokenId = v.TokenId
			csvRecord.Decimals = token.Decimals
			csvRecord.Ids = make([]uint64, 0)
			records[recordKey] = csvRecord
		}
		csvRecord.Amount = csvRecord.Amount.Add(v.Amount)
		csvRecord.Ids = append(csvRecord.Ids, v.Id)
	}

	recordsNew := make(map[string]*handle.CsvRecord)
	config.Cfg.Das.AutoMint.PaymentMinPrice = 50
	for k, v := range records {
		token, _ := tokens[v.TokenId]
		price := v.Amount.Div(decimal.NewFromInt(int64(math.Pow10(int(v.Decimals))))).Mul(token.Price)
		if price.LessThan(decimal.NewFromInt(config.Cfg.Das.AutoMint.PaymentMinPrice)) {
			fmt.Printf("account: %s, token_id: %s, price: %s$ less than min price: %d$, skip it\n",
				v.Account, v.TokenId, price, config.Cfg.Das.AutoMint.PaymentMinPrice)
			continue
		}

		recordKeys, ok := common.TokenId2RecordKeyMap[v.TokenId]
		if !ok {
			t.Fatalf("token id: [%s] to record key mapping failed", v.TokenId)
			return
		}
		record, err := dbDao.GetRecordsByAccountIdAndTypeAndLabel(v.AccountId, "address", handle.LabelSubDIDApp, recordKeys)
		if err != nil {
			t.Fatal(err)
			return
		}
		if record.Id == 0 {
			fmt.Printf("account: %s, token_id: %s no address set, skip it\n", v.Account, v.TokenId)
			continue
		}
		v.Address = record.Value
		oAmount := v.Amount.DivRound(decimal.NewFromInt(int64(math.Pow10(int(v.Decimals)))), v.Decimals)
		v.Amount = oAmount.Mul(decimal.NewFromFloat(1 - 0.1))
		recordsNew[k] = v
		fmt.Printf("account: %s, token_id: %s, oAmount: %s, amount: %s, price: %s$\n", v.Account, v.TokenId, oAmount, v.Amount, price)
	}

	txtPath := fmt.Sprintf("./%s.csv", account)
	if err := writeToFile(txtPath, "parent_account,payment_address,payment_type,amount\n"); err != nil {
		fmt.Println(err)
	}
	for _, v := range recordsNew {
		msg := fmt.Sprintf("%s,%s,%s,%s\n", v.Account, v.Address, v.TokenId, v.Amount.String())
		if err := writeToFile(txtPath, msg); err != nil {
			fmt.Println(err)
		}
	}
}
