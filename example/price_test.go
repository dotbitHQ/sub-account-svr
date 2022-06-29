package example

import (
	"context"
	"das_sub_account/tables"
	"das_sub_account/txtool"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/nervosnetwork/ckb-sdk-go/rpc"
	"sync"
	"testing"
)

func getClientTestnet2() (rpc.Client, error) {
	ckbUrl := "https://testnet.ckb.dev/"
	indexerUrl := "https://testnet.ckb.dev/indexer"
	return rpc.DialWithIndexer(ckbUrl, indexerUrl)
}

func getNewDasCoreTestnet2() (*core.DasCore, error) {
	client, err := getClientTestnet2()
	if err != nil {
		return nil, err
	}

	env := core.InitEnvOpt(common.DasNetTypeTestnet2,
		common.DasContractNameConfigCellType,
	)
	var wg sync.WaitGroup
	ops := []core.DasCoreOption{
		core.WithClient(client),
		core.WithDasContractArgs(env.ContractArgs),
		core.WithDasContractCodeHash(env.ContractCodeHash),
		core.WithDasNetType(common.DasNetTypeTestnet2),
		core.WithTHQCodeHash(env.THQCodeHash),
	}
	dc := core.NewDasCore(context.Background(), &wg, ops...)
	// contract
	dc.InitDasContract(env.MapContract)
	// config cell
	if err = dc.InitDasConfigCell(); err != nil {
		return nil, err
	}
	return dc, nil
}

func TestGetCustomScriptMintTotalCapacity(t *testing.T) {
	priceApi := txtool.PriceApiDefault{}
	var mintList []tables.TableSmtRecordInfo
	mintList = append(mintList, tables.TableSmtRecordInfo{
		Account:       "tzh13.a.bit",
		RegisterYears: 2,
	})
	mintList = append(mintList, tables.TableSmtRecordInfo{
		Account:       "tzh14.a.bit",
		RegisterYears: 3,
	})

	dc, err := getNewDasCoreTestnet2()
	if err != nil {
		t.Fatal(err)
	}
	quoteCell, err := dc.GetQuoteCell()
	if err != nil {
		t.Fatal(err)
	}
	builder, err := dc.ConfigCellDataBuilderByTypeArgs(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		t.Fatal(err)
	}

	rate, err := molecule.Bytes2GoU32(builder.ConfigCellSubAccount.NewSubAccountCustomPriceDasProfitRate().RawData())
	if err != nil {
		t.Fatal(err)
	}

	res, err := txtool.GetCustomScriptMintTotalCapacity(&txtool.ParamCustomScriptMintTotalCapacity{
		PriceApi:                              &priceApi,
		MintList:                              mintList,
		Quote:                                 quoteCell.Quote(),
		NewSubAccountCustomPriceDasProfitRate: rate,
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res.OwnerCapacity, res.DasCapacity, res.OwnerCapacity+res.DasCapacity)
}

func TestConvertSubAccountCellOutputData(t *testing.T) {
	d := witness.ConvertSubAccountCellOutputData(common.Hex2Bytes("0x554a4da165de2ed38d052309876eb7ae58d1da77b41ead1f4a8d8a0e11a40958004e725300000000000000000000000001f15f519ecb226cd763b2bcbcab093e63f89100c07ac0caebc032c788b187ec99"))
	fmt.Println(d.OwnerProfit, d.DasProfit)
	// 224.80000000 70.20000000
	// 0 1400000000

	i := 233.99972488 + 200.00000000 + 200.00000000
	fmt.Println(i)
	o := 514.99972488 + 118.99979068
	fmt.Println(o)
	fmt.Println(i-o, 514.99972488-233.99972488)

}
