package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Radiqeee/taskapi/internal/auth"
	"github.com/Radiqeee/taskapi/internal/store"
)

type credentials struct {
	Email string `json:"email" binding:"required,email"`
	// bcrypt ignores everything past 72 bytes, so reject longer passwords.
	Password string `json:"password" binding:"required,min=8,max=72"`
}

func (h *Handler) register(c *gin.Context) {
	var req credentials
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		slog.Error("hash password", "err", err)
		writeError(c, http.StatusInternalServerError, "internal error")
		return
	}

	user, err := h.store.CreateUser(c.Request.Context(), strings.ToLower(req.Email), hash)
	if errors.Is(err, store.ErrConflict) {
		writeError(c, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		writeStoreError(c, err)
		return
	}

	token, err := h.tokens.Issue(user.ID)
	if err != nil {
		slog.Error("issue token", "err", err)
		writeError(c, http.StatusInternalServerError, "internal error")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"token": token, "user": user})
}

func (h *Handler) login(c *gin.Context) {
	var req credentials
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	user, err := h.store.GetUserByEmail(c.Request.Context(), strings.ToLower(req.Email))
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeStoreError(c, err)
		return
	}
	// Same response for unknown email and wrong password, so emails can't be enumerated.
	if err != nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(c, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := h.tokens.Issue(user.ID)
	if err != nil {
		slog.Error("issue token", "err", err)
		writeError(c, http.StatusInternalServerError, "internal error")
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}
