package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yokitheyo/go_wallet_test/internal/model"
	"go.uber.org/zap"
)

type Repository interface {
	ChangeBalance(ctx context.Context, req model.WalletRequest) (int64, error)
	GetBalance(walletID uuid.UUID) (int64, error)
	Close() error
	DB() *sql.DB
}

type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
}

type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) ChangeBalance(ctx context.Context, req model.WalletRequest) (int64, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepo) GetBalance(walletID uuid.UUID) (int64, error) {
	args := m.Called(walletID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepo) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRepo) DB() *sql.DB {
	args := m.Called()
	return args.Get(0).(*sql.DB)
}

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

func (m *MockLogger) Info(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

func (m *MockLogger) Warn(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

func (m *MockLogger) Error(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

func (m *MockLogger) Fatal(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

var _ Repository = (*MockRepo)(nil)
var _ Logger = (*MockLogger)(nil)

func newTestRouter(repo Repository, logger Logger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		logger.Debug("Health check request")
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   "2023-01-01T00:00:00Z",
		})
	})

	v1 := router.Group("/api/v1")
	{
		v1.POST("/wallet", func(c *gin.Context) {
			var req model.WalletRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				logger.Error("invalid request payload", zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{
					"error":  "invalid request payload",
					"detail": err.Error(),
				})
				return
			}

			newBal, err := repo.ChangeBalance(c.Request.Context(), req)
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
		})

		v1.GET("/wallets/:id", func(c *gin.Context) {
			idStr := c.Param("id")
			id, err := uuid.Parse(idStr)
			if err != nil {
				logger.Warn("invalid uuid in getBalance", zap.String("id", idStr), zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
				return
			}
			bal, err := repo.GetBalance(id)
			if err != nil {
				logger.Error("internal error on GetBalance", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "detail": err.Error()})
				return
			}
			logger.Info("balance retrieved", zap.String("wallet_id", id.String()), zap.Int64("balance", bal))
			c.JSON(http.StatusOK, gin.H{"balance": bal})
		})
	}

	return router
}

func setupTestRouter() (*gin.Engine, *MockRepo, *MockLogger) {
	mockLogger := &MockLogger{}
	mockRepo := &MockRepo{}

	router := newTestRouter(mockRepo, mockLogger)

	return router, mockRepo, mockLogger
}

func TestHealthCheck(t *testing.T) {
	router, _, mockLogger := setupTestRouter()

	// Настраиваем ожидания для логгера
	mockLogger.On("Debug", "Health check request", mock.Anything).Return()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])

	mockLogger.AssertExpectations(t)
}

func TestDepositWithdraw_Success(t *testing.T) {
	router, mockRepo, mockLogger := setupTestRouter()

	walletID := uuid.New()
	req := model.WalletRequest{
		WalletID:      walletID,
		OperationType: model.Deposit,
		Amount:        100,
	}

	mockRepo.On("ChangeBalance", mock.Anything, req).Return(int64(1100), nil)
	mockLogger.On("Info", "balance changed successfully", mock.Anything).Return()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(1100), response["balance"])

	mockRepo.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestDepositWithdraw_InsufficientFunds(t *testing.T) {
	router, mockRepo, mockLogger := setupTestRouter()

	walletID := uuid.New()
	req := model.WalletRequest{
		WalletID:      walletID,
		OperationType: model.Withdraw,
		Amount:        1000,
	}

	mockRepo.On("ChangeBalance", mock.Anything, req).Return(int64(0), fmt.Errorf("insufficient balance"))
	mockLogger.On("Warn", "insufficient funds", mock.Anything).Return()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "insufficient funds", response["error"])

	mockRepo.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestDepositWithdraw_InvalidRequest(t *testing.T) {
	router, _, mockLogger := setupTestRouter()

	// Настраиваем ожидания для логгера
	mockLogger.On("Error", "invalid request payload", mock.Anything).Return()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/v1/wallet", bytes.NewBufferString(`{"invalid": "json"`))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockLogger.AssertExpectations(t)
}

func TestGetBalance_Success(t *testing.T) {
	router, mockRepo, mockLogger := setupTestRouter()

	walletID := uuid.New()
	expectedBalance := int64(1000)

	mockRepo.On("GetBalance", walletID).Return(expectedBalance, nil)
	mockLogger.On("Info", "balance retrieved", mock.Anything).Return()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/wallets/"+walletID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(expectedBalance), response["balance"])

	mockRepo.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestGetBalance_InvalidUUID(t *testing.T) {
	router, _, mockLogger := setupTestRouter()

	// Настраиваем ожидания для логгера
	mockLogger.On("Warn", "invalid uuid in getBalance", mock.Anything).Return()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/wallets/invalid-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid uuid", response["error"])

	mockLogger.AssertExpectations(t)
}
