package handle

import (
	"das_sub_account/config"
	"errors"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
)

type ReqSignInInfo struct {
	core.ChainTypeAddress
}

func (h *HttpHandle) SignInInfo(ctx *gin.Context) {
	var (
		funcName               = "SignInInfo"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqSignInInfo
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

	if err = h.doSignInInfo(ctx, &req, &apiResp); err != nil {
		log.Error("doSignInInfo err:", err.Error(), funcName, clientIp, ctx)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSignInInfo(ctx *gin.Context, req *ReqSignInInfo, apiResp *api_code.ApiResp) error {
	tokenVal, err := ctx.Cookie("token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			apiResp.ApiRespErr(api_code.ApiCodeUnauthorized, "unauthorized")
			return nil
		}
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return nil
	}

	claims := &Claims{}
	tkn, err := jwt.ParseWithClaims(tokenVal, claims, func(token *jwt.Token) (any, error) {
		return []byte(config.Cfg.Das.JwtKey), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			apiResp.ApiRespErr(api_code.ApiCodeUnauthorized, "unauthorized")
			return nil
		}
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return nil
	}
	if !tkn.Valid {
		apiResp.ApiRespErr(api_code.ApiCodeUnauthorized, "unauthorized")
		return nil
	}

	addrHex, err := req.FormatChainTypeAddress(h.DasCore.NetType(), false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return nil
	}
	address := common.FormatAddressPayload(addrHex.AddressPayload, addrHex.DasAlgorithmId)

	accId := common.Bytes2Hex(common.GetAccountIdByAccount(claims.Account))
	accInfo, err := h.DbDao.GetAccountInfoByAccountId(accId)
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
	if !strings.EqualFold(address, accInfo.Owner) && !strings.EqualFold(address, accInfo.Manager) {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
		return nil
	}
	return nil
}
