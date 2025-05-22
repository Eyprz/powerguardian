package exporter

import (
	"log"
	"net/http"
	"os"
	"time"

	"powerguardian/internal/config"
)

var metricsFile string

func HttpServer(config config.Config, file string, listenAddr string) *http.Server {
	metricsFile = file
	http.HandleFunc("/metrics", exporter)
	server := &http.Server{
		Addr:         listenAddr,
		Handler:      nil,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  20 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	return server
}

func exporter(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile(metricsFile)
	if err != nil {
		log.Fatalf("Failed to read %s: %v", metricsFile, err)
		http.Error(w, "Failed to read metrics file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err = w.Write(content)
	if err != nil {
		log.Fatalf("Failed to write response: %v", err)
		return
	}
}
