package handler

import (
	"net/http"
	"strconv"
	"subscriptionsservice/internal/models"
	"subscriptionsservice/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SubscriptionHandler отвечает за обработку HTTP-запросов подписок
type SubscriptionHandler struct {
	service *service.SubscriptionService
	log     *zap.Logger
}

func NewSubscriptionHandler(srv *service.SubscriptionService, log *zap.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{service: srv, log: log}
}

// RegisterRoutes регистрирует маршруты
func (h *SubscriptionHandler) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/subscriptions")

	g.POST("/", h.CreateSubscription)
	g.GET("/", h.List)
	g.GET("/:id", h.GetByID)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	g.POST("/summary", h.Summary)
}

// CreateSubscription godoc
// @Summary Создать подписку
// @Description Создает новую подписку пользователя
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription body models.Subscription true "Subscription"
// @Success 201 {object} models.Subscription
// @Failure 400 {object} map[string]string
// @Router /subscriptions/ [post]
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	var sub models.Subscription
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateSubscription(c.Request.Context(), &sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// List godoc
// @Summary Получить список подписок
// @Tags subscriptions
// @Produce json
// @Success 200 {array} models.Subscription
// @Router /subscriptions/ [get]
func (h *SubscriptionHandler) List(c *gin.Context) {
	subs, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, subs)
}

// GetByID godoc
// @Summary Получить подписку по ID
// @Tags subscriptions
// @Produce json
// @Param id path int true "Subscription ID"
// @Success 200 {object} models.Subscription
// @Failure 404 {object} map[string]string
// @Router /subscriptions/{id} [get]
func (h *SubscriptionHandler) GetByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	sub, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

// Update godoc
// @Summary Обновить подписку
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path int true "Subscription ID"
// @Param subscription body models.Subscription true "Subscription"
// @Success 200 {object} models.Subscription
// @Router /subscriptions/{id} [put]
func (h *SubscriptionHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var sub models.Subscription
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sub.ID = id
	if err := h.service.Update(c.Request.Context(), &sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

// Delete godoc
// @Summary Удалить подписку
// @Tags subscriptions
// @Param id path int true "Subscription ID"
// @Success 204
// @Router /subscriptions/{id} [delete]
func (h *SubscriptionHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// Summary godoc
// @Summary Получить сумму подписок за период
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param summary body models.SummaryRequest true "Summary query"
// @Success 200 {object} map[string]int
// @Router /subscriptions/summary [post]
func (h *SubscriptionHandler) Summary(c *gin.Context) {
	var req models.SummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.log.Debug("got dates",
		zap.Time("from", req.From.Time),
		zap.Time("to", req.To.Time),
	)

	sum, err := h.service.Summary(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": sum})
}
