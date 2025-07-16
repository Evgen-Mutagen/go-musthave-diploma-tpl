package controller

import (
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/service"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"go.uber.org/zap"
)

type AuthController struct {
	authService service.AuthService
	logger      *zap.Logger
}

func NewAuthController(authService service.AuthService, logger *zap.Logger) *AuthController {
	return &AuthController{
		authService: authService,
		logger:      logger,
	}
}

func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := render.DecodeJSON(r.Body, &request); err != nil {
		c.logger.Debug("Invalid request format", zap.Error(err))
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	user, token, err := c.authService.Register(r.Context(), request.Login, request.Password)
	if err != nil {
		c.logger.Warn("Registration failed",
			zap.String("login", request.Login),
			zap.Error(err))

		switch err {
		case service.ErrUserAlreadyExists:
			http.Error(w, "Login already exists", http.StatusConflict)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	c.logger.Info("User registered successfully",
		zap.Int64("user_id", user.ID),
		zap.String("login", user.Login))

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusOK)
}

func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := render.DecodeJSON(r.Body, &request); err != nil {
		c.logger.Debug("Invalid request format", zap.Error(err))
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	user, token, err := c.authService.Login(r.Context(), request.Login, request.Password)
	if err != nil {
		c.logger.Warn("Login failed",
			zap.String("login", request.Login),
			zap.Error(err))

		switch err {
		case service.ErrInvalidCredentials:
			http.Error(w, "Invalid login or password", http.StatusUnauthorized)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	c.logger.Info("User logged in successfully",
		zap.Int64("user_id", user.ID),
		zap.String("login", user.Login))

	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusOK)
}
