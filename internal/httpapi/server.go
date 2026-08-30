package httpapi

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "isitdown/docs"
	"isitdown/internal/store"
)

type Server struct {
	store *store.Store
	mux   *http.ServeMux
}

func NewServer(s *store.Store) *Server {
	srv := &Server{store: s, mux: http.NewServeMux()}
	srv.routes()
	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /status", s.handleStatus)
	s.mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	s.mux.HandleFunc("POST /monitors", s.handleCreateMonitor)
	s.mux.HandleFunc("GET /monitors", s.handleListMonitors)
	s.mux.HandleFunc("GET /monitors/{id}", s.handleGetMonitor)
	s.mux.HandleFunc("PUT /monitors/{id}", s.handleUpdateMonitor)
	s.mux.HandleFunc("DELETE /monitors/{id}", s.handleDeleteMonitor)
	s.mux.HandleFunc("GET /monitors/{id}/checks", s.handleListChecks)
}
