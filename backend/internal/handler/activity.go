package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/weibaohui/opendeepwiki/backend/config"
)

type ActivityHandler struct {
	cfg *config.Config
}

func NewActivityHandler(cfg *config.Config) *ActivityHandler {
	return &ActivityHandler{cfg: cfg}
}

type ActivityConfigResponse struct {
	Enabled         bool   `json:"enabled"`
	DefaultInterval string `json:"default_interval"`
	DecreaseUnit    string `json:"decrease_unit"`
	CheckInterval   string `json:"check_interval"`
	ResetHour       int    `json:"reset_hour"`
}

func (h *ActivityHandler) GetConfig(c *gin.Context) {
	cfg := h.cfg.Activity
	c.JSON(http.StatusOK, ActivityConfigResponse{
		Enabled:         cfg.Enabled,
		DefaultInterval: cfg.DefaultInterval.String(),
		DecreaseUnit:    cfg.DecreaseUnit.String(),
		CheckInterval:   cfg.CheckInterval.String(),
		ResetHour:       cfg.ResetHour,
	})
}

type UpdateActivityConfigRequest struct {
	Enabled         *bool   `json:"enabled" binding:"required"`
	DefaultInterval *string `json:"default_interval" binding:"required"`
	DecreaseUnit    *string `json:"decrease_unit" binding:"required"`
	CheckInterval   *string `json:"check_interval" binding:"required"`
	ResetHour       *int    `json:"reset_hour" binding:"required"`
}

func (h *ActivityHandler) UpdateConfig(c *gin.Context) {
	var req UpdateActivityConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse durations
	defaultInterval, err := time.ParseDuration(*req.DefaultInterval)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid default_interval format"})
		return
	}

	decreaseUnit, err := time.ParseDuration(*req.DecreaseUnit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid decrease_unit format"})
		return
	}

	checkInterval, err := time.ParseDuration(*req.CheckInterval)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid check_interval format"})
		return
	}

	// Validate reset hour
	if *req.ResetHour < 0 || *req.ResetHour > 23 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reset hour must be between 0 and 23"})
		return
	}

	// Update config
	h.cfg.Activity.Enabled = *req.Enabled
	h.cfg.Activity.DefaultInterval = defaultInterval
	h.cfg.Activity.DecreaseUnit = decreaseUnit
	h.cfg.Activity.CheckInterval = checkInterval
	h.cfg.Activity.ResetHour = *req.ResetHour

	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully"})
}

func (h *ActivityHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/activity/config", h.GetConfig)
	router.PUT("/activity/config", h.UpdateConfig)
}
