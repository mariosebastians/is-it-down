package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "isitdown/docs"
	"isitdown/internal/store"
)

type Server struct {
	store  *store.Store
	router chi.Router
}

func NewServer(s *store.Store) *Server {
	srv := &Server{store: s, router: chi.NewRouter()}
	srv.middleware()
	srv.routes()
	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) middleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(30 * time.Second))
}

func (s *Server) routes() {
	s.router.Get("/healthz", s.handleHealthz)
	s.router.Get("/status", s.handleStatus)
	s.router.Handle("/swagger/*", httpSwagger.WrapHandler)

	s.router.Route("/monitors", func(r chi.Router) {
		r.Post("/", s.handleCreateMonitor)
		r.Get("/", s.handleListMonitors)
		r.Get("/{id}", s.handleGetMonitor)
		r.Put("/{id}", s.handleUpdateMonitor)
		r.Delete("/{id}", s.handleDeleteMonitor)
		r.Get("/{id}/checks", s.handleListChecks)
	})
}
