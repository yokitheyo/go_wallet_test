package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yokitheyo/go_wallet_test/internal/middleware"
	"github.com/yokitheyo/go_wallet_test/internal/model"
	"github.com/yokitheyo/go_wallet_test/internal/repo"
	"go.uber.org/zap"
)

func NewRouter(r *repo.Repo, logger *zap.Logger) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))

	v1 := router.Group("/api/v1")
	{
		v1.POST("/wallet", depositWithdraw(r, logger))
		v1.GET("/wallets/:id", getBalance(r, logger))
	}
	return router
}

func depositWithdraw(r *repo.Repo, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.WalletRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logger.Error("invalid request payload", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "invalid request payload",
				"detail": err.Error(),
			})
			return
		}

		newBal, err := r.ChangeBalance(req)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "insufficient") {
				logger.Warn("insufficient funds", zap.Any("request", req), zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient funds"})
			} else {
				logger.Error("internal error on ChangeBalance", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":  "internal error",
					"detail": err.Error(),
				})
			}
			return
		}

		logger.Info("balance changed successfully", zap.Any("request", req), zap.Int64("new_balance", newBal))
		c.JSON(http.StatusOK, gin.H{"balance": newBal})
	}
}

func getBalance(r *repo.Repo, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			logger.Warn("invalid uuid in getBalance", zap.String("id", idStr), zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
			return
		}
		bal, err := r.GetBalance(id)
		if err != nil {
			logger.Error("internal error on GetBalance", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "detail": err.Error()})
			return
		}
		logger.Info("balance retrieved", zap.String("wallet_id", id.String()), zap.Int64("balance", bal))
		c.JSON(http.StatusOK, gin.H{"balance": bal})
	}
}
