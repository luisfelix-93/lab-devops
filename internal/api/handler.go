package api

import (
	"lab-devops/internal/domain"
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
	Title          string `json:"title"`
	Type           string `json:"type"`
	Instructions   string `json:"instructions"`
	InitialCode    string `json:"initial_code"`
	TrackID        string `json:"track_id"`
	LabOrder       int    `json:"lab_order"`
	ValidationCode string `json:"validation_code"`
}

type CreateTrackRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
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

	ctx := c.Request().Context()

	// Variáveis para capturar o retorno do serviço
	var logStream <-chan service.ExecutionResult
	var finalState <-chan service.ExecutionFinalState
	var wsID string
	var errExec error
	var isValidation bool
	var shouldValidateAfter bool // Flag para indicar se deve validar após execução

	// Lógica de Decisão: Executar ou Validar?
	switch msg.Action {
	case "execute":
		log.Printf("INFO [Handler]: Executando comando do usuário (Lab %s)", labID)
		isValidation = false
		shouldValidateAfter = true // Define que após sucesso, deve rodar validação
		// Chama ExecuteLab (retorna 4 valores)
		logStream, finalState, wsID, errExec = h.labService.ExecuteLab(ctx, labID, msg.UserCode)

	case "validate":
		log.Printf("INFO [Handler]: Validando solução (Lab %s)", labID)
		isValidation = true
		// Chama ValidateLab (retorna 4 valores)
		logStream, finalState, wsID, errExec = h.labService.ValidateLab(ctx, labID)

	default:
		log.Printf("AVISO [Handler]: Ação desconhecida: %s", msg.Action)
		return nil
	}

	if errExec != nil {
		log.Printf("ERRO [Handler]: Falha ao iniciar execução: %v", errExec)
		ws.WriteJSON(ServerMessage{Type: "error", Payload: errExec.Error()})
		return errExec
	}

	// Loop de Streaming (Goroutine para não bloquear o WS)
	go func() {
		for {
			select {
			// Caso A: Nova linha de log do executor
			case logLine, ok := <-logStream:
				if !ok {
					logStream = nil // Canal fechado
					continue
				}
				if err := ws.WriteJSON(ServerMessage{Type: "log", Payload: logLine.Line}); err != nil {
					log.Printf("AVISO [Handler]: Erro ao escrever log no ws: %v", err)
					return
				}

			// Caso B: A execução terminou (Estado Final)
			case state, ok := <-finalState:
				if !ok {
					return
				}

				// Se houve erro na execução (Exit Code != 0)
				if state.Error != nil {
					log.Printf("INFO [Handler]: Execução falhou: %v", state.Error)
					ws.WriteJSON(ServerMessage{Type: "error", Payload: state.Error.Error()})

					// Feedback específico se for validação
					if isValidation {
						ws.WriteJSON(ServerMessage{Type: "log", Payload: "❌ A validação falhou. Verifique a sua solução e tente novamente."})
					}
					return
				}

				// SUCESSO (Exit Code 0)

				// Se precisava validar depois de executar e não estamos já validando:
				if shouldValidateAfter {
					log.Printf("INFO [Handler]: Execução ok. Iniciando validação automática.")
					ws.WriteJSON(ServerMessage{Type: "log", Payload: "\n✅ Execução concluída com sucesso. Iniciando validação...\n"})

					// Inicia a Validação
					logStream, finalState, wsID, errExec = h.labService.ValidateLab(ctx, labID)
					if errExec != nil {
						log.Printf("ERRO [Handler]: Falha ao iniciar validação automática: %v", errExec)
						ws.WriteJSON(ServerMessage{Type: "error", Payload: errExec.Error()})
						return
					}

					// Atualiza flags
					shouldValidateAfter = false
					isValidation = true

					// Reinicia o loop com os novos canais
					continue
				}

				log.Printf("INFO [Handler]: Execução concluída com sucesso.")

				if isValidation {
					// Se foi Validação e passou -> Marca como COMPLETED
					log.Printf("INFO [Handler]: Lab validado! Salvando status completed.")
					if err := h.labService.SaveWorkspaceStatus(ctx, wsID, domain.WorkspaceStatusCompleted); err != nil {
						log.Printf("ERRO [Handler]: Falha ao salvar status: %v", err)
					}
					ws.WriteJSON(ServerMessage{Type: "complete", Payload: "✅ Parabéns! Laboratório concluído com sucesso."})
				} else {
					// Se foi apenas Execute -> Apenas avisa que terminou
					ws.WriteJSON(ServerMessage{Type: "complete", Payload: "Comando executado."})
				}

				// Salva o estado do Terraform (se houver)
				if state.NewState != nil {
					h.labService.SaveWorkspaceState(ctx, wsID, state.NewState)
				}
				return // Fim da execução

			// Caso C: Cancelamento do contexto HTTP
			case <-ctx.Done():
				log.Printf("AVISO [Handler]: Contexto cancelado.")
				return
			}
		}
	}()

	// Manter a conexão WebSocket viva até o cliente desconectar
	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			log.Printf("INFO [Handler]: Cliente desconectado: %v", err)
			break
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

func (h *Handler) HandleCreateLab(c echo.Context) error {
	var req CreateLabRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Payload inválido"})
	}

	lab, err := h.labService.CreateLab(
		c.Request().Context(),
		req.Title, req.Type, req.Instructions, req.InitialCode,
		req.TrackID, req.LabOrder, req.ValidationCode,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, lab)
}
func (h *Handler) HandlerDeleteLab(c echo.Context) error {
	labId := c.Param("labId")
	err := h.labService.CleanLab(c.Request().Context(), labId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Lab deletado com sucesso"})
}

func (h *Handler) HandleCreateTrack(c echo.Context) error {
	var req CreateTrackRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Payload inválido"})
	}

	track, err := h.labService.CreateTrack(c.Request().Context(), req.Title, req.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, track)
}

func (h *Handler) HandleListTracks(c echo.Context) error {
	tracks, err := h.labService.ListTracks(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, tracks)
}

func (h *Handler) HandleUpdateLab(c echo.Context) error {
	var req CreateLabRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Payload inválido"})
	}

	labId := c.Param("labId")
	lab, err := h.labService.UpdateLab(c.Request().Context(), labId, req.Title, req.Type, req.Instructions, req.InitialCode, req.TrackID, req.LabOrder, req.ValidationCode)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, lab)
}

func (h *Handler) HandleUpdateTrack(c echo.Context) error {
	var req CreateTrackRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Payload inválido"})
	}

	trackId := c.Param("trackId")
	track, err := h.labService.UpdateTrack(c.Request().Context(), trackId, req.Title, req.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, track)
}

func (h *Handler) HandleDeleteLab(c echo.Context) error {
	labId := c.Param("labId")
	err := h.labService.DeleteLab(c.Request().Context(), labId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Lab deletado com sucesso"})
}

func (h *Handler) HandleDeleteTrack(c echo.Context) error {
	trackId := c.Param("trackId")
	err := h.labService.DeleteTrack(c.Request().Context(), trackId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Track deletado com sucesso"})
}
