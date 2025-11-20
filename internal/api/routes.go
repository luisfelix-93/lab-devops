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

	
	
	// Rota para listar todos os labs
	g.GET("/labs", h.HandleListLabs)

	// Rota para criar um laboratório
	g.POST("/labs", h.HandleCreateLab)

	// Rota para deletar um laboratório
	g.DELETE("/labs/:labId", h.HandlerDeleteLab)

	g.GET("/tracks", h.HandleListTracks)
    // Rota para criar uma nova Trilha
    g.POST("/tracks", h.HandleCreateTrack)
}