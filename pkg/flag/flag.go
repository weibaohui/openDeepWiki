package flag

import (
	"flag"
	"os"
	"strconv"
	"sync"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"k8s.io/klog/v2"
)

var config *Config
var once sync.Once

type Config struct {
	Port                 int     // gin 监听端口
	KubeConfig           string  // KUBECONFIG文件路径
	ApiKey               string  // OPENAI_API_KEY
	ApiURL               string  // OPENAI_API_URL
	ApiModel             string  // OPENAI_MODEL
	Debug                bool    // 调试模式，同步修改所有的debug模式
	LogV                 int     // klog的日志级别klog.V(this)
	InCluster            bool    // 是否集群内模式
	LoginType            string  // password,oauth,token,.. 登录方式，默认为password
	EnableTempAdmin      bool    // 是否启用临时管理员账户配置
	AdminUserName        string  // 管理员用户名，启用临时管理员账户配置后生效
	AdminPassword        string  // 管理员密码，启用临时管理员账户配置后生效
	JwtTokenSecret       string  // JWT token secret
	NodeShellImage       string  // nodeShell 镜像
	KubectlShellImage    string  // kubectlShell 镜像
	ImagePullTimeout     int     // 镜像拉取超时时间（秒）
	SqlitePath           string  // sqlite 数据库路径
	AnySelect            bool    // 是否开启任意选择，默认开启
	PrintConfig          bool    // 是否打印配置信息
	Version              string  // 版本号，由编译时自动注入
	GitCommit            string  // git commit, 由编译时自动注入
	GitTag               string  // git tag, 由编译时自动注入
	GitRepo              string  // git仓库地址, 由编译时自动注入
	BuildDate            string  // 编译时间, 由编译时自动注入
	ConnectCluster       bool    // 启动程序后，是否自动连接发现的集群，默认关闭
	ProductName          string  // 产品名称
	ResourceCacheTimeout int     // 资源缓存时间（秒）
	Temperature          float32 // 模型温度
	TopP                 float32 //  模型topP参数
	MaxHistory           int32   //  模型对话上下文历史记录数
}

