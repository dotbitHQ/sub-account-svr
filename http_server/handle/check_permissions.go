package handle

import (
	"das_sub_account/config"
	"errors"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
)

type ReqCheckPermissions struct {
	core.ChainTypeAddress
	Account string `json:"account"`
}

func (h *HttpHandle) CheckPermissions(ctx *gin.Context) {
	var apiResp api_code.ApiResp
	defer func() {
		if apiResp.ErrNo != 0 {
			ctx.JSON(http.StatusOK, apiResp)
			ctx.Abort()
		}
	}()

	tokenVal, err := ctx.Cookie("token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			apiResp.ApiRespErr(api_code.ApiCodeUnauthorized, "unauthorized")
			return
		}
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return
	}

	claims := &Claims{}
	tkn, err := jwt.ParseWithClaims(tokenVal, claims, func(token *jwt.Token) (any, error) {
		return []byte(config.Cfg.Das.JwtKey), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			apiResp.ApiRespErr(api_code.ApiCodeUnauthorized, "unauthorized")
			return
		}
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return
	}
	if !tkn.Valid {
		apiResp.ApiRespErr(api_code.ApiCodeUnauthorized, "unauthorized")
		return
	}

	var req ReqCheckPermissions
	if err := ctx.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return
	}
	addrHex, err := req.FormatChainTypeAddress(h.DasCore.NetType(), false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return
	}
	address := common.FormatAddressPayload(addrHex.AddressPayload, addrHex.DasAlgorithmId)

	if !strings.EqualFold(address, claims.Address) ||
		addrHex.DasAlgorithmId != claims.Aid ||
		addrHex.DasSubAlgorithmId != claims.SubAid {
		apiResp.ApiRespErr(api_code.ApiCodeUnauthorized, "unauthorized")
		return
	}

	accId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	accInfo, err := h.DbDao.GetAccountInfoByAccountId(accId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query parent account")
		return
	}
	if accInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account does not exist")
		return
	}
	if accInfo.IsExpired() {
		apiResp.ApiRespErr(api_code.ApiCodeParentAccountExpired, "account expired")
		return
	}
	if !strings.EqualFold(address, accInfo.Owner) && !strings.EqualFold(address, accInfo.Manager) {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
		return
	}
}
