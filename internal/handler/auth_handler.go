package handler

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fuzumoe/linkTorch-api/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
	userService service.UserService
}

func NewAuthHandler(authService service.AuthService, userService service.UserService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginBasic godoc
// @Summary      Login via Basic Auth header and generate JWT token
// @Description  Authenticates a user using Basic Authorization header and returns a JWT token
// @Description  Requires "Authorization: Basic base64(email:password)" header
// @Tags         auth
// @Produce      json
// @Param        Authorization header string true "Basic base64(email:password)"
// @Success      200 {object} map[string]interface{} "JWT token generated"
// @Failure      400 {object} map[string]interface{} "Invalid request or login error"
// @Failure      401 {object} map[string]interface{} "Authentication failed"
// @Router       /login/basic [post]
func (h *AuthHandler) LoginBasic(c *gin.Context) {
	const prefix = "Basic "
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, prefix) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization header missing or invalid"})
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(authHeader[len(prefix):])
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 encoding in authorization header"})
		return
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid basic auth format"})
		return
	}
	email, password := parts[0], parts[1]

	userDTO, err := h.userService.Authenticate(email, password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authentication failed"})
		return
	}

	token, err := h.authService.Generate(userDTO.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// LoginJWT godoc
// @Summary      Login via JSON payload and generate JWT token
// @Description  Authenticates a user using email and password provided in JSON and returns a JWT token
// @Description  Example request: {"email": "user@example.com", "password": "userpassword"}
// @Description  Example response: {"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        loginRequest  body      LoginRequest  true  "Login request payload"
// @Success      200           {object}  map[string]interface{} "JWT token generated"
// @Failure      400           {object}  map[string]interface{} "Invalid request or login error"
// @Failure      401           {object}  map[string]interface{} "Authentication failed"
// @Router       /login/jwt [post]
func (h *AuthHandler) LoginJWT(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid login request"})
		return
	}

	userDTO, err := h.userService.Authenticate(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authentication failed"})
		return
	}

	token, err := h.authService.Generate(userDTO.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// Logout godoc
// @Summary      Logout and invalidate JWT token
// @Description  Invalidates the current JWT token so it can no longer be used
// @Tags         auth
// @Produce      json
// @Success      200 {object} map[string]interface{} "Logout message"
// @Failure      400 {object} map[string]interface{} "Invalid token or request"
// @Security     JWTAuth
// @Security     BasicAuth
// @Router       /logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization header missing"})
		return
	}

	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenString := authHeader[len("Bearer "):]
		claims, err := h.authService.Validate(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		err = h.authService.Invalidate(claims.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
		return
	}

	if strings.HasPrefix(authHeader, "Basic ") {
		decoded, err := base64.StdEncoding.DecodeString(authHeader[len("Basic "):])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 encoding in authorization header"})
			return
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid basic auth format"})
			return
		}
		email, password := parts[0], parts[1]

		_, err = h.userService.Authenticate(email, password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported authorization type"})
}

func (h *AuthHandler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/login/basic", h.LoginBasic)
	rg.POST("/login/jwt", h.LoginJWT)
}

func (h *AuthHandler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	rg.POST("/logout", h.Logout)
}
