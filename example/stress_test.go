package example

import (
	"das_sub_account/http_server/handle"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"sync"
	"testing"
	"time"
)

func TestStressTest(t *testing.T) {
	privateKey := ""
	//account := "2022111001.bit"
	//if err := doSubAccountCreate(account, privateKey, 40); err != nil {
	//	t.Fatal(err)
	//}

	//account = "1668064233-38.2022111001.bit"
	//if err := doSubAccountEdit(account, privateKey); err != nil {
	//	t.Fatal(err)
	//}

	var wg sync.WaitGroup

	doSubAccountEditConcurrency(&wg, privateKey)

	wg.Wait()

	//var wg sync.WaitGroup
	//doSubAccountCreateCycle(&wg, account, privateKey)
	//doSubAccountEditConcurrency(&wg, privateKey)
	//wg.Wait()

}

func doSubAccountCreate(account, privateKey string, count int) error {
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
		Account:        account,
		SubAccountList: nil,
	}
	req.SubAccountList = make([]handle.CreateSubAccount, 0)
	pre := time.Now().Unix()
	for i := 0; i < count; i++ {
		req.SubAccountList = append(req.SubAccountList, handle.CreateSubAccount{
			Account:       fmt.Sprintf("%d-%d.%s", pre, i, account),
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
		return fmt.Errorf("doReq err: %s", err.Error())
	}

	if err := doSign(data.SignInfoList, privateKey); err != nil {
		return fmt.Errorf("doSign err: %s", err.Error())
	}

	if err := doTransactionSend(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		return fmt.Errorf("doTransactionSend err: %s", err.Error())
	}
	return nil
}

func doSubAccountCreateCycle(wg *sync.WaitGroup, account, privateKey string) {
	count := 40
	reqCount := 0
	errCount := 0
	okCount := 3

	ticker := time.NewTicker(time.Second * 30)
	wg.Add(1)
	go func() {
		for {
			select {
			case <-ticker.C:
				reqCount++
				if err := doSubAccountCreate(account, privateKey, count); err != nil {
					fmt.Println("doSubAccountCreate err:", err.Error())
					errCount++
				} else {
					okCount--
				}
			}
			fmt.Println("doSubAccountCreateCycle:", reqCount, errCount, okCount)
			if okCount == 0 {
				wg.Done()
				break
			}
		}
	}()
}

func doSubAccountEdit(account, privateKey string) error {
	now := time.Now().Unix()
	url := ApiUrl + "/sub/account/edit"
	req := handle.ReqSubAccountEdit{
		ChainTypeAddress: core.ChainTypeAddress{
			Type: "blockchain",
			KeyInfo: core.KeyInfo{
				CoinType: "60",
				ChainId:  "5",
				Key:      "0xc9f53b1d85356B60453F867610888D89a0B667Ad",
			},
		},
		Account: account,
		EditKey: common.EditKeyRecords,
		EditValue: handle.EditInfo{
			Records: []handle.EditRecord{
				{
					Key:   "twitter",
					Type:  "profile",
					Label: "",
					Value: fmt.Sprintf("%d", now),
					TTL:   "",
				},
			},
		},
	}

	var data handle.RespSubAccountEdit

	if err := doReq(url, req, &data); err != nil {
		return fmt.Errorf("doReq err: %s", err.Error())
	}

	if err := doSign(data.SignInfoList, privateKey); err != nil {
		return fmt.Errorf("doSign err: %s", err.Error())
	}

	if err := doTransactionSend(handle.ReqTransactionSend{
		SignInfoList: data.SignInfoList,
	}); err != nil {
		return fmt.Errorf("doTransactionSend err: %s", err.Error())
	}
	return nil
}

func doSubAccountEditConcurrency(wg *sync.WaitGroup, privateKey string) {
	var chanEditAccount = make(chan string, 10)
	go func() {
		wg.Add(1)
		defer wg.Done()

		for _, v := range editList {
			chanEditAccount <- v
		}
		close(chanEditAccount)
	}()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for {
				select {
				case acc, ok := <-chanEditAccount:
					if !ok {
						wg.Done()
						return
					}
					time.Sleep(time.Second * 1) // 3~5 mint
					if err := doSubAccountEdit(acc, privateKey); err != nil {
						fmt.Println("doSubAccountEdit err: ", err.Error())
					}
				}
			}
		}()
	}
}

var editList = []string{
	//"1668063813-29.2022111001.bit",

	"1668063813-23.2022111001.bit",
	"1668063813-15.2022111001.bit",
	"1668063813-11.2022111001.bit",
	"1668063813-6.2022111001.bit",
	//"1668063813-22.2022111001.bit",
	//"1668063813-16.2022111001.bit",
	//"1668063813-3.2022111001.bit",
	//"1668063813-25.2022111001.bit",
	//"1668063813-14.2022111001.bit",

	//"1668063813-7.2022111001.bit",
}
