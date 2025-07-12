package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/fuzumoe/urlinsight-backend/internal/service"
)

type URLHandler struct {
	urlService service.URLService
}

func NewURLHandler(svc service.URLService) *URLHandler { return &URLHandler{urlService: svc} }

func parseUintParam(c *gin.Context, name string) (uint, bool) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(v), true
}

func paginationFromQuery(c *gin.Context) repository.Pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	return repository.Pagination{Page: page, PageSize: size}
}

// @Summary Create URL row
// @Tags    urls
// @Accept  json
// @Produce json
// @Param   input body model.CreateURLInput true "URL to crawl"
// @Success 201 {object} map[string]uint "{id}"
// @Failure 400 {object} map[string]string "error"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /api/urls [post]
func (h *URLHandler) Create(c *gin.Context) {
	var in model.CreateURLInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	id, err := h.urlService.Create(&in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// @Summary List URLs (paginated)
// @Tags    urls
// @Produce json
// @Param   page      query int false "page"
// @Param   page_size query int false "page_size"
// @Success 200 {array} model.URLDTO
// @Security JWTAuth
// @Security BasicAuth
// @Router  /api/urls [get]
func (h *URLHandler) List(c *gin.Context) {
	uidAny, _ := c.Get("userID")
	userID := uidAny.(uint)

	items, err := h.urlService.List(userID, paginationFromQuery(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

// @Summary Get one URL row
// @Tags    urls
// @Produce json
// @Param   id path int true "URL ID"
// @Success 200 {object} model.URLDTO
// @Security JWTAuth
// @Security BasicAuth
// @Router  /api/urls/{id} [get]
func (h *URLHandler) Get(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
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
// @Router  /api/urls/{id} [put]
func (h *URLHandler) Update(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
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
// @Router  /api/urls/{id} [delete]
func (h *URLHandler) Delete(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
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
// @Success 202 {object} map[string]string "queued"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /api/urls/{id}/start [patch]
func (h *URLHandler) Start(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.urlService.Start(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": model.StatusQueued})
}

// @Summary Stop crawl
// @Tags    urls
// @Produce json
// @Param   id path int true "URL ID"
// @Success 202 {object} map[string]string "stopped"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /api/urls/{id}/stop [patch]
func (h *URLHandler) Stop(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
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
// @Success 200 {object} model.URLDTO
// @Security JWTAuth
// @Security BasicAuth
// @Router  /api/urls/{id}/results [get]
func (h *URLHandler) Results(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	dto, err := h.urlService.Results(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
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
}
