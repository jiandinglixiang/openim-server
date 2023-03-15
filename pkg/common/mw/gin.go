package mw

import (
	"OpenIM/internal/apiresp"
	"OpenIM/pkg/common/config"
	"OpenIM/pkg/common/constant"
	"OpenIM/pkg/common/db/cache"
	"OpenIM/pkg/common/db/controller"
	"OpenIM/pkg/common/tokenverify"
	"OpenIM/pkg/errs"
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"io"
	"net/http"
)

func CorsHandler() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		context.Header("Access-Control-Allow-Methods", "*")
		context.Header("Access-Control-Allow-Headers", "*")
		context.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar") // 跨域关键设置 让浏览器可以解析
		context.Header("Access-Control-Max-Age", "172800")                                                                                                                                                           // 缓存请求信息 单位为秒
		context.Header("Access-Control-Allow-Credentials", "false")                                                                                                                                                  //  跨域请求是否需要带cookie信息 默认设置为true
		context.Header("content-type", "application/json")                                                                                                                                                           // 设置返回格式是json
		//Release all option pre-requests
		if context.Request.Method == http.MethodOptions {
			context.JSON(http.StatusOK, "Options Request!")
		}
		context.Next()
	}
}

func GinParseOperationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodPost {
			operationID := c.Request.Header.Get(constant.OperationID)
			if operationID == "" {
				body, err := io.ReadAll(c.Request.Body)
				if err != nil {
					c.String(400, "read request body error: "+err.Error())
					c.Abort()
					return
				}
				req := struct {
					OperationID string `json:"operationID"`
				}{}
				if err := json.Unmarshal(body, &req); err != nil {
					c.String(400, "get operationID error: "+err.Error())
					c.Abort()
					return
				}
				if req.OperationID == "" {
					c.String(400, "operationID empty")
					c.Abort()
					return
				}
				c.Request.Body = io.NopCloser(bytes.NewReader(body))
				operationID = req.OperationID
				c.Request.Header.Set(constant.OperationID, operationID)
			}
			c.Set(constant.OperationID, operationID)
			c.Next()
			return
		}
		c.Next()
	}
}
func GinParseToken(rdb redis.UniversalClient) gin.HandlerFunc {
	dataBase := controller.NewAuthDatabase(cache.NewCacheModel(rdb), config.Config.TokenPolicy.AccessSecret, config.Config.TokenPolicy.AccessExpire)
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost:
			token := c.Request.Header.Get(constant.Token)
			if token == "" {
				apiresp.GinError(c, errs.ErrArgs.Wrap())
				c.Abort()
				return
			}
			claims, err := tokenverify.GetClaimFromToken(token)
			if err != nil {
				apiresp.GinError(c, errs.ErrTokenUnknown.Wrap())
				c.Abort()
				return
			}
			m, err := dataBase.GetTokensWithoutError(c, claims.UID, claims.Platform)
			if err != nil {
				apiresp.GinError(c, errs.ErrTokenNotExist.Wrap())
				c.Abort()
				return
			}
			if len(m) == 0 {
				apiresp.GinError(c, errs.ErrTokenNotExist.Wrap())
				c.Abort()
				return
			}
			if v, ok := m[token]; ok {
				switch v {
				case constant.NormalToken:
				case constant.KickedToken:
					apiresp.GinError(c, errs.ErrTokenKicked.Wrap())
					c.Abort()
					return
				default:
					apiresp.GinError(c, errs.ErrTokenUnknown.Wrap())
					c.Abort()
					return
				}
			}
			c.Set(constant.OpUserIDPlatformID, constant.PlatformNameToID(claims.Platform))
			c.Set(constant.OpUserID, claims.UID)
			c.Next()
		}

	}
}
