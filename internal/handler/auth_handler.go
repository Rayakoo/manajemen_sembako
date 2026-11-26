package handler

import (
	"errors"
	"net/http"
	"net/url"

	"ayam-batu-api/internal/middleware"
	"ayam-batu-api/internal/repository"
	"ayam-batu-api/internal/response"
	"ayam-batu-api/internal/service"

	"github.com/gin-gonic/gin"
)

const oauthStateCookieName = "oauth_state"

// AuthHandler exposes local and Google OAuth authentication endpoints.
type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) SignUp(c *gin.Context) {
	var input service.SignUpInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.authService.SignUp(input)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyUsed) {
			response.Error(c, http.StatusConflict, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to create account")
		return
	}

	response.JSON(c, http.StatusCreated, "account created successfully", result)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input service.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.authService.Login(input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			response.Error(c, http.StatusUnauthorized, err.Error())
		case errors.Is(err, service.ErrGoogleAccountOnly):
			response.Error(c, http.StatusBadRequest, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "failed to login")
		}
		return
	}

	response.JSON(c, http.StatusOK, "login successful", result)
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	if !h.authService.IsGoogleOAuthConfigured() {
		response.Error(c, http.StatusServiceUnavailable, service.ErrGoogleOAuthNotEnabled.Error())
		return
	}

	state, err := h.authService.GenerateOAuthState()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to generate oauth state")
		return
	}

	authURL, err := h.authService.GoogleAuthURL(state)
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, err.Error())
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(oauthStateCookieName, state, 600, "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	if errParam := c.Query("error"); errParam != "" {
		response.Error(c, http.StatusBadRequest, "google authentication failed: "+errParam)
		return
	}

	cookieState, _ := c.Cookie(oauthStateCookieName)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(oauthStateCookieName, "", -1, "/", "", false, true)

	state := c.Query("state")
	if err := h.authService.ValidateOAuthState(state, cookieState); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	code := c.Query("code")
	if code == "" {
		response.Error(c, http.StatusBadRequest, "authorization code is required")
		return
	}

	result, frontendRedirectURL, err := h.authService.HandleGoogleCallback(c.Request.Context(), code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGoogleOAuthNotEnabled):
			response.Error(c, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, service.ErrGoogleEmailUnverified):
			response.Error(c, http.StatusBadRequest, err.Error())
		default:
			response.Error(c, http.StatusBadGateway, "failed to complete google login")
		}
		return
	}

	if frontendRedirectURL != "" {
		target, err := url.Parse(frontendRedirectURL)
		if err == nil {
			q := target.Query()
			q.Set("token", result.Token)
			target.RawQuery = q.Encode()
			c.Redirect(http.StatusFound, target.String())
			return
		}
	}

	response.JSON(c, http.StatusOK, "google login successful", result)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.authService.GetProfile(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "user not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to load profile")
		return
	}

	response.JSON(c, http.StatusOK, "profile retrieved", user)
}
