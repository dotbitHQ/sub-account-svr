package example

import (
	"das_sub_account/http_server/handle"
	"fmt"
	"github.com/dotbitHQ/das-lib/core"
	"testing"
)

const (
	ApiUrlInternal = "http://127.0.0.1:8126/v1"
)

func TestInternalSubAccountCreate(t *testing.T) {
	url := ApiUrlInternal + "/internal/sub/account/create"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0x15a33588908cf8edb27d1abe3852bf287abd3891",
			},
		},
		Account: "aaaazzxxx.bit",
		SubAccountList: []handle.CreateSubAccount{
			{
				Account:       "100001.aaaazzxxx.bit",
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
			{
				Account:       "100002.aaaazzxxx.bit",
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

	var data handle.RespInternalSubAccountCreate
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestInternalSubAccountCreate2(t *testing.T) {
	url := ApiUrlInternal + "/internal/sub/account/create"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0x15a33588908cf8edb27d1abe3852bf287abd3891",
			},
		},
		Account: "tzh20220718.bit",
		SubAccountList: []handle.CreateSubAccount{
			{
				Account:       "100001.aaaazzxxx.bit",
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
			{
				Account:       "100002.aaaazzxxx.bit",
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

	req.SubAccountList = make([]handle.CreateSubAccount, 0)
	for i := 0; i < 98; i++ {
		req.SubAccountList = append(req.SubAccountList, handle.CreateSubAccount{
			Account:       fmt.Sprintf("3000%d.aaaazzxxx.bit", i),
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

	var data handle.RespInternalSubAccountCreate
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestTestInternalSubAccountCreate3(t *testing.T) {
	doCreate := func(account string) {
		url := ApiUrlInternal + "/internal/sub/account/create"
		req := handle.ReqSubAccountCreate{
			ChainTypeAddress: core.ChainTypeAddress{
				Type: "blockchain",
				KeyInfo: core.KeyInfo{
					CoinType: "60",
					ChainId:  "5",
					Key:      "0x15a33588908cf8edb27d1abe3852bf287abd3891",
				},
			},
			Account:        account,
			SubAccountList: nil,
		}
		req.SubAccountList = make([]handle.CreateSubAccount, 0)
		for i := 0; i < 3; i++ {
			req.SubAccountList = append(req.SubAccountList, handle.CreateSubAccount{
				Account:       fmt.Sprintf("4001%d.%s", i, account),
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
		var data handle.RespInternalSubAccountCreate
		if err := doReq(url, req, &data); err != nil {
			t.Fatal(err)
		}
	}

	doCreate("1234567881.bit")
	doCreate("1234567882.bit")
}

func TestInternalSubAccountMint(t *testing.T) {
	url := ApiUrlInternal + "/internal/sub/account/mint"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account: "ぁぁ123ぁぁ.bit",
		SubAccountList: []handle.CreateSubAccount{
			{
				Account:       "ぁtest1.ぁぁ123ぁぁ.bit",
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
				Account:       "ぁtest2.ぁぁ123ぁぁ.bit",
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

	var data handle.RespInternalSubAccountMint
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}

func TestMulInternalSubAccountMint(t *testing.T) {
	url := ApiUrlInternal + "/internal/sub/account/mint"
	req := handle.ReqSubAccountCreate{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account:        "tzh20220718.bit",
		SubAccountList: nil,
	}
	for i := 0; i < 98; i++ {
		req.SubAccountList = append(req.SubAccountList, handle.CreateSubAccount{
			Account:       fmt.Sprintf("test03-%d.tzh20220718.bit", i),
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

	var data handle.RespInternalSubAccountMint
	if err := doReq(url, req, &data); err != nil {
		t.Fatal(err)
	}
}
