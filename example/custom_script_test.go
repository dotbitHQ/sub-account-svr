package example

import (
	"context"
	"das_sub_account/http_server/handle"
	"encoding/binary"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"testing"
)

func TestCustomScript(t *testing.T) {
	privateKey := ""

	args := common.Bytes2Hex(make([]byte, 33))
	args = "0x01f15f519ecb226cd763b2bcbcab093e63f89100c07ac0caebc032c788b187ec99"
	fmt.Println(args)
	url := ApiUrl + "/custom/script/set"
	req := handle.ReqCustomScript{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: common.CoinTypeEth,
				ChainId:  "",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account:          "ぁぁ123ぁぁ.bit",
		CustomScriptArgs: args,
		CustomScriptConfig: map[uint8]witness.CustomScriptPrice{
			5: {10000, 10000},
		},
	}
	var data handle.RespCustomScript
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}

	if err := doSign(data.SignInfoList, privateKey); err != nil {
		t.Fatal(err)
	}

	if err := doTransactionSend(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestCustomScriptInfo(t *testing.T) {
	url := ApiUrl + "/custom/script/info"
	req := handle.ReqCustomScriptInfo{Account: "ぁぁ123ぁぁ.bit"}
	var data handle.RespCustomScriptInfo
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestSubAccountMintPrice(t *testing.T) {
	url := ApiUrl + "/custom/script/price"
	req := handle.ReqCustomScriptPrice{SubAccount: "tzh001.tzh2022070601.bit"}
	var data handle.RespCustomScriptPrice
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestConvertSubAccountCellOutputData(t *testing.T) {
	d := witness.ConvertSubAccountCellOutputData(common.Hex2Bytes("0x9c7d8e41528b34bae45e271e7fa38466c1a4dcc807d30a42093398edc593146d00a3e111000000000000000000000000"))
	fmt.Println(d.CustomScriptArgs, d.OwnerProfit, d.DasProfit, d.SmtRoot)
}

func TestOwnerProfit(t *testing.T) {
	url := ApiUrl + "/owner/profit"
	req := handle.ReqOwnerProfit{ChainTypeAddress: core.ChainTypeAddress{
		Type: "blockchain",
		KeyInfo: core.KeyInfo{
			CoinType: common.CoinTypeEth,
			ChainId:  "",
			Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
		},
	}, Account: "tzh2022070601.bit"}
	var data handle.RespOwnerProfit
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestPrice(t *testing.T) {
	// 0.02 $
	fmt.Println((60000 / 10770) * common.OneCkb)
	fmt.Println(100000 * common.OneCkb / 3720)
	fmt.Println((26600000000 / 10000) * 2000)
	fmt.Println(common.OneCkb)
}

func TestCustomScriptPrice(t *testing.T) {
	dc, err := getNewDasCoreTestnet2()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := dc.Client().GetTransaction(context.Background(), types.HexToHash("0x3b117c4ffe4430cd1a295eff6e8e8bb602be0ee02d633a17f43ec15d93ceff21"))
	if err != nil {
		t.Fatal(err)
	}

	rate := uint64(0)
	for _, v := range tx.Transaction.CellDeps {
		fmt.Println(v.DepType, v.OutPoint.TxHash.String())
		cellDepsTx, err := dc.Client().GetTransaction(context.Background(), v.OutPoint.TxHash)
		if err != nil {
			t.Fatal(err)
		}
		refOutputsData := cellDepsTx.Transaction.OutputsData[v.OutPoint.Index]
		refOutputs := cellDepsTx.Transaction.Outputs[v.OutPoint.Index]
		if refOutputs.Type != nil && common.Bytes2Hex(refOutputs.Type.Args) == common.ArgsQuoteCell {
			fmt.Println(cellDepsTx.Transaction.Hash.String())
			rate = binary.BigEndian.Uint64(refOutputsData[2:])
			fmt.Println(rate)
			break
		}
	}

	subAccBuilder, err := dc.ConfigCellDataBuilderByTypeArgs(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		t.Fatal(err)
	}
	profi, _ := molecule.Bytes2GoU64(subAccBuilder.ConfigCellSubAccount.NewSubAccountCustomPriceDasProfitRate().RawData())

	_, conf, err := witness.ConvertCustomScriptConfigByTx(tx.Transaction)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(conf.Body)

	subDetail := witness.ConvertSubAccountCellOutputData(tx.Transaction.OutputsData[0])
	fmt.Println(subDetail.OwnerProfit)
	subAccountMap, err := witness.SubAccountBuilderMapFromTx(tx.Transaction)
	for _, v := range subAccountMap {
		fmt.Println(v.Account)
		price, _ := conf.GetPriceBySubAccount(v.Account)
		fmt.Println(price)
		registerYears := uint64(1)
		priceCkb := (registerYears * price.New / rate) * common.OneCkb
		dasCkb := (priceCkb / common.PercentRateBase) * uint64(profi)
		ownerCkb := priceCkb - dasCkb
		fmt.Println(v.Account, ownerCkb)
	}

}
