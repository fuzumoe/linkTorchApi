package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
	"github.com/fuzumoe/linkTorch-api/internal/service"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) parseUintParam(c *gin.Context, name string) (uint, bool) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(v), true
}

func (h *UserHandler) paginationFromQuery(c *gin.Context) repository.Pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	return repository.Pagination{Page: page, PageSize: size}
}

// @Summary Create User
// @Tags    users
// @Accept  json
// @Produce json
// @Param   input body model.CreateUserInput true "User to create"
// @Success 201 {object} map[string]uint "{id}"
// @Failure 400 {object} map[string]string "error"
// @Failure 500 {object} map[string]string "error"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var input model.CreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	userID, err := h.userService.Register(&input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": userID})
}

// @Summary Get Authenticated User
// @Tags    users
// @Produce json
// @Success 200 {object} model.UserDTO
// @Failure 401 {object} map[string]string "error"
// @Failure 404 {object} map[string]string "error"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /users/me [get]
func (h *UserHandler) Me(c *gin.Context) {
	uidAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uidAny.(uint)

	user, err := h.userService.Get(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// @Summary Search Users
// @Tags    users
// @Produce json
// @Param   q query string true "Search query"
// @Param   sort query string false "Sort by field"
// @Param   filter query string false "Filter by field"
// @Success 200 {object} model.PaginatedResponse[model.UserDTO] "Paginated User list"
// @Failure 400 {object} map[string]string "error"
// @Failure 500 {object} map[string]string "error"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /users/search [get]
func (h *UserHandler) Get(c *gin.Context) {
	uRoleAny, exists := c.Get("user_role")
	if !exists || uRoleAny != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admins can search users"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter is required"})
		return
	}

	sort := c.DefaultQuery("sort", "")
	filter := c.DefaultQuery("filter", "")
	paginatedResult, err := h.userService.Search(query, sort, filter, h.paginationFromQuery(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, paginatedResult)
}

// @Summary Update User
// @Tags    users
// @Accept  json
// @Produce json
// @Param   id   path uint true "User ID"
// @Param   input body model.UpdateUserInput true "User to update"
// @Success 200 {object} model.UserDTO
// @Failure 400 {object} map[string]string "error"
// @Failure 404 {object} map[string]string "error"
// @Failure 500 {object} map[string]string "error"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}

	var input model.UpdateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	uRoleAny, roleExists := c.Get("user_role")
	uidAny, uidExists := c.Get("user_id")

	if !roleExists || !uidExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userRole := uRoleAny.(string)
	userID := uidAny.(uint)

	if userRole != "admin" {
		if userID != id {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot update other users"})
			return
		}
		if input.Role != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "only admins can update user roles"})
			return
		}
	}

	user, err := h.userService.Update(id, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// @Summary Delete User
// @Tags    users
// @Produce json
// @Param   id path uint true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error"
// @Failure 404 {object} map[string]string "error"
// @Failure 500 {object} map[string]string "error"
// @Security JWTAuth
// @Security BasicAuth
// @Router  /users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	uRoleAny, exists := c.Get("user_role")
	if !exists || uRoleAny != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admins can delete users"})
		return
	}

	id, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}

	err := h.userService.Delete(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *UserHandler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	rg.POST("/users", h.Create)
	rg.GET("/users/me", h.Me)
	rg.GET("/users/search", h.Get)
	rg.GET("/users/:id", h.Get)
	rg.PUT("/users/:id", h.Update)
	rg.DELETE("/users/:id", h.Delete)
}
