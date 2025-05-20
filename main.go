package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/fatih/color"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/controller/admin/config"
	"github.com/weibaohui/openDeepWiki/pkg/controller/admin/mcp"
	"github.com/weibaohui/openDeepWiki/pkg/controller/admin/user"
	"github.com/weibaohui/openDeepWiki/pkg/controller/chat"
	"github.com/weibaohui/openDeepWiki/pkg/controller/doc"
	"github.com/weibaohui/openDeepWiki/pkg/controller/login"
	"github.com/weibaohui/openDeepWiki/pkg/controller/param"
	"github.com/weibaohui/openDeepWiki/pkg/controller/repo"
	"github.com/weibaohui/openDeepWiki/pkg/controller/sso"
	"github.com/weibaohui/openDeepWiki/pkg/controller/user/mcpkey"
	"github.com/weibaohui/openDeepWiki/pkg/controller/user/profile"
	"github.com/weibaohui/openDeepWiki/pkg/flag"
	"github.com/weibaohui/openDeepWiki/pkg/mcpservers"
	"github.com/weibaohui/openDeepWiki/pkg/middleware"
	_ "github.com/weibaohui/openDeepWiki/pkg/models" // 注册模型
	"github.com/weibaohui/openDeepWiki/pkg/service"
	"k8s.io/klog/v2"
)

//go:embed ui/dist/*
var embeddedFiles embed.FS
var Version string
var GitCommit string
var GitTag string
var GitRepo string
var BuildDate string

// Init 完成服务的初始化，包括加载配置、设置版本信息、初始化 AI 服务、注册集群及其回调，并启动资源监控。
func Init() {
	// 初始化配置
	cfg := flag.Init()
	// 从数据库中更新配置
	err := service.ConfigService().UpdateFlagFromDBConfig()
	if err != nil {
		klog.Errorf("加载数据库内配置信息失败 error: %v", err)
	}
	cfg.Version = Version
	cfg.GitCommit = GitCommit
	cfg.GitTag = GitTag
	cfg.GitRepo = GitRepo
	cfg.BuildDate = BuildDate
	cfg.ShowConfigInfo()

	// 打印版本和 Git commit 信息
	klog.V(2).Infof("版本: %s\n", Version)
	klog.V(2).Infof("Git Commit: %s\n", GitCommit)
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	go service.McpService().Init()

}