func Init() *Config {
	once.Do(func() {
		config = &Config{}
		loadEnv()
		config.InitFlags()

	})
	return config
}
func (c *Config) ShowConfigInfo() {
	// 根据PrintConfig决定是否打印配置信息
	if c.PrintConfig {
		klog.Infof("配置加载顺序:1.启动参数->2.环境变量->3.数据库参数设置（界面配置）,后加载的配置覆盖前面的配置")
		klog.Infof("已开启配置信息打印选项.\n%s:\n %+v\n%s\n", color.RedString("↓↓↓↓↓↓生产环境请务必关闭↓↓↓↓↓↓"), utils.ToJSON(config), color.RedString("↑↑↑↑↑生产环境请务必关闭↑↑↑↑↑↑"))
		c.ShowConfigCloseMethod()
	}
}
func (c *Config) ShowConfigCloseMethod() {
	klog.Infof("关闭打印选项方法：\n1. %s\n2. %s \n3. %s  \n", color.RedString("平台管理-参数设置-打印配置，选择关闭"), color.RedString("启动参数 --print-config = false"), color.RedString("env PRINT_CONFIG=false"))
}
func loadEnv() {
	env := os.Getenv("OPEN_DEEP_WIKI_ENV")
	if "" == env {
		// 默认开发环境加载".env.dev.local"
		env = "dev"
	}
	// 依次加载并覆盖
	if err := godotenv.Overload(".env", ".env."+env+".local"); err != nil {
		klog.Warningf("Error loading .env file: %v", err)
	}
}
func (c *Config) InitFlags() {

	// 如果有其他类似的引用，请参考下面的方式进行整合
	// 初始化klog
	klog.InitFlags(nil)

	// 将Go的flag绑定到pflag
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// 环境变量绑定
	// 默认端口为3721
	defaultPort := getEnvAsInt("PORT", 3721)

	// 默认apiKey为环境变量OPENAI_API_KEY/OPENAI_API_URL/
	defaultApiKey := getEnv("OPENAI_API_KEY", "")
	defaultApiURL := getEnv("OPENAI_API_URL", "")
	defaultModel := getEnv("OPENAI_MODEL", "Qwen/Qwen2.5-7B-Instruct")

	// 默认登录方式为password
	defaultLoginType := getEnv("LOGIN_TYPE", "password")
	defaultAdminUserName := getEnv("ADMIN_USERNAME", "")
	defaultAdminPassword := getEnv("ADMIN_PASSWORD", "")

	// 默认debug为false
	defaultDebug := getEnvAsBool("DEBUG", false)

	// jwt token secret
	defaultJwtTokenSecret := getEnv("JWT_TOKEN_SECRET", "your-secret-key")

	// 输出日志的级别
	defaultLogV := getEnv("LOG_V", "2")

	// sqlite数据库文件路径
	defaultSqlitePath := getEnv("SQLITE_PATH", "./data/openDeepWiki.db")

	// 默认开启任意选择
	defaultAnySelect := getEnvAsBool("ANY_SELECT", true)

	// 默认不打印配置
	defaultPrintConfig := getEnvAsBool("PRINT_CONFIG", false)
	// 默认不启用临时管理员账户配置
	defaultEnableTempAdmin := getEnvAsBool("ENABLE_TEMP_ADMIN", false)

	// 默认产品名称
	defaultProductName := getEnv("PRODUCT_NAME", "OpenDeepWiki")

	// 默认资源缓存时间为60秒
	defaultResourceCacheTimeout := getEnvAsInt("RESOURCE_CACHE_TIMEOUT", 60)

	pflag.BoolVarP(&c.Debug, "debug", "d", defaultDebug, "调试模式")
	pflag.IntVarP(&c.Port, "port", "p", defaultPort, "监听端口,默认3721")
	pflag.StringVarP(&c.ApiKey, "chatgpt-key", "k", defaultApiKey, "大模型的自定义API Key")
	pflag.StringVarP(&c.ApiURL, "chatgpt-url", "u", defaultApiURL, "大模型的自定义API URL")
	pflag.StringVarP(&c.ApiModel, "chatgpt-model", "m", defaultModel, "大模型的自定义模型名称")
	pflag.StringVar(&c.LoginType, "login-type", defaultLoginType, "登录方式，password, oauth, token等,default is password")
	pflag.BoolVar(&c.EnableTempAdmin, "enable-temp-admin", defaultEnableTempAdmin, "是否启用临时管理员账户配置，默认关闭")
	pflag.StringVar(&c.AdminUserName, "admin-username", defaultAdminUserName, "管理员用户名，启用临时管理员账户配置后生效")
	pflag.StringVar(&c.AdminPassword, "admin-password", defaultAdminPassword, "管理员密码，启用临时管理员账户配置后生效")
	pflag.StringVar(&c.JwtTokenSecret, "jwt-token-secret", defaultJwtTokenSecret, "登录后生成JWT token 使用的Secret")
	pflag.IntVar(&c.LogV, "log-v", 2, "klog的日志级别klog.V(2)")
	pflag.StringVar(&c.SqlitePath, "sqlite-path", defaultSqlitePath, "sqlite数据库文件路径，默认./data/openDeepWiki.db")
	pflag.BoolVar(&c.AnySelect, "any-select", defaultAnySelect, "是否开启任意选择，默认开启")
	pflag.BoolVar(&c.PrintConfig, "print-config", defaultPrintConfig, "是否打印配置信息，默认关闭")
	pflag.StringVar(&c.ProductName, "product-name", defaultProductName, "产品名称，默认为OpenDeepWiki")
	pflag.IntVar(&c.ResourceCacheTimeout, "resource-cache-timeout", defaultResourceCacheTimeout, "资源缓存时间（秒），默认60秒")
	// 检查是否设置了 --v 参数
	if vFlag := pflag.Lookup("v"); vFlag == nil || vFlag.Value.String() == "0" {
		// 如果没有设置，手动将 --v 设置为 环境变量值
		_ = flag.Set("v", defaultLogV)
	}
	pflag.Parse()

}

// getEnv 读取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt 读取环境变量，如果不存在则返回默认值
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool 获取环境变量的布尔值，支持 "true"/"false"（大小写不敏感）和 "1"/"0"，否则返回默认值
func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
