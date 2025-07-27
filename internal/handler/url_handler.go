package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/service"
)

type URLHandler struct {
	urlService service.URLService
}

func NewURLHandler(urlService service.URLService) *URLHandler {
	return &URLHandler{urlService: urlService}
}

func (h *URLHandler) parseUintParam(c *gin.Context, name string) (uint, bool) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(v), true
}

func (h *URLHandler) paginationFromQuery(c *gin.Context) repository.Pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	return repository.Pagination{Page: page, PageSize: size}
}

// @Summary Create URL row
// @Tags    urls
// @Accept  json
// @Produce json
// @Param   input body model.URLCreateRequestDTO true "URL to crawl"
// @Success 201 {object} map[string]uint "{id}"
// @Failure 400 {object} map[string]string "error"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /urls [post]
func (h *URLHandler) Create(c *gin.Context) {
	var requestDTO model.URLCreateRequestDTO
	if err := c.ShouldBindJSON(&requestDTO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	uidAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	inputDTO := &model.CreateURLInputDTO{
		UserID:      uidAny.(uint),
		OriginalURL: requestDTO.OriginalURL,
	}

	id, err := h.urlService.Create(inputDTO)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// @Summary List URLs (paginated)
// @Tags    urls
// @Produce json
// @Param   page      query int false "page" default(1) example(1)
// @Param   page_size query int false "page_size" default(10) example(10)
// @Success 200 {object} model.PaginatedResponse[model.URLDTO] "Paginated URL list"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /urls [get]
func (h *URLHandler) List(c *gin.Context) {
	uidAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uidAny.(uint)

	paginatedResult, err := h.urlService.List(userID, h.paginationFromQuery(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, paginatedResult)
}

// @Summary Get one URL row
// @Tags    urls
// @Produce json
// @Param   id path int true "URL ID"
// @Success 200 {object} model.URLDTO
// @Security JWTAuth
// @Security BasicAuth
// @Router  /urls/{id} [get]
func (h *URLHandler) Get(c *gin.Context) {
	id, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	dto, err := h.urlService.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}

// @Summary Update URL row
// @Tags    urls
// @Accept  json
// @Produce json
// @Param   id path int true "URL ID"
// @Param   input body model.UpdateURLInput true "fields"
// @Success 200 {object} map[string]string "updated"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /urls/{id} [put]
func (h *URLHandler) Update(c *gin.Context) {
	id, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}

	var in model.UpdateURLInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if err := h.urlService.Update(id, &in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// @Summary Delete URL row
// @Tags    urls
// @Produce json
// @Param   id path int true "URL ID"
// @Success 200 {object} map[string]string "deleted"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /urls/{id} [delete]
func (h *URLHandler) Delete(c *gin.Context) {
	id, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.urlService.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// @Summary Start crawl
// @Tags    urls
// @Produce json
// @Param   id path int true "URL ID"
// @Param   priority query int false "Priority (1-10, default 5)" default(5)
// @Success 202 {object} map[string]string "queued"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /urls/{id}/start [patch]
func (h *URLHandler) Start(c *gin.Context) {
	id, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}

	priorityStr := c.DefaultQuery("priority", "5")
	priority, err := strconv.Atoi(priorityStr)
	if err != nil || priority < 1 || priority > 10 {
		priority = 5 // Default priority
	}

	if priorityStr != "5" {
		if err := h.urlService.StartWithPriority(id, priority); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"status": model.StatusQueued, "priority": priority})
	} else {
		if err := h.urlService.Start(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"status": model.StatusQueued})
	}
}

// @Summary Stop crawl
// @Tags    urls
// @Produce json
// @Param   id path int true "URL ID"
// @Success 202 {object} map[string]string "stopped"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /urls/{id}/stop [patch]
func (h *URLHandler) Stop(c *gin.Context) {
	id, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.urlService.Stop(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": model.StatusStopped})
}

// @Summary Latest analysis snapshot + links
// @Tags    urls
// @Produce json
// @Param   id path int true "URL ID"
// @Success 200 {object} model.URLResultsDTO
// @Failure 404 {object} map[string]string "not found"
// @Failure 400 {object} map[string]string "bad request"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /urls/{id}/results [get]
func (h *URLHandler) Results(c *gin.Context) {
	id, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}

	url, analysisResults, links, err := h.urlService.ResultsWithDetails(id)
	if err != nil {

		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dto := &model.URLResultsDTO{
		URL:             url.ToDTO(),
		AnalysisResults: analysisResults,
		Links:           links,
	}

	c.JSON(http.StatusOK, dto)
}

// @Summary Adjust crawler workers
// @Tags    crawler
// @Produce json
// @Param   action query string true "Action (add or remove)" Enums(add, remove)
// @Param   count query int true "Number of workers to add/remove"
// @Success 200 {object} map[string]string "adjusted"
// @Failure 400 {object} map[string]string "bad request"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /crawler/workers [patch]
func (h *URLHandler) AdjustWorkers(c *gin.Context) {
	action := c.Query("action")
	countStr := c.Query("count")

	if action != "add" && action != "remove" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be 'add' or 'remove'"})
		return
	}

	count, err := strconv.Atoi(countStr)
	if err != nil || count <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "count must be a positive integer"})
		return
	}

	if err := h.urlService.AdjustCrawlerWorkers(action, count); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Successfully %s %d workers", action+"ed", count)})
}

// @Summary Get recent crawl results
// @Tags    crawler
// @Produce json
// @Success 200 {array} crawler.CrawlResult "array of recent crawl results"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /crawler/results [get]
func (h *URLHandler) GetCrawlResults(c *gin.Context) {

	c.JSON(http.StatusOK, gin.H{
		"message": "This endpoint would stream real-time crawl results. In a production implementation, consider using WebSockets or Server-Sent Events.",
		"note":    "The enhanced crawler now supports real-time result streaming via channels. This HTTP endpoint is just a placeholder.",
	})
}

func (h *URLHandler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	rg.POST("/urls", h.Create)
	rg.GET("/urls", h.List)
	rg.GET("/urls/:id", h.Get)
	rg.PUT("/urls/:id", h.Update)
	rg.DELETE("/urls/:id", h.Delete)
	rg.PATCH("/urls/:id/start", h.Start)
	rg.PATCH("/urls/:id/stop", h.Stop)
	rg.GET("/urls/:id/results", h.Results)
	rg.PATCH("/crawler/workers", h.AdjustWorkers)
	rg.GET("/crawler/results", h.GetCrawlResults)
}
