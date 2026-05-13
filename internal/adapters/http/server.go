package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/ports"
	"github.com/super-duper-bassoon/internal/core/services"
)

type Deps struct {
	API           *services.APIHandlerService
	HealthRepo    ports.HealthRepository
	NATSConnected func() bool
	DBHealthy     func() bool
	Logger        *log.Logger
}

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	h := &handlers{deps: deps}

	r.Post("/workflows", h.createWorkflow)
	r.Get("/workflows", h.listWorkflows)
	r.Get("/workflows/{id}", h.getWorkflow)
	r.Put("/workflows/{id}", h.editWorkflow)
	r.Delete("/workflows/{id}", h.deleteWorkflow)
	r.Post("/workflows/{id}/trigger", h.triggerWorkflow)
	r.Post("/workflows/{id}/activate", h.activateWorkflow)
	r.Post("/workflows/{id}/deactivate", h.deactivateWorkflow)
	r.Get("/workflows/{id}/runs", h.listRuns)

	r.Get("/clients", h.listClients)
	r.Get("/clients/{id}", h.getClient)

	r.Get("/runs/{id}", h.getRun)
	r.Get("/runs/{id}/results", h.getRunResults)

	r.Get("/health", h.listAllHealth)
	r.Get("/health/{workflow_type}", h.getHealth)
	r.Get("/health/liveness", h.liveness)
	r.Get("/health/readiness", h.readiness)

	r.Get("/bans", h.listBans)
	r.Get("/bans/{client_id}", h.getClientBans)
	r.Put("/bans/{client_id}/unban", h.unbanClient)

	r.Get("/circuits", h.listCircuitStates)
	r.Get("/circuits/{workflow_id}", h.getCircuitState)

	r.Get("/status", h.systemStatus)

	return r
}

type handlers struct {
	deps Deps
}

func (h *handlers) writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		if h.deps.Logger != nil {
			h.deps.Logger.Printf("write json error: %v", err)
		}
	}
}

func (h *handlers) writeError(w http.ResponseWriter, err error) {
	apiErr := toAPIError(err)
	h.writeJSON(w, apiErr.GetHTTPStatus(), apiErr)
}

func toAPIError(err error) *domain.APIError {
	if err == nil {
		return domain.NewAPIError("INTERNAL", "unknown error")
	}
	var apiErr *domain.APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	switch {
	case errors.Is(err, domain.ErrWorkflowNotFound), errors.Is(err, domain.ErrRunNotFound),
		errors.Is(err, domain.ErrClientNotFound), errors.Is(err, domain.ErrBanNotFound),
		errors.Is(err, domain.ErrCircuitStateNotFound):
		return domain.NewAPIError("NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrInvalidFilter), errors.Is(err, domain.ErrInvalidWorkflow),
		errors.Is(err, domain.ErrMissingRequiredField), errors.Is(err, domain.ErrInvalidRequest),
		errors.Is(err, domain.ErrInvalidPolicy):
		return domain.NewAPIError("VALIDATION_ERROR", err.Error())
	case errors.Is(err, domain.ErrWorkflowInactive):
		return domain.NewAPIError("CONFLICT", err.Error())
	}
	return domain.NewAPIError("INTERNAL", err.Error())
}

func (h *handlers) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, domain.NewAPIError("BAD_REQUEST", "invalid JSON"))
		return
	}
	wf, err := h.deps.API.CreateWorkflow(r.Context(), &req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, wf)
}

func (h *handlers) listWorkflows(w http.ResponseWriter, r *http.Request) {
	wfs, err := h.deps.API.ListWorkflows(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"items": wfs, "total": len(wfs)})
}

func (h *handlers) getWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wf, err := h.deps.API.GetWorkflow(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, wf)
}

func (h *handlers) editWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req domain.EditWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, domain.NewAPIError("BAD_REQUEST", "invalid JSON"))
		return
	}
	wf, err := h.deps.API.EditWorkflow(r.Context(), id, &req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, wf)
}

func (h *handlers) deleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.deps.API.DeleteWorkflow(r.Context(), id); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) triggerWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req domain.TriggerWorkflowRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	run, err := h.deps.API.TriggerWorkflow(r.Context(), id, req.Reason)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, run)
}

func (h *handlers) activateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wf, err := h.deps.API.ActivateWorkflow(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, wf)
}

func (h *handlers) deactivateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wf, err := h.deps.API.DeactivateWorkflow(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, wf)
}

func (h *handlers) listRuns(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	runs, err := h.deps.API.ListRuns(r.Context(), id, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"items": runs, "total": len(runs)})
}

func (h *handlers) listClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.deps.API.ListClients(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"items": clients, "total": len(clients)})
}

func (h *handlers) getClient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.deps.API.GetClient(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, c)
}

func (h *handlers) getRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	run, err := h.deps.API.GetRun(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, run)
}

func (h *handlers) getRunResults(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	results, err := h.deps.API.GetRunResults(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"items": results, "total": len(results)})
}

func (h *handlers) listAllHealth(w http.ResponseWriter, r *http.Request) {
	healths, err := h.deps.HealthRepo.ListAllWorkflowTypeHealths(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"items": healths, "total": len(healths)})
}

func (h *handlers) getHealth(w http.ResponseWriter, r *http.Request) {
	wt := chi.URLParam(r, "workflow_type")
	hh, err := h.deps.API.GetHealth(r.Context(), wt)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, hh)
}

func (h *handlers) liveness(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (h *handlers) readiness(w http.ResponseWriter, r *http.Request) {
	dbOk := h.deps.DBHealthy == nil || h.deps.DBHealthy()
	natsOk := h.deps.NATSConnected == nil || h.deps.NATSConnected()
	if !dbOk || !natsOk {
		h.writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"db": dbOk, "nats": natsOk})
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *handlers) listBans(w http.ResponseWriter, r *http.Request) {
	bans, err := h.deps.API.ListAllBans(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"items": bans, "total": len(bans)})
}

func (h *handlers) getClientBans(w http.ResponseWriter, r *http.Request) {
	cid := chi.URLParam(r, "client_id")
	bans, err := h.deps.API.GetBans(r.Context(), cid)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"items": bans, "total": len(bans)})
}

func (h *handlers) unbanClient(w http.ResponseWriter, r *http.Request) {
	cid := chi.URLParam(r, "client_id")
	var req domain.UnbanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, domain.NewAPIError("BAD_REQUEST", "invalid JSON"))
		return
	}
	if err := h.deps.API.UnbanClient(r.Context(), cid, &req); err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "client_id": cid})
}

func (h *handlers) listCircuitStates(w http.ResponseWriter, r *http.Request) {
	states, err := h.deps.API.ListCircuitStates(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"items": states, "total": len(states)})
}

func (h *handlers) getCircuitState(w http.ResponseWriter, r *http.Request) {
	wfid := chi.URLParam(r, "workflow_id")
	s, err := h.deps.API.GetCircuitState(r.Context(), wfid)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, s)
}

func (h *handlers) systemStatus(w http.ResponseWriter, r *http.Request) {
	dbOk := h.deps.DBHealthy == nil || h.deps.DBHealthy()
	natsOk := h.deps.NATSConnected == nil || h.deps.NATSConnected()
	s := h.deps.API.SystemStatus(r.Context(), natsOk, dbOk)
	h.writeJSON(w, http.StatusOK, s)
}
