package handle

import (
	"das_sub_account/http_server/api_code"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/gin-gonic/gin"
	"net/http"
)

type RespConfigInfo struct {
	SubAccountBasicCapacity        uint64 `json:"sub_account_basic_capacity"`
	SubAccountPreparedFeeCapacity  uint64 `json:"sub_account_prepared_fee_capacity"`
	SubAccountNewSubAccountPrice   uint64 `json:"sub_account_new_sub_account_price"`
	SubAccountRenewSubAccountPrice uint64 `json:"sub_account_renew_sub_account_price"`
	SubAccountCommonFee            uint64 `json:"sub_account_common_fee"`
}

func (h *HttpHandle) ConfigInfo(ctx *gin.Context) {
	var (
		funcName = "ConfigInfo"
		clientIp = GetClientIp(ctx)
		apiResp  api_code.ApiResp
		err      error
	)
	log.Info("ApiReq:", funcName, clientIp)

	if err = h.doConfigInfo(&apiResp); err != nil {
		log.Error("doConfigInfo err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doConfigInfo(apiResp *api_code.ApiResp) error {
	var resp RespConfigInfo

	builder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}
	resp.SubAccountBasicCapacity, _ = molecule.Bytes2GoU64(builder.ConfigCellSubAccount.BasicCapacity().RawData())
	resp.SubAccountPreparedFeeCapacity, _ = molecule.Bytes2GoU64(builder.ConfigCellSubAccount.PreparedFeeCapacity().RawData())
	resp.SubAccountNewSubAccountPrice, _ = molecule.Bytes2GoU64(builder.ConfigCellSubAccount.NewSubAccountPrice().RawData())
	resp.SubAccountRenewSubAccountPrice, _ = molecule.Bytes2GoU64(builder.ConfigCellSubAccount.RenewSubAccountPrice().RawData())
	resp.SubAccountCommonFee, _ = molecule.Bytes2GoU64(builder.ConfigCellSubAccount.CommonFee().RawData())

	apiResp.ApiRespOK(resp)
	return nil
}
