package handle

import (
	"das_sub_account/config"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
	"time"
)

type ReqSignIn struct {
	core.ChainTypeAddress
	Account     string `json:"account" binding:"required"`
	Timestamp   int64  `json:"timestamp" binding:"required"`
	Signature   string `json:"signature" binding:"required"`
	SignAddress string `json:"sign_address"`
}

type RespSignIn struct {
}

type Claims struct {
	Account   string                   `json:"account"`
	Address   string                   `json:"address"`
	Aid       common.DasAlgorithmId    `json:"aid"`
	SubAid    common.DasSubAlgorithmId `json:"sub_aid"`
	Timestamp int64                    `json:"timestamp"`
	Signature string                   `json:"signature"`
	jwt.RegisteredClaims
}

func (h *HttpHandle) SignIn(ctx *gin.Context) {
	var (
		funcName               = "SignIn"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqSignIn
		apiResp                api_code.ApiResp
		err                    error
	)
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, ctx)

	if err = ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ctx.ShouldBindJSON err:", err.Error(), funcName, clientIp, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}

	if err = h.doSignIn(ctx, &req, &apiResp); err != nil {
		log.Error("doSignIn err:", err.Error(), funcName, clientIp, ctx)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSignIn(ctx *gin.Context, req *ReqSignIn, apiResp *api_code.ApiResp) error {
	now := time.Now()
	timestamp := time.UnixMilli(req.Timestamp)
	if now.After(timestamp.Add(time.Minute * 5)) {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "timestamp expired, valid for 5 minutes")
		return nil
	}

	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil
	}

	signAddress := res.AddressHex
	if res.DasAlgorithmId == common.DasAlgorithmIdWebauthn {
		if req.SignAddress == "" {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "sign_address can't be empty")
			return nil
		}
		signAddressHex, err := h.DasCore.Daf().NormalToHex(core.DasAddressNormal{
			ChainType:     common.ChainTypeWebauthn,
			AddressNormal: req.SignAddress,
		})
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return err
		}
		signAddress = signAddressHex.AddressHex

		idx, err := h.DasCore.GetIdxOfKeylist(*res, signAddressHex)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return err
		}
		if idx == -1 {
			err = fmt.Errorf("permission denied")
			apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, err.Error())
			return err
		}
		h.DasCore.AddPkIndexForSignMsg(&req.Signature, idx)
	}

	signMsg := fmt.Sprintf("Account: %s Timestamp: %d", req.Account, req.Timestamp)
	if ok, _, err := api_code.VerifySignature(res.DasAlgorithmId, signMsg, req.Signature, signAddress); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
		return nil
	} else if !ok {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "signature invalid")
		return nil
	}

	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	accInfo, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query parent account")
		return nil
	}
	if accInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account does not exist")
		return nil
	}
	if accInfo.IsExpired() {
		apiResp.ApiRespErr(api_code.ApiCodeParentAccountExpired, "account expired")
		return nil
	}
	if !strings.EqualFold(accInfo.Manager, res.AddressHex) && !strings.EqualFold(accInfo.Owner, res.AddressHex) {
		apiResp.ApiRespErr(api_code.ApiCodeNoAccountPermissions, "no account permissions")
		return nil
	}

	claims := &Claims{
		Account:   req.Account,
		Address:   res.AddressHex,
		Aid:       res.DasAlgorithmId,
		SubAid:    res.DasSubAlgorithmId,
		Timestamp: req.Timestamp,
		Signature: req.Signature,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(timestamp.Add(time.Hour * 24)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.Cfg.Das.JwtKey))
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return err
	}

	if h.DasCore.NetType() == common.DasNetTypeMainNet {
		ctx.SetCookie("token", tokenString, int(claims.ExpiresAt.Sub(now).Seconds()), "/", "topdid.com", true, true)
	} else {
		ctx.SetCookie("token", tokenString, int(claims.ExpiresAt.Sub(now).Seconds()), "/", "", false, false)
	}
	resp := &RespSignIn{}
	apiResp.ApiRespOK(resp)
	return nil
}
