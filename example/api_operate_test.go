package example

import (
	"das_sub_account/http_server/handle"
	"das_sub_account/task"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/sign"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/scorpiotzh/toolib"
	"log"
	"regexp"
	"sync"
	"testing"
	"time"
)

func TestSubAccountInit(t *testing.T) {
	url := ApiUrl + "/sub/account/init"
	req := handle.ReqSubAccountInit{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: common.CoinTypeEth,
				ChainId:  "",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account: "aaatzh0630.bit",
	}

	var data handle.RespSubAccountInit

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}

	if err := doSign(data.SignInfoList, ""); err != nil {
		t.Fatal(err)
	}

	if err := doTransactionSend(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		t.Fatal(err)
	}

}

func TestSubAccountCheck(t *testing.T) {
	url := ApiUrl + "/sub/account/check"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account: "0001.bit",
		SubAccountList: []handle.CreateSubAccount{
			{
				Account:       "bให้😊บaริก😊02.0001.bit",
				RegisterYears: 1,
				ChainTypeAddress: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
					},
				},
			},
			{
				Account:       "00009.0001.bit",
				RegisterYears: 1,
				ChainTypeAddress: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
					},
				},
			},
		},
	}

	var data handle.RespSubAccountCheck

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestSubAccountCreate(t *testing.T) {
	url := ApiUrl + "/sub/account/create"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account: "5ph2lc3zs6x.bit",
		SubAccountList: []handle.CreateSubAccount{
			{
				Account:       "00011.0001.bit",
				RegisterYears: 2,
				ChainTypeAddress: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0x15a33588908cf8edb27d1abe3852bf287abd3891",
					},
				},
			},
			{
				Account:       "00012.0001.bit",
				RegisterYears: 2,
				ChainTypeAddress: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
					},
				},
			},
		},
	}

	req.SubAccountList = make([]handle.CreateSubAccount, 0)
	for i := 0; i < 98; i++ {
		req.SubAccountList = append(req.SubAccountList, handle.CreateSubAccount{
			Account:       fmt.Sprintf("test01-%d.5ph2lc3zs6x.bit", i),
			RegisterYears: 1,
			ChainTypeAddress: core.ChainTypeAddress{
				Type: "blockchain",
				KeyInfo: core.KeyInfo{
					CoinType: "60",
					ChainId:  "5",
					Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
				},
			},
		})
	}

	var data handle.RespSubAccountCreate

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}

	if err := doSign(data.SignInfoList, "bfb23b0d4cbcc78b3849c04b551bcc88910f47338ee223beebbfb72856e25efa"); err != nil {
		t.Fatal(err)
	}

	if err := doTransactionSend(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSubAccountCreate2(t *testing.T) {
	privateKey := ""
	url := ApiUrl + "/sub/account/create"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account: "tzh20220804.bit",
		SubAccountList: []handle.CreateSubAccount{
			{
				Account:       "tesat01😊.tzh20220804.bit",
				RegisterYears: 1,
				ChainTypeAddress: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
					},
				},
			},
			{
				Account:       "ให้😊บaริก😊02.tzh20220804.bit",
				RegisterYears: 1,
				ChainTypeAddress: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0x15a33588908cf8edb27d1abe3852bf287abd3891",
					},
				},
			},
		},
	}

	var data handle.RespSubAccountCreate

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

func TestSubAccountCreate3(t *testing.T) {
	url := ApiUrl + "/sub/account/create"
	privateKey := ""

	doCreate := func(account string) {
		req := handle.ReqSubAccountCreate{
			ChainTypeAddress: core.ChainTypeAddress{
				Type: "blockchain",
				KeyInfo: core.KeyInfo{
					CoinType: "60",
					ChainId:  "5",
					Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
				},
			},
			Account: account, //"aaaazzxxx.bit",
			SubAccountList: []handle.CreateSubAccount{
				{
					Account:       "00011.0001.bit",
					RegisterYears: 2,
					ChainTypeAddress: core.ChainTypeAddress{
						Type: "blockchain",
						KeyInfo: core.KeyInfo{
							CoinType: "60",
							ChainId:  "5",
							Key:      "0x15a33588908cf8edb27d1abe3852bf287abd3891",
						},
					},
				},
				{
					Account:       "00012.0001.bit",
					RegisterYears: 2,
					ChainTypeAddress: core.ChainTypeAddress{
						Type: "blockchain",
						KeyInfo: core.KeyInfo{
							CoinType: "60",
							ChainId:  "5",
							Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
						},
					},
				},
			},
		}

		req.SubAccountList = make([]handle.CreateSubAccount, 0)
		for i := 0; i < 50; i++ {
			req.SubAccountList = append(req.SubAccountList, handle.CreateSubAccount{
				Account:       fmt.Sprintf("0️⃣1️⃣2️⃣3️⃣4️⃣5️⃣6️⃣7️⃣8️⃣9️⃣%d.%s", i, account),
				RegisterYears: 1,
				ChainTypeAddress: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0x15a33588908cf8edb27d1abe3852bf287abd3891",
					},
				},
			})
		}

		var data handle.RespSubAccountCreate

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

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		doCreate("tzh202220928-01.bit")
	}()
	go func() {
		defer wg.Done()
		doCreate("tzh202220928-2.bit")
	}()
	wg.Wait()
}

