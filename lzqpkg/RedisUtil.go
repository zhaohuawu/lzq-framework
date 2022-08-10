package lzqpkg

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

/**
 * @Author  糊涂的老知青
 * @Date    2022/3/11
 * @Version 1.0.0
 */

type redisUtil struct {
}

var RedisUtil = redisUtil{}

func (r *redisUtil) NewRedis(c *gin.Context, useMultiTenancy bool, cacheNames ...string) *RedisHelper {
	prefixKey := ""
	for i, v := range cacheNames {
		if len(v) > 0 {
			if i == 0 {
				prefixKey = v
			} else {
				prefixKey = fmt.Sprintf("%v:%v", prefixKey, v)
			}
		}
	}
	return &RedisHelper{
		ginCtx:            c,
		isUseMultiTenancy: useMultiTenancy,
		prefixKey:         prefixKey,
	}
}
