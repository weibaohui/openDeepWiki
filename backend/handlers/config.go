package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/opendeepwiki/backend/config"
)

type ConfigHandler struct {
	cfg *config.Config
}

func NewConfigHandler(cfg *config.Config) *ConfigHandler {
	return &ConfigHandler{cfg: cfg}
}

type ConfigResponse struct {
	LLM    LLMConfigResponse    `json:"llm"`
	GitHub GitHubConfigResponse `json:"github"`
}

type LLMConfigResponse struct {
	APIURL    string `json:"api_url"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
}

type GitHubConfigResponse struct {
	Token string `json:"token"`
}

func (h *ConfigHandler) Get(c *gin.Context) {
	resp := ConfigResponse{
		LLM: LLMConfigResponse{
			APIURL:    h.cfg.LLM.APIURL,
			APIKey:    maskKey(h.cfg.LLM.APIKey),
			Model:     h.cfg.LLM.Model,
			MaxTokens: h.cfg.LLM.MaxTokens,
		},
		GitHub: GitHubConfigResponse{
			Token: maskKey(h.cfg.GitHub.Token),
		},
	}

	c.JSON(http.StatusOK, resp)
}

type UpdateConfigRequest struct {
	LLM    *LLMConfigRequest    `json:"llm,omitempty"`
	GitHub *GitHubConfigRequest `json:"github,omitempty"`
}

type LLMConfigRequest struct {
	APIURL    string `json:"api_url,omitempty"`
	APIKey    string `json:"api_key,omitempty"`
	Model     string `json:"model,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
}

type GitHubConfigRequest struct {
	Token string `json:"token,omitempty"`
}

func (h *ConfigHandler) Update(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.LLM != nil {
		if req.LLM.APIURL != "" {
			h.cfg.LLM.APIURL = req.LLM.APIURL
		}
		if req.LLM.APIKey != "" && req.LLM.APIKey != maskKey(h.cfg.LLM.APIKey) {
			h.cfg.LLM.APIKey = req.LLM.APIKey
		}
		if req.LLM.Model != "" {
			h.cfg.LLM.Model = req.LLM.Model
		}
		if req.LLM.MaxTokens > 0 {
			h.cfg.LLM.MaxTokens = req.LLM.MaxTokens
		}
	}

	if req.GitHub != nil {
		if req.GitHub.Token != "" && req.GitHub.Token != maskKey(h.cfg.GitHub.Token) {
			h.cfg.GitHub.Token = req.GitHub.Token
		}
	}

	config.UpdateConfig(h.cfg)

	if err := h.cfg.Save("config.yaml"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config updated"})
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "********"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