func TestSubAccountEdit(t *testing.T) {
	url := ApiUrl + "/sub/account/edit"
	req := handle.ReqSubAccountEdit{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0x15a33588908cf8edb27d1abe3852bf287abd3891",
			},
		},
		Account: "00003.aaaazzxxx.bit",
		EditKey: common.EditKeyRecords,
		EditValue: handle.EditInfo{
			Owner: core.ChainTypeAddress{
				Type: "blockchain",
				KeyInfo: core.KeyInfo{
					CoinType: "60",
					ChainId:  "5",
					Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
				},
			},
			Manager: core.ChainTypeAddress{
				Type: "blockchain",
				KeyInfo: core.KeyInfo{
					CoinType: "60",
					ChainId:  "5",
					Key:      "0x15a33588908cf8edb27d1abe3852bf287abd3891",
				},
			},
			Records: []handle.EditRecord{
				{
					Key:   "twitter",
					Type:  "profile",
					Label: "",
					Value: "111",
					TTL:   "",
				},
			},
			OwnerChainType:   0,
			OwnerAddress:     "",
			ManagerChainType: 0,
			ManagerAddress:   "",
		},
	}

	var data handle.RespSubAccountEdit

	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}

	if err := doSign(data.SignInfoList, ""); err != nil {
		t.Fatal(err)
	}

	if err := doTransactionSend(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSubAccountEdit2(t *testing.T) {
	for i := 1; i < 2; i++ {
		url := ApiUrl + "/sub/account/edit"
		req := handle.ReqSubAccountEdit{
			ChainTypeAddress: core.ChainTypeAddress{
				Type: "blockchain",
				KeyInfo: core.KeyInfo{
					CoinType: "60",
					ChainId:  "5",
					Key:      "0x15a33588908cf8edb27d1abe3852bf287abd3891",
				},
			},
			Account: fmt.Sprintf("0001%d.aaaazzxxx.bit", i),
			EditKey: common.EditKeyRecords,
			EditValue: handle.EditInfo{
				Owner: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
					},
				},
				Manager: core.ChainTypeAddress{
					Type: "blockchain",
					KeyInfo: core.KeyInfo{
						CoinType: "60",
						ChainId:  "5",
						Key:      "0x15a33588908cf8edb27d1abe3852bf287abd3891",
					},
				},
				Records: []handle.EditRecord{
					{
						Key:   "twitter",
						Type:  "profile",
						Label: "",
						Value: "111",
						TTL:   "",
					},
				},
				OwnerChainType:   0,
				OwnerAddress:     "",
				ManagerChainType: 0,
				ManagerAddress:   "",
			},
		}
		for j := 0; j < 45; j++ {
			req.EditValue.Records = append(req.EditValue.Records, handle.EditRecord{
				Index: 0,
				Key:   "eth",
				Type:  "address",
				Label: "eth",
				Value: "0x15a33588908cf8edb27d1abe3852bf287abd3891",
				TTL:   "300",
			})
		}

		var data handle.RespSubAccountEdit

		if err := doReq(url, req, &data); err != nil {
			t.Fatal(err)
		}

		if err := doSign(data.SignInfoList, ""); err != nil {
			t.Fatal(err)
		}

		if err := doTransactionSend(handle.ReqTransactionSend{
			SignInfoList: data.SignInfoList,
		}); err != nil {
			t.Fatal(err)
		}
	}

}

func doSign(data handle.SignInfoList, privateKey string) error {
	for i, _ := range data.List {
		if err := task.DoSign(data.Action, data.List[i].SignList, privateKey); err != nil {
			return fmt.Errorf("task.DoSign err: %s", err.Error())
		}
	}
	fmt.Println("=========== doSign:", toolib.JsonString(data.List))
	return nil
}

func doTransactionSend(req handle.ReqTransactionSend) error {
	url := ApiUrl + "/transaction/send"

	var data handle.RespTransactionSend

	if err := doReq(url, req, &data); err != nil {
		return fmt.Errorf("doReq err: %s", err.Error())
	}
	return nil
}

func TestTime(t *testing.T) {
	t1 := "2022-03-21 20:00:00"
	timeTemplate1 := "2006-01-02 15:04:05"
	stamp, _ := time.ParseInLocation(timeTemplate1, t1, time.Local)
	log.Println(stamp.Unix())

	fmt.Println(time.Now().UnixNano()/1e6, time.Now().Add(time.Millisecond*100).UnixNano()/1e6)
}

func TestVerifyEthSignature(t *testing.T) {
	signMsg := common.Hex2Bytes("0x030659a5613fdfa29453196400bf44d553fc883dbe757536ce9846a8e8324d29527bc4932cbdf0b485522331e1ed2b065bc6163712c373e362e34a0483125dce00")
	rawByte := "from did: 0x6d616e6167657205c9f53b1d85356b60453f867610888d89a0b667ad0515a33588908cf8edb27d1abe3852bf287abd38910100000000000000"
	address := "0x15a33588908cf8edb27d1abe3852bf287abd3891"

	fmt.Println(sign.VerifyPersonalSignature(signMsg, []byte(rawByte), address))
}

func TestRecords(t *testing.T) {
	var records []witness.Record

	for i := 0; i < 46; i++ {
		records = append(records, witness.Record{
			Key:   "eth",
			Type:  "address",
			Label: "eth",
			Value: "0x15a33588908cf8edb27d1abe3852bf287abd3891",
			TTL:   300,
		})
	}

	wi := witness.ConvertToCellRecords(records)
	fmt.Println(wi.TotalSize())
}

func TestAddress(t *testing.T) {
	address := "0x15a33588908cf8edb27d1abe3852bf287abd3891"
	fmt.Println(regexp.MatchString("^0x[0-9a-fA-F]{40}$", address))
}
