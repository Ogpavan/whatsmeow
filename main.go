package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"wa-mvp-api/internal/api"
	"wa-mvp-api/internal/session"
)

func main() {
	manager := session.GetManager()
	if err := manager.RestoreSessionsOnStartup(); err != nil {
		log.Printf("restore sessions error: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/", api.HandleDocs)
	api.RegisterSessionRoutes(r)

	srv := &http.Server{
		Addr:              ":9090",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}
