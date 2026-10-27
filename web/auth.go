package web

import (
	"fmt"
	"github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"time"
	"trojan/core"
	"trojan/util"
	"trojan/web/controller"
)

var (
	identityKey    = "id"
	authMiddleware *jwt.GinJWTMiddleware
	err            error
)

// Login auth用户验证结构体
type Login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

func getSecretKey() string {
	sk, _ := core.GetValue("secretKey")
	if sk == "" {
		sk = util.RandString(15, util.ALL)
		core.SetValue("secretKey", sk)
	}
	return sk
}

func jwtInit(timeout int) {
	authMiddleware, err = jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "trojan-manager",
		Key:         []byte(getSecretKey()),
		Timeout:     time.Minute * time.Duration(timeout),
		MaxRefresh:  time.Minute * time.Duration(timeout),
		IdentityKey: identityKey,
		SendCookie:  true,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*Login); ok {
				return jwt.MapClaims{
					identityKey: v.Username,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &Login{
				Username: claims[identityKey].(string),
			}
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var (
				password  string
				loginVals Login
			)
			if err := c.ShouldBind(&loginVals); err != nil {
				return "", jwt.ErrMissingLoginValues
			}
			userID := loginVals.Username
			pass := loginVals.Password
			if err != nil {
				return nil, err
			}
			if userID != "admin" {
				mysql := core.GetMysql()
				user := mysql.GetUserByName(userID)
				if user == nil {
					return nil, jwt.ErrFailedAuthentication
				}
				password = user.EncryptPass
			} else {
				if password, err = core.GetValue(userID + "_pass"); err != nil {
					return nil, err
				}
			}
			if password == pass {
				return &loginVals, nil
			}
			return nil, jwt.ErrFailedAuthentication
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			if _, ok := data.(*Login); ok {
				return true
			}
			return false
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
			})
		},
		TokenLookup:   "header: Authorization, query: token, cookie: jwt",
		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
	})

	if err != nil {
		fmt.Println("JWT Error:" + err.Error())
	}
}

// registerAdmin 仅在初次部署、管理员密码为空时允许设置密码
func registerAdmin(c *gin.Context) {
	responseBody := controller.ResponseBody{Msg: "success"}
	defer controller.TimeCost(time.Now(), &responseBody)

	// 严格检查：若系统已有管理员密码，绝对禁止未授权重复注册覆盖
	adminPass, _ := core.GetValue("admin_pass")
	if adminPass != "" {
		responseBody.Msg = "系统管理员已存在，禁止重复注册"
		c.JSON(403, responseBody)
		return
	}

	pass := c.PostForm("password")
	if pass == "" {
		responseBody.Msg = "管理员密码不能为空"
		c.JSON(400, responseBody)
		return
	}

	err := core.SetValue("admin_pass", pass)
	if err != nil {
		responseBody.Msg = err.Error()
		c.JSON(500, responseBody)
		return
	}
	c.JSON(200, responseBody)
}

// resetPassword 重置密码（必须登录后才可调用）
func resetPassword(c *gin.Context) {
	responseBody := controller.ResponseBody{Msg: "success"}
	defer controller.TimeCost(time.Now(), &responseBody)

	currentUser := RequestUsername(c)
	pass := c.PostForm("password")
	if pass == "" {
		responseBody.Msg = "密码不能为空"
		c.JSON(400, responseBody)
		return
	}

	targetUser := c.DefaultPostForm("username", currentUser)
	// NOTE: 非管理员只能修改自身密码，防止横向越权
	if currentUser != "admin" {
		targetUser = currentUser
	}

	err := core.SetValue(fmt.Sprintf("%s_pass", targetUser), pass)
	if err != nil {
		responseBody.Msg = err.Error()
	}
	c.JSON(200, responseBody)
}

// RequestUsername 获取请求接口的用户名
func RequestUsername(c *gin.Context) string {
	claims := jwt.ExtractClaims(c)
	return claims[identityKey].(string)
}

// Auth 权限router
func Auth(r *gin.Engine, timeout int) *jwt.GinJWTMiddleware {
	jwtInit(timeout)

	newInstall := gin.H{"code": 201, "message": "No administrator account found inside the database", "data": nil}
	r.NoRoute(authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		claims := jwt.ExtractClaims(c)
		fmt.Printf("NoRoute claims: %#v\n", claims)
		c.JSON(404, gin.H{"code": 404, "message": "Page not found"})
	})
	r.GET("/auth/check", func(c *gin.Context) {
		result, _ := core.GetValue("admin_pass")
		if result == "" {
			c.JSON(201, newInstall)
		} else {
			title, err := core.GetValue("login_title")
			if err != nil {
				title = "trojan 管理平台"
			}
			c.JSON(200, gin.H{
				"code":    200,
				"message": "success",
				"data": map[string]string{
					"title": title,
				},
			})
		}
	})
	r.POST("/auth/login", authMiddleware.LoginHandler)
	// 仅初次未初始化时允许访问，已初始化自动拒绝
	r.POST("/auth/register", registerAdmin)
	authO := r.Group("/auth")
	authO.Use(authMiddleware.MiddlewareFunc())
	{
		authO.GET("/loginUser", func(c *gin.Context) {
			result, _ := core.GetValue("admin_pass")
			if result == "" {
				c.JSON(201, newInstall)
			} else {
				c.JSON(200, gin.H{
					"code":    200,
					"message": "success",
					"data": map[string]string{
						"username": RequestUsername(c),
					},
				})
			}
		})
		authO.POST("/reset_pass", resetPassword)
		authO.POST("/logout", authMiddleware.LogoutHandler)
		authO.POST("/refresh_token", authMiddleware.RefreshHandler)
	}
	return authMiddleware
}
