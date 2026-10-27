package web

import (
	"embed"
	"fmt"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"trojan/core"
	"trojan/trojan"
	"trojan/util"
	"trojan/web/controller"
)

//go:embed templates/*
var f embed.FS

// adminOnlyMiddleware 确保只有管理员账号 admin 才能进行服务配置与全局用户管理
func adminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if RequestUsername(c) != "admin" {
			c.JSON(403, gin.H{"code": 403, "msg": "权限不足，仅管理员可执行该管理操作"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func userRouter(router *gin.Engine) {
	user := router.Group("/trojan/user")
	{
		// 普通用户和管理员均可获取自身信息
		user.GET("", func(c *gin.Context) {
			requestUser := RequestUsername(c)
			c.JSON(200, controller.UserList(requestUser))
		})

		// 以下操作仅管理员允许执行
		adminUser := user.Group("")
		adminUser.Use(adminOnlyMiddleware())
		{
			adminUser.GET("/page", func(c *gin.Context) {
				curPageStr := c.DefaultQuery("curPage", "1")
				pageSizeStr := c.DefaultQuery("pageSize", "10")
				curPage, _ := strconv.Atoi(curPageStr)
				pageSize, _ := strconv.Atoi(pageSizeStr)
				c.JSON(200, controller.PageUserList(curPage, pageSize))
			})
			adminUser.POST("", func(c *gin.Context) {
				username := c.PostForm("username")
				password := c.PostForm("password")
				c.JSON(200, controller.CreateUser(username, password))
			})
			adminUser.POST("/update", func(c *gin.Context) {
				sid := c.PostForm("id")
				username := c.PostForm("username")
				password := c.PostForm("password")
				id, _ := strconv.Atoi(sid)
				c.JSON(200, controller.UpdateUser(uint(id), username, password))
			})
			adminUser.POST("/expire", func(c *gin.Context) {
				sid := c.PostForm("id")
				sDays := c.PostForm("useDays")
				id, _ := strconv.Atoi(sid)
				useDays, _ := strconv.Atoi(sDays)
				c.JSON(200, controller.SetExpire(uint(id), uint(useDays)))
			})
			adminUser.DELETE("/expire", func(c *gin.Context) {
				sid := c.Query("id")
				id, _ := strconv.Atoi(sid)
				c.JSON(200, controller.CancelExpire(uint(id)))
			})
			adminUser.DELETE("", func(c *gin.Context) {
				stringId := c.Query("id")
				id, _ := strconv.Atoi(stringId)
				c.JSON(200, controller.DelUser(uint(id)))
			})
		}
	}
}

func trojanRouter(router *gin.Engine) {
	trojan := router.Group("/trojan")
	trojan.Use(adminOnlyMiddleware())
	{
		trojan.POST("/start", func(c *gin.Context) {
			c.JSON(200, controller.Start())
		})
		trojan.POST("/stop", func(c *gin.Context) {
			c.JSON(200, controller.Stop())
		})
		trojan.POST("/restart", func(c *gin.Context) {
			c.JSON(200, controller.Restart())
		})
		trojan.GET("/loglevel", func(c *gin.Context) {
			c.JSON(200, controller.GetLogLevel())
		})
		trojan.GET("/export", func(c *gin.Context) {
			result := controller.ExportCsv(c)
			if result != nil {
				c.JSON(200, result)
			}
		})
		trojan.POST("/import", func(c *gin.Context) {
			c.JSON(200, controller.ImportCsv(c))
		})
		trojan.POST("/update", func(c *gin.Context) {
			c.JSON(200, controller.Update())
		})
		trojan.POST("/switch", func(c *gin.Context) {
			tType := c.DefaultPostForm("type", "trojan")
			c.JSON(200, controller.SetTrojanType(tType))
		})
		trojan.POST("/loglevel", func(c *gin.Context) {
			slevel := c.DefaultPostForm("level", "1")
			level, _ := strconv.Atoi(slevel)
			c.JSON(200, controller.SetLogLevel(level))
		})
		trojan.POST("/domain", func(c *gin.Context) {
			c.JSON(200, controller.SetDomain(c.PostForm("domain")))
		})
		trojan.GET("/log", func(c *gin.Context) {
			controller.Log(c)
		})
		trojan.GET("/port", func(c *gin.Context) {
			c.JSON(200, controller.GetPort())
		})
		trojan.POST("/port", func(c *gin.Context) {
			sPort := c.PostForm("port")
			port, err := strconv.Atoi(sPort)
			if err != nil {
				c.JSON(200, gin.H{"msg": "端口格式不正确"})
				return
			}
			c.JSON(200, controller.SetPort(port))
		})
		trojan.POST("/multiport", func(c *gin.Context) {
			portsStr := c.PostForm("ports")
			var portList []int
			if portsStr != "" {
				for _, part := range strings.Split(portsStr, ",") {
					part = strings.TrimSpace(part)
					if p, err := strconv.Atoi(part); err == nil && p > 0 && p <= 65535 {
						portList = append(portList, p)
					}
				}
			}
			c.JSON(200, controller.SetMultiPort(portList))
		})
		trojan.GET("/bbr", func(c *gin.Context) {
			c.JSON(200, controller.GetBBR())
		})
		trojan.POST("/bbr", func(c *gin.Context) {
			c.JSON(200, controller.SetBBR())
		})
	}
}

func dataRouter(router *gin.Engine) {
	data := router.Group("/trojan/data")
	data.Use(adminOnlyMiddleware())
	{
		data.POST("", func(c *gin.Context) {
			sID := c.PostForm("id")
			sQuota := c.PostForm("quota")
			id, _ := strconv.Atoi(sID)
			quota, _ := strconv.Atoi(sQuota)
			c.JSON(200, controller.SetData(uint(id), quota))
		})
		data.DELETE("", func(c *gin.Context) {
			sID := c.Query("id")
			id, _ := strconv.Atoi(sID)
			c.JSON(200, controller.CleanData(uint(id)))
		})
		data.POST("/resetDay", func(c *gin.Context) {
			dayStr := c.DefaultPostForm("day", "1")
			day, _ := strconv.Atoi(dayStr)
			c.JSON(200, controller.UpdateResetDay(uint(day)))
		})
		data.GET("/resetDay", func(c *gin.Context) {
			c.JSON(200, controller.GetResetDay())
		})
	}
}

func commonRouter(router *gin.Engine) {
	common := router.Group("/common")
	{
		common.GET("/version", func(c *gin.Context) {
			c.JSON(200, controller.Version())
		})
		common.GET("/serverInfo", func(c *gin.Context) {
			c.JSON(200, controller.ServerInfo())
		})
		common.GET("/clashRules", func(c *gin.Context) {
			c.JSON(200, controller.GetClashRules())
		})
		common.POST("/clashRules", func(c *gin.Context) {
			rules := c.PostForm("rules")
			c.JSON(200, controller.SetClashRules(rules))
		})
		common.DELETE("/clashRules", func(c *gin.Context) {
			c.JSON(200, controller.ResetClashRules())
		})
		common.POST("/loginInfo", func(c *gin.Context) {
			c.JSON(200, controller.SetLoginInfo(c.PostForm("title")))
		})
	}
}

func staticRouter(router *gin.Engine) {
	staticFs, _ := fs.Sub(f, "templates/static")
	router.StaticFS("/static", http.FS(staticFs))

	router.GET("/", func(c *gin.Context) {
		indexHTML, _ := f.ReadFile("templates/" + "index.html")
		c.Writer.Write(indexHTML)
	})
}

func noTokenRouter(router *gin.Engine) {
	router.GET("/trojan/user/subscribe", func(c *gin.Context) {
		controller.ClashSubInfo(c)
	})
}

// Start web启动入口
func Start(host string, port, timeout int, isSSL bool) {
	router := gin.Default()
	router.SetTrustedProxies(nil)
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	staticRouter(router)
	noTokenRouter(router)
	router.Use(Auth(router, timeout).MiddlewareFunc())
	trojanRouter(router)
	userRouter(router)
	dataRouter(router)
	commonRouter(router)
	controller.ScheduleTask()
	controller.CollectTask()
	_ = trojan.ApplyMultiPortRules()
	util.OpenPort(port)
	if isSSL {
		config := core.GetConfig()
		ssl := &config.SSl
		router.RunTLS(fmt.Sprintf("%s:%d", host, port), ssl.Cert, ssl.Key)
	} else {
		router.Run(fmt.Sprintf("%s:%d", host, port))
	}
}
