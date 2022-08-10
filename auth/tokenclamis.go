package token

import (
	"github.com/gin-gonic/gin"
)

/**
 * @Author  糊涂的老知青
 * @Date    2022/7/18
 * @Version 1.0.0
 */

func GetClaims(c *gin.Context) *TokenClaims {
	if claims, exists := c.Get("GlobalTokenClaims"); !exists {
		// panic("登录失效，请重新登录")
		return &TokenClaims{}
		//return TokenClaims{}, errors.New("登录失效，请重新登录")
	} else {
		waitUse := claims.(*TokenClaims)
		return waitUse
	}
}

func GetCurrentUserId(c *gin.Context) string {
	claims := GetClaims(c)
	return claims.Id

}

func GetCurrentUserName(c *gin.Context) string {
	claims := GetClaims(c)
	return claims.Name

}

func GetCurrentLoginName(c *gin.Context) string {
	claims := GetClaims(c)
	return claims.LoginName

}

func GetCurrentTenantId(c *gin.Context) string {
	claims := GetClaims(c)
	return claims.TenantId

}
