package example

import (
	"context"
	"das_sub_account/tables"
	"das_sub_account/txtool"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/molecule"
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
		Action:                                common.DasActionCreateSubAccount,
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

func TestPrice2(t *testing.T) {
	usd := uint64(105000)
	priceCkb := (usd / 3780) * common.OneCkb
	fmt.Println(priceCkb)
	dasCkb := (priceCkb / common.PercentRateBase) * uint64(300)
	fmt.Println(priceCkb, dasCkb)
}
