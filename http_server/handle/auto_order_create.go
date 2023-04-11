package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"das_sub_account/unipay"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
	"time"
)

type ReqAutoOrderCreate struct {
	core.ChainTypeAddress
	ActionType tables.ActionType `json:"action_type"`
	SubAccount string            `json:"sub_account"`
	TokenId    string            `json:"token_id"`
	Years      uint64            `json:"years"`
}

type RespAutoOrderCreate struct {
	OrderId        string          `json:"order_id"`
	PaymentAddress string          `json:"payment_address"`
	Amount         decimal.Decimal `json:"amount"`
}

func (h *HttpHandle) AutoOrderCreate(ctx *gin.Context) {
	var (
		funcName = "AutoOrderCreate"
		clientIp = GetClientIp(ctx)
		req      ReqAutoOrderCreate
		apiResp  api_code.ApiResp
		err      error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req))

	if err = h.doAutoOrderCreate(&req, &apiResp); err != nil {
		log.Error("doAutoOrderCreate err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doAutoOrderCreate(req *ReqAutoOrderCreate, apiResp *api_code.ApiResp) error {
	var resp RespAutoOrderCreate
	// check key info
	hexAddr, err := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("key-info[%s-%s] invalid", req.KeyInfo.CoinType, req.KeyInfo.Key))
		return nil
	}
	// check sub account name
	parentAccountId := h.checkSubAccountName(apiResp, req.SubAccount)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check parent account
	parentAccount, err := h.checkParentAccount(apiResp, parentAccountId)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check sub account
	subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.SubAccount))
	accStatus, _, err := h.checkSubAccount(apiResp, hexAddr, subAccountId)
	if err != nil {
		return err
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	} else if accStatus == AccStatusMinting {
		apiResp.ApiRespErr(api_code.ApiCodeSubAccountMinting, fmt.Sprintf("sub-account[%s] is minting", req.SubAccount))
		return nil
	} else if accStatus == AccStatusMinted {
		apiResp.ApiRespErr(api_code.ApiCodeSubAccountMinted, fmt.Sprintf("sub-account[%s] has been minted", req.SubAccount))
		return nil
	}

	// get max years
	if maxYear := h.getMaxYears(parentAccount); req.Years > maxYear {
		apiResp.ApiRespErr(api_code.ApiCodeBeyondMaxYears, fmt.Sprintf("sub-account[%s] has been minted", req.SubAccount))
		return nil
	}

	// todo check price: blacklist or price rule
	// todo get price
	amount := decimal.Zero

	// todo create order
	config.Cfg.Server.UniPayUrl = "http://127.0.0.1:9090"
	res, err := unipay.CreateOrder(unipay.ReqOrderCreate{
		ChainTypeAddress: req.ChainTypeAddress,
		BusinessId:       unipay.BusinessIdAutoSubAccount,
		Amount:           amount,
		PayTokenId:       req.TokenId,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to create order")
		return fmt.Errorf("unipay.CreateOrder err: %s", err.Error())
	}
	log.Info("doAutoOrderCreate:", res.OrderId, res.PaymentAddress, amount)
	orderInfo := tables.OrderInfo{
		OrderId:     res.OrderId,
		ActionType:  req.ActionType,
		Account:     req.SubAccount,
		AccountId:   subAccountId,
		Years:       req.Years,
		AlgorithmId: hexAddr.DasAlgorithmId,
		PayAddress:  hexAddr.AddressHex,
		TokenId:     req.TokenId,
		Amount:      decimal.Decimal{}, // todo
		USDAmount:   decimal.Decimal{}, // todo
		PayStatus:   tables.PayStatusUnpaid,
		OrderStatus: tables.OrderStatusDefault,
		Timestamp:   time.Now().Unix(),
	}
	if err = h.DbDao.CreateOrderInfo(orderInfo); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to create order")
		return fmt.Errorf("CreateOrderInfo err: %s", err.Error())
	}

	resp.OrderId = res.OrderId
	resp.Amount = amount
	resp.PaymentAddress = res.PaymentAddress

	apiResp.ApiRespOK(resp)
	return nil
}
