package service

import (
	"github.com/fatih/color"
	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/flag"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

type configService struct {
	db *gorm.DB
}

func NewConfigService() *configService {
	return &configService{db: dao.DB()}
}

func (s *configService) GetConfig() (*models.Config, error) {
	var config models.Config
	if err := s.db.First(&config).Error; err != nil {
		return nil, err
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.TopP == 0 {
		config.TopP = 1
	}
	if config.MaxHistory == 0 {
		config.MaxHistory = 10
	}
	if config.MaxIterations == 0 {
		config.MaxIterations = 10
	}

	return &config, nil
}

func (s *configService) UpdateConfig(config *models.Config) error {

	err := s.db.Save(config).Error
	if err != nil {
		return err
	}
	// 保存后，让其生效
	err = s.UpdateFlagFromDBConfig()
	if err != nil {
		return err
	}
	return nil
}

// UpdateFlagFromDBConfig 从数据库中加载配置，更新Flag方法中的值
func (s *configService) UpdateFlagFromDBConfig() error {
	cfg := flag.Init()
	m, err := s.GetConfig()
	if err != nil {
		return err
	}
	if cfg.PrintConfig {
		klog.Infof("已开启配置信息打印选项。下面是数据库配置的回显.\n%s:\n %+v\n%s\n", color.RedString("↓↓↓↓↓↓生产环境请务必关闭↓↓↓↓↓↓"), utils.ToJSON(m), color.RedString("↑↑↑↑↑↑生产环境请务必关闭↑↑↑↑↑↑"))
		cfg.ShowConfigCloseMethod()
	}

	cfg.AnySelect = m.AnySelect

	cfg.ApiKey = m.ApiKey
	cfg.ApiModel = m.ApiModel
	cfg.ApiURL = m.ApiURL

	if m.ProductName != "" {
		cfg.ProductName = m.ProductName
	}

	cfg.PrintConfig = m.PrintConfig

	if m.ResourceCacheTimeout > 0 {
		cfg.ResourceCacheTimeout = m.ResourceCacheTimeout
	}
	if cfg.ResourceCacheTimeout == 0 {
		cfg.ResourceCacheTimeout = 60
	}
	if m.Temperature > 0 {
		cfg.Temperature = m.Temperature
	}
	if m.TopP > 0 {
		cfg.TopP = m.TopP
	}
	if m.MaxHistory > 0 {
		cfg.MaxHistory = m.MaxHistory
	}
	if m.MaxIterations > 0 {
		cfg.MaxIterations = m.MaxIterations
	}
	// JwtTokenSecret 暂不启用，因为前端也要处理
	// cfg.JwtTokenSecret = m.JwtTokenSecret
	// LoginType 暂不启用，因为就一种password
	// cfg.LoginType = m.LoginType

	return nil
}
