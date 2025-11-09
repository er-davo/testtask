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
// @Param subscription body models.Subscription true "Данные подписки"
// @Success 201 {object} models.Subscription "Успешное создание"
// @Failure 400 {object} map[string]string "Некорректный запрос"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /subscriptions/ [post]
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	var sub models.Subscription
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := models.Validate(&sub); err != nil {
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
// @Description Возвращает список подписок с пагинацией
// @Tags subscriptions
// @Produce json
// @Param limit query int false "Количество элементов на странице (по умолчанию 10)"
// @Param offset query int false "Смещение (по умолчанию 0)"
// @Success 200 {object} map[string]interface{} "data: список подписок, limit, offset"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /subscriptions/ [get]
func (h *SubscriptionHandler) List(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		limit = 10
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	subs, err := h.service.List(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list subscriptions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   subs,
		"limit":  limit,
		"offset": offset,
	})
}

// GetByID godoc
// @Summary Получить подписку по ID
// @Description Возвращает данные подписки по ID
// @Tags subscriptions
// @Produce json
// @Param id path int true "ID подписки"
// @Success 200 {object} models.Subscription "Найдена"
// @Failure 400 {object} map[string]string "Некорректный ID"
// @Failure 404 {object} map[string]string "Не найдена"
// @Router /subscriptions/{id} [get]
func (h *SubscriptionHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	sub, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// Update godoc
// @Summary Обновить подписку
// @Description Обновляет данные существующей подписки
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path int true "ID подписки"
// @Param subscription body models.Subscription true "Обновленные данные подписки"
// @Success 200 {object} models.Subscription "Обновлено"
// @Failure 400 {object} map[string]string "Некорректные данные"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /subscriptions/{id} [put]
func (h *SubscriptionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var sub models.Subscription
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	sub.ID = id

	if err := models.Validate(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.Update(c.Request.Context(), &sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update subscription"})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// Delete godoc
// @Summary Удалить подписку
// @Description Удаляет подписку по ID
// @Tags subscriptions
// @Param id path int true "ID подписки"
// @Success 204 "Удалено"
// @Failure 400 {object} map[string]string "Некорректный ID"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /subscriptions/{id} [delete]
func (h *SubscriptionHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete subscription"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Summary godoc
// @Summary Получить сумму подписок за период
// @Description Возвращает общую сумму подписок за указанный период с учетом фильтров
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param summary body models.SummaryRequest true "Параметры периода и фильтров"
// @Success 200 {object} map[string]int "Сумма подписок"
// @Failure 400 {object} map[string]string "Некорректный запрос"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /subscriptions/summary [post]
func (h *SubscriptionHandler) Summary(c *gin.Context) {
	var req models.SummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := models.Validate(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sum, err := h.service.Summary(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": sum})
}
