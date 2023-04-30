package txtool

import (
	"github.com/dotbitHQ/das-lib/common"
	"github.com/nervosnetwork/ckb-sdk-go/address"
	"testing"
)

func TestAddress(t *testing.T) {
	parseAddress, err := address.Parse("ckt1qzda0cr08m85hc8jlnfp3zer7xulejywt49kt2rr0vthywaa50xwsqgr3ll6alm8s6rm4w9nlq87ptr0l0zgyhq3zvv3s")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(len(parseAddress.Script.Args))
	t.Log(common.Bytes2Hex(parseAddress.Script.Args))
}
