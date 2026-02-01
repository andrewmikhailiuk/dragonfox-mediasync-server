package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"

	"dragonfox-mediasync-server/hub"
	"dragonfox-mediasync-server/protocol"
	ws "dragonfox-mediasync-server/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, using environment variables")
	}
	setupLogger()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	broadcaster := hub.New()
	handler := protocol.NewHandler(broadcaster)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler(broadcaster, handler))
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/stats", statsHandler(broadcaster))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		slog.Info("server starting", "port", port)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("server shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}

func setupLogger() {
	level := slog.LevelInfo
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
}

func wsHandler(broadcaster *hub.Hub, handler *protocol.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("upgrade error", "error", err)
			return
		}

		room := r.URL.Query().Get("room")
		if room == "" {
			room = "default"
		}

		wsConn := ws.NewConn(uuid.New().String(), room, conn, broadcaster, handler)
		wsConn.Start()
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func statsHandler(broadcaster *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rooms, clients := broadcaster.Stats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"rooms": rooms, "clients": clients})
	}
}