// main 启动并配置 Web 服务，注册所有路由和中间件，加载嵌入式静态资源，并监听配置端口。
// 包括管理后台、认证、AI 聊天、文档、用户自助、仓库管理等 API 分组，以及健康检查和静态页面服务。
// 启动并配置 Web 服务，注册所有 API 路由、静态资源和中间件，输出本地和网络访问地址，启动失败时记录致命错误。
func main() {
	Init()

	r := gin.Default()

	cfg := flag.Init()
	if !cfg.Debug {
		// debug 模式可以崩溃
		r.Use(middleware.CustomRecovery())
	}
	r.Use(cors.Default())
	r.Use(gzip.Gzip(gzip.BestCompression))
	r.Use(middleware.SetCacheHeaders())
	r.Use(middleware.RequireLogin())

	r.MaxMultipartMemory = 100 << 20 // 100 MiB

	// MCP SSE SERVER
	sseServer := mcpservers.NewMCPSSEServer()
	r.GET("/mcp/sse", mcpservers.Adapt(sseServer.SSEHandler))
	r.POST("/mcp/sse", mcpservers.Adapt(sseServer.SSEHandler))
	r.POST("/mcp/message", mcpservers.Adapt(sseServer.MessageHandler))

	// Admin routes
	adminGroup := r.Group("/admin")
	adminGroup.Use(middleware.RequireAdmin())
	{
		// user 平台管理员可操作，管理用户
		adminGroup.GET("/user/list", user.List)
		adminGroup.POST("/user/save", user.Save)
		adminGroup.POST("/user/delete/:ids", user.Delete)
		adminGroup.POST("/user/update_psw/:id", user.UpdatePsw)
		adminGroup.GET("/user/option_list", user.UserOptionList)
		// 2FA 平台管理员可操作，管理用户
		adminGroup.POST("/user/2fa/disable/:id", user.Disable2FA)
		// user_group
		adminGroup.GET("/user_group/list", user.ListUserGroup)
		adminGroup.POST("/user_group/save", user.SaveUserGroup)
		adminGroup.POST("/user_group/delete/:ids", user.DeleteUserGroup)
		adminGroup.GET("/user_group/option_list", user.GroupOptionList)

		// 平参数配置
		adminGroup.GET("/config/all", config.GetConfig)
		adminGroup.POST("/config/update", config.UpdateConfig)

		// mcp
		adminGroup.GET("/mcp/list", mcp.ServerList)
		adminGroup.GET("/mcp/server/:name/tools/list", mcp.ToolsList)
		adminGroup.POST("/mcp/connect/:name", mcp.Connect)
		adminGroup.POST("/mcp/delete", mcp.Delete)
		adminGroup.POST("/mcp/save", mcp.AddOrUpdate)
		adminGroup.POST("/mcp/save/id/:id/status/:status", mcp.QuickSave)
		adminGroup.POST("/mcp/tool/save/id/:id/status/:status", mcp.ToolQuickSave)
		adminGroup.GET("/mcp/log/list", mcp.MCPLogList)

		// sso
		adminGroup.GET("/config/sso/list", config.SSOConfigList)
		adminGroup.POST("/config/sso/save", config.SSOConfigSave)
		adminGroup.POST("/config/sso/delete/:ids", config.SSOConfigDelete)
		adminGroup.POST("/config/sso/save/id/:id/status/:enabled", config.SSOConfigQuickSave)

	}

	// 挂载子目录
	pagesFS, _ := fs.Sub(embeddedFiles, "ui/dist/pages")
	r.StaticFS("/public/pages", http.FS(pagesFS))
	assetsFS, _ := fs.Sub(embeddedFiles, "ui/dist/assets")
	r.StaticFS("/assets", http.FS(assetsFS))

	r.GET("/favicon.ico", func(c *gin.Context) {
		favicon, _ := embeddedFiles.ReadFile("ui/dist/favicon.ico")
		c.Data(http.StatusOK, "image/x-icon", favicon)
	})

	// 直接返回 index.html
	r.GET("/", func(c *gin.Context) {
		index, err := embeddedFiles.ReadFile("ui/dist/index.html") // 这里路径必须匹配
		if err != nil {
			c.String(http.StatusInternalServerError, "Internal Server Error")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	auth := r.Group("/auth")
	{
		auth.POST("/login", login.LoginByPassword)
		auth.GET("/sso/config", sso.GetSSOConfig)
		auth.GET("/oidc/:name/sso", sso.GetAuthCodeURL)
		auth.GET("/oidc/:name/callback", sso.HandleCallback)
	}

	// 公共参数
	params := r.Group("/params", middleware.RequireLogin())
	{
		// 获取当前登录用户的角色，登录即可
		params.GET("/user/role", param.UserRole)
		// 获取某个配置项
		params.GET("/config/:key", param.Config)
		// 获取当前软件版本信息
		params.GET("/version", param.Version)

	}
	ai := r.Group("/ai", middleware.RequireLogin())
	{

		// chatgpt
		ai.GET("/chat/any_selection", chat.AnySelection)
		ai.GET("/chat/ws_chatgpt", chat.GPTShell)
		ai.GET("/chat/ws_chatgpt/history", chat.History)
		ai.GET("/chat/ws_chatgpt/history/reset", chat.Reset)

	}
	dc := r.Group("/doc", middleware.RequireLogin())
	{
		dc.POST("/repo/:id/analysis", doc.Analysis)
		dc.POST("/repo/init", doc.Init)
		dc.GET("/repo/logs", doc.GetLatestLogs)
		dc.GET("/repo/:id/analysis/history", doc.GetAnalysisHistory)
		dc.GET("/repo/analysis/:id/results", doc.GetAnalysisResults)
	}

	mgm := r.Group("/mgm", middleware.RequireLogin())
	{

		// user profile 用户自助操作
		mgm.GET("/user/profile", profile.Profile)
		mgm.POST("/user/profile/update_psw", profile.UpdatePsw)
		// user profile 2FA 用户自助操作
		mgm.POST("/user/profile/2fa/generate", profile.Generate2FASecret)
		mgm.POST("/user/profile/2fa/disable", profile.Disable2FA)
		mgm.POST("/user/profile/2fa/enable", profile.Enable2FA)

		// MCP密钥管理
		mgm.GET("/user/profile/mcpkeys/list", mcpkey.List)
		mgm.POST("/user/profile/mcpkeys/create", mcpkey.Create)
		mgm.POST("/user/profile/mcpkeys/delete/:id", mcpkey.Delete)

		// 代码仓库管理
		repo.RegisterRoutes(mgm)
	}

	showBootInfo(Version, flag.Init().Port)
	err := r.Run(fmt.Sprintf(":%d", flag.Init().Port))
	if err != nil {
		klog.Fatalf("Error %v", err)
	}
}

func showBootInfo(version string, port int) {

	// 获取本机所有 IP 地址
	ips, err := utils.GetLocalIPs()
	if err != nil {
		klog.Fatalf("获取本机 IP 失败: %v", err)
		os.Exit(1)
	}
	// 打印 Vite 风格的启动信息
	color.Green("%s  启动成功", version)
	fmt.Printf("%s  ", color.GreenString("➜"))
	fmt.Printf("%s    ", color.New(color.Bold).Sprint("Local:"))
	fmt.Printf("%s\n", color.MagentaString("http://localhost:%d/", port))

	for _, ip := range ips {
		fmt.Printf("%s  ", color.GreenString("➜"))
		fmt.Printf("%s  ", color.New(color.Bold).Sprint("Network:"))
		fmt.Printf("%s\n", color.MagentaString("http://%s:%d/", ip, port))
	}

}
