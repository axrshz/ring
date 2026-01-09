// protocol/http.go
package protocol

import (
	"encoding/json"
	"io"
	"net/http"
	"ring/cache"
	"time"
)

type Server struct {
    cache *cache.Cache
    addr  string
}

func NewServer(addr string, c *cache.Cache) *Server {
    return &Server{
        cache: c,
        addr:  addr,
    }
}

type SetRequest struct {
    Key   string `json:"key"`
    Value string `json:"value"`
    TTL   int    `json:"ttl"` // seconds
}

func (s *Server) Start() error {
    mux := http.NewServeMux()
    mux.HandleFunc("/get", s.handleGet)
    mux.HandleFunc("/set", s.handleSet)
    mux.HandleFunc("/delete", s.handleDelete)

    return http.ListenAndServe(s.addr, mux)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
    key := r.URL.Query().Get("key")
    if key == "" {
        http.Error(w, "key required", http.StatusBadRequest)
        return
    }

    value, found := s.cache.Get(key)
    if !found {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }

    w.Write(value)
}

func (s *Server) handleSet(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    var req SetRequest
    if err := json.Unmarshal(body, &req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    ttl := time.Duration(req.TTL) * time.Second
    s.cache.Set(req.Key, []byte(req.Value), ttl)
    w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
    key := r.URL.Query().Get("key")
    if key == "" {
        http.Error(w, "key required", http.StatusBadRequest)
        return
    }

    s.cache.Delete(key)
    w.WriteHeader(http.StatusOK)
}