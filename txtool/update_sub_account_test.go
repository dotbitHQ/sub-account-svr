package txtool

import (
	"github.com/dotbitHQ/das-lib/common"
	"github.com/influxdata/influxdb/pkg/testing/assert"
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

func TestCreateCkbAddress(t *testing.T) {
	res, err := address.GenerateShortAddress(address.Testnet)
	assert.NoError(t, err)
	t.Log(res.Address)
	t.Log(res.LockArgs)
	t.Log(res.PrivateKey)
}
