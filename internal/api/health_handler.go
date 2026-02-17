package api

import (
	"lab-devops/internal/service"
	"net/http"

	"github.com/labstack/echo/v4"
)

// HandleHealthCheck verifica o estado da aplicação e dependências
// GET /api/v1/health
func (h *Handler) HandleHealthCheck(c echo.Context) error {
	ctx := c.Request().Context()
	health := h.healthService.CheckHealth(ctx)

	httpStatus := http.StatusOK
	if health.Status == service.StatusUnavailable {
		httpStatus = http.StatusServiceUnavailable
	}

	return c.JSON(httpStatus, health)
}
