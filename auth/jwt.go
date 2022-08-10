package token

/**
 * @Author  糊涂的老知青
 * @Date    2021/10/30
 * @Version 1.0.0
 */

import (
	"lzq-framework/config"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type TokenClaims struct {
	LoginName string `json:"loginName"`
	Name      string `json:"name"`
	SysType   string `json:"sysType"`
	TenantId  string `json:"tenantId"`
	jwt.StandardClaims
}

type JwtConfig struct {
	JwtIssuer     string `mapstructure:"JwtIssuer"`
	JwtSecret     string `mapstructure:"JwtSecret"`
	JwtExpireDate int    `mapstructure:"JwtExpireDate"`
}

// var GlobalTokenClaims = &TokenClaims{}

const (
	SysTypeAdmin = "admin"
	SysTypeWeb   = "web"
)

//GenerateToken 签发用户Token
func GenerateToken(userId, loginName, userName, sysType string, tenantId string) (string, error) {
	var jwtConfig JwtConfig
	if err := config.LzqConfig.Sub("jwt").Unmarshal(&jwtConfig); err != nil {
		return "", err
	}
	nowTime := time.Now()
	expireTime := nowTime.Add(time.Duration(jwtConfig.JwtExpireDate*24) * time.Hour)

	claims := TokenClaims{
		loginName,
		userName,
		sysType,
		"",
		jwt.StandardClaims{
			Id:        userId,
			ExpiresAt: expireTime.Unix(),
			Issuer:    jwtConfig.JwtIssuer,
		},
	}
	useMultiTenancy := config.LzqConfig.GetBool("server.UseMultiTenancy")
	if useMultiTenancy {
		claims.TenantId = tenantId
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := tokenClaims.SignedString([]byte(jwtConfig.JwtSecret))

	return accessToken, err
}

// ParseToken 解析Token
func ParseToken(accessToken string) (*TokenClaims, error) {
	tokenClaims, err := jwt.ParseWithClaims(accessToken, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.LzqConfig.GetString("jwt.JwtSecret")), nil
	})
	if tokenClaims != nil {
		if claims, ok := tokenClaims.Claims.(*TokenClaims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}
	return nil, err
}
