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

				// Se a execução foi bem-sucedida, salva o novo estado
				log.Printf("INFO [Handler]: Execução concluída, salvando estado...")
				// TODO: Precisamos do WorkspaceID aqui!
				// (Vamos precisar de um ajuste no service ou no handler)
				// Por agora, apenas notificamos o sucesso.
				// h.labService.SaveWorkspaceState(ctx, wsID, state.NewState)
				
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