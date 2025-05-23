package service

import (
	"fmt"

	"github.com/weibaohui/openDeepWiki/pkg/ai"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/flag"
	"k8s.io/klog/v2"
)

type aiService struct {
}

var local ai.IAI

func (c *aiService) DefaultClient() (ai.IAI, error) {
	enable := c.IsEnabled()
	if !enable {
		return nil, fmt.Errorf("ChatGPT功能未开启")
	}

	if local != nil {
		return local, nil
	}

	if client, err := c.openAIClient(); err == nil {
		local = client
	}
	return local, nil

}
func (c *aiService) ReloadDefaultClient() (ai.IAI, error) {
	 
	if client, err := c.openAIClient(); err == nil {
		local = client
	}
	return local, nil

}

func (c *aiService) openAIClient() (ai.IAI, error) {
	cfg := flag.Init()

	aiProvider := ai.Provider{
		Name:        "openai",
		Model:       cfg.ApiModel,
		Password:    cfg.ApiKey,
		BaseURL:     cfg.ApiURL,
		Temperature: 0.7,
		TopP:        1,
		MaxHistory:  10,
		TopK:        0,
		MaxTokens:   1000,
	}

	// Temperature: 0.7,
	// 	TopP:        1,
	// 		MaxHistory:  10,
	if cfg.Temperature > 0 {
		aiProvider.Temperature = cfg.Temperature
	}
	if cfg.TopP > 0 {
		aiProvider.TopP = cfg.TopP
	}
	if cfg.MaxHistory > 0 {
		aiProvider.MaxHistory = cfg.MaxHistory
	}

	if cfg.Debug {
		klog.V(4).Infof("ai BaseURL: %v\n", aiProvider.BaseURL)
		klog.V(4).Infof("ai Model : %v\n", aiProvider.Model)
		klog.V(4).Infof("ai Key: %v\n", utils.MaskString(aiProvider.Password, 5))
	}

	aiClient := ai.NewClient(aiProvider.Name)
	if err := aiClient.Configure(&aiProvider); err != nil {
		return nil, err
	}
	return aiClient, nil
}

func (c *aiService) IsEnabled() bool {
	return true
}
