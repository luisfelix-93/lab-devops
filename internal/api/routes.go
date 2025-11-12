package api

import (
	"github.com/labstack/echo/v4"
)

// RegisterRoutes regista todas as rotas da API.
func RegisterRoutes(e *echo.Echo, h *Handler) {
	// Agrupa as rotas sob /api/v1
	g := e.Group("/api/v1")
	
	// Rota para buscar os detalhes de um Lab (HTTP GET)
	// ex: GET /api/v1/labs/lab-tf-01
	g.GET("/labs/:labID", h.HandleGetLabDetails)
	
	// Rota para executar um Lab (WebSocket)
	// ex: WS /api/v1/labs/lab-tf-01/execute
	g.GET("/labs/:labID/execute", h.HandlerLabExecute)
	
	// TODO: Adicionar uma rota para listar todos os labs
	// g.GET("/labs", h.HandleListLabs)
}