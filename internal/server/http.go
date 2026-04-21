package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"ringg/internal/cache"
	"ringg/internal/membership"
)

const forwardedByHeader = "X-Ringg-Forwarded-By"

type Handler struct {
	store      *cache.Store
	mux        *http.ServeMux
	membership *membership.State
	client     *http.Client
}

type cacheValueResponse struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	NodeID string `json:"node_id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type Options struct {
	Membership *membership.State
	Client     *http.Client
}

func NewHandler(store *cache.Store, options Options) *Handler {
	client := options.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}

	h := &Handler{
		store:      store,
		mux:        http.NewServeMux(),
		membership: options.Membership,
		client:     client,
	}

	h.routes()

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) routes() {
	h.mux.HandleFunc("GET /healthz", h.handleHealthz)
	h.mux.HandleFunc("GET /cluster", h.handleMembers)
	h.mux.HandleFunc("GET /members", h.handleMembers)
	h.mux.HandleFunc("POST /members/join", h.handleJoin)
	h.mux.HandleFunc("POST /members/leave", h.handleLeave)
	h.mux.HandleFunc("GET /cache/{key}", h.handleGet)
	h.mux.HandleFunc("PUT /cache/{key}", h.handlePut)
	h.mux.HandleFunc("DELETE /cache/{key}", h.handleDelete)
}

func (h *Handler) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleMembers(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.membership.Snapshot())
}

func (h *Handler) handleJoin(w http.ResponseWriter, r *http.Request) {
	var request membership.Member
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "failed to decode join request")
		return
	}

	request.Status = membership.StatusAlive
	if err := h.membership.Upsert(request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.membership.Snapshot())
}

func (h *Handler) handleLeave(w http.ResponseWriter, _ *http.Request) {
	if err := h.membership.MarkLeft(h.membership.LocalNodeID()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark local node as left")
		return
	}

	writeJSON(w, http.StatusOK, h.membership.Snapshot())
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	if h.forwardToOwnerIfNeeded(w, r, key) {
		return
	}

	value, err := h.store.Get(key)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			writeError(w, http.StatusNotFound, "key not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "failed to fetch value")
		return
	}

	writeJSON(w, http.StatusOK, cacheValueResponse{
		Key:    key,
		Value:  value,
		NodeID: h.membership.LocalNodeID(),
	})
}

func (h *Handler) handlePut(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	if h.forwardToOwnerIfNeeded(w, r, key) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	value := string(body)
	h.store.Set(key, value)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	if h.forwardToOwnerIfNeeded(w, r, key) {
		return
	}

	h.store.Delete(key)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) forwardToOwnerIfNeeded(w http.ResponseWriter, r *http.Request, key string) bool {
	owner, err := h.membership.GetOwner(key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve key owner")
		return true
	}

	if owner.NodeID == h.membership.LocalNodeID() {
		return false
	}

	if r.Header.Get(forwardedByHeader) != "" {
		writeError(w, http.StatusBadGateway, "request was forwarded but still did not reach the owner; cluster config may be inconsistent")
		return true
	}

	if err := h.forwardRequest(w, r, owner.Addr); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return true
	}

	return true
}

func (h *Handler) forwardRequest(w http.ResponseWriter, r *http.Request, targetAddr string) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.New("failed to read request body for forwarding")
	}

	targetURL := strings.TrimRight(targetAddr, "/") + r.URL.Path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	forwardedRequest, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		return errors.New("failed to create forwarded request")
	}

	copyHeaders(forwardedRequest.Header, r.Header)
	forwardedRequest.Header.Set(forwardedByHeader, h.membership.LocalNodeID())

	response, err := h.client.Do(forwardedRequest)
	if err != nil {
		return errors.New("failed to forward request to owner node")
	}
	defer response.Body.Close()

	copyHeaders(w.Header(), response.Header)
	w.WriteHeader(response.StatusCode)

	if _, err := io.Copy(w, response.Body); err != nil {
		return errors.New("failed to copy owner response")
	}

	return nil
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
