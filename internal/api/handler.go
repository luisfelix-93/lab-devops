package api

import (
	"lab-devops/internal/service"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type Handler struct {
	labService *service.LabService
}

func NewHandler(svc *service.LabService) *Handler {
	return &Handler{
		labService: svc,
	}
}

type ClientMessage struct {
	Action   string `json:"action"`
	UserCode string `json:"user_code"`
}

type ServerMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload,omitempty"`
}

type CreateLabRequest struct {
    Title        string `json:"title"`
    Type         string `json:"type"`
    Instructions string `json:"instructions"`
    InitialCode  string `json:"initial_code"`
}

func (h *Handler) HandlerLabExecute(c echo.Context) error {
	labID := c.Param("labID")
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("ERRO [Handler]: Falha no upgrade do websocket: %v", err)
		return err
	}
	
	defer ws.Close()
	log.Printf("INFO [Handler]: Cliente WebSocket conectado para Lab %s", labID)

	var msg ClientMessage
	if err := ws.ReadJSON(&msg); err != nil {
		log.Printf("AVISO [Handler]: Erro ao ler msg inicial: %v", err)
		return err
	}

	if msg.Action != "execute" {
		log.Printf("AVISO [Handler]: Ação desconhecida: %s", msg.Action)
		return nil
	}

	log.Printf("INFO [Handler]: Recebido comando 'execute' para Lab %s", labID)

	ctx := c.Request().Context()
	logStream, finalState, err := h.labService.ExecuteLab(ctx, labID, msg.UserCode)
	if err != nil {
		log.Printf("ERRO [Handler]: Falha ao chamar ExecuteLab: %v", err)
		ws.WriteJSON(ServerMessage{Type: "error", Payload: err.Error()})
		return err
	}

	go func() {
		for {
			select {
			// Caso A: Chegou uma nova linha de log
			case logLine, ok := <-logStream:
				if !ok {
					// Canal de log foi fechado, mas finalState ainda pode vir
					logStream = nil // Evita selecionar este case novamente
					continue
				}
				// Envia o log para o cliente
				if err := ws.WriteJSON(ServerMessage{Type: "log", Payload: logLine.Line}); err != nil {
					log.Printf("AVISO [Handler]: Erro ao escrever log no ws: %v", err)
					return
				}

			// Caso B: A execução terminou (com sucesso ou erro)
			case state, ok := <-finalState:
				if !ok {
					// Canal final fechado, terminamos
					return
				}
				
				// Se a execução falhou, envia o erro
				if state.Error != nil {
					log.Printf("INFO [Handler]: Execução falhou: %v", state.Error)
					ws.WriteJSON(ServerMessage{Type: "error", Payload: state.Error.Error()})
					// Não salvamos o estado se deu erro
					return
				}
				//Sucesso! (Exit Code 0)
				// Se a execução foi bem-sucedida, salva o novo estado
				log.Printf("INFO [Handler]: Execução concluída, salvando estado para Workspace %s...", state.WorkspaceID)
				if err := h.labService.SaveWorkspaceState(ctx, state.WorkspaceID, state.NewState); err != nil {
					log.Printf("ERRO [Handler]: Falha ao salvar estado do workspace: %v", err)
					// Mesmo que salvar o estado falhe, a execução em si foi um sucesso.
					// Poderíamos decidir enviar um erro aqui, mas por enquanto vamos notificar o sucesso.
					ws.WriteJSON(ServerMessage{Type: "error", Payload: "Falha ao salvar o estado final da execução."})
					return
				}

				if err := h.labService.SaveWorkspaceStatus(ctx, state.WorkspaceID, "complete"); err != nil {
					log.Printf("ERRO [Handler]: Falha ao salvar status do workspace: %v", err)
					return
				}
				
				ws.WriteJSON(ServerMessage{Type: "complete", Payload: "Execução concluída com sucesso!"})
				return // Termina a goroutine

			// Caso C: O contexto do request foi cancelado
			case <-ctx.Done():
				log.Printf("AVISO [Handler]: Contexto cancelado, fechando stream.")
				return
			}
		}
	}()

	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			log.Printf("INFO [Handler]: Cliente desconectado: %v", err)
			break // Sai do loop e encerra a função
		}
	}

	return nil

}

func (h *Handler) HandleGetLabDetails(c echo.Context) error {
	labID := c.Param("labID")
	
	// Chama o serviço
	lab, ws, err := h.labService.GetLabDetails(c.Request().Context(), labID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	
	// Retorna uma resposta combinada
	response := struct {
		Lab       interface{} `json:"lab"`
		Workspace interface{} `json:"workspace"`
	}{
		Lab:       lab,
		Workspace: ws,
	}
	
	return c.JSON(http.StatusOK, response)
}

func (h *Handler) HandleListLabs(c echo.Context) error {
	labs, err := h.labService.ListLabs(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, labs)
}

func (h *Handler) HandlerCreateLab(c echo.Context) error {
	var req CreateLabRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Payload inválido"}) 
	}

	lab, err := h.labService.CreateLab(c.Request().Context(), req.Title, req.Type, req.Instructions, req.InitialCode)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }

    return c.JSON(http.StatusCreated, lab)
}

func (h * Handler) HandlerDeleteLab(c echo.Context) error {
	labId := c.Param("labId")
	err := h.labService.CleanLab(c.Request().Context(), labId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Lab deletado com sucesso"})
}