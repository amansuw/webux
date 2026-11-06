package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/brendan4linux/webux/internal/plank"
)

var plankUpgrader = websocket.Upgrader{
	CheckOrigin:     wsCheckOrigin,
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
}

// PlankHandler bridges the monkos UI to Hermes Agent (P.L.A.N.K.).
type PlankHandler struct {
	mgr *plank.Manager
}

func NewPlankHandler() *PlankHandler {
	return &PlankHandler{mgr: plank.NewManager()}
}

// Status handles GET /api/plank/status
func (h *PlankHandler) Status(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	st := h.mgr.GetStatus(ctx)
	writeJSON(w, map[string]any{
		"online":        st.Online,
		"serving":       st.Serving,
		"url":           st.URL,
		"model":         st.Model,
		"provider":      st.Provider,
		"skin":          st.Skin,
		"personality":   st.Personality,
		"agent_name":    st.AgentName,
		"tagline":       st.Tagline,
		"error":         st.Error,
		"started_by_us": st.StartedByUs,
	})
}

// ServeWS handles GET /ws/plank — authenticated proxy to Hermes JSON-RPC WS.
func (h *PlankHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	err := h.mgr.Ensure(ctx)
	cancel()
	if err != nil {
		http.Error(w, `{"error":"Cannot. Reason: Hermes offline. `+err.Error()+`"}`, http.StatusBadGateway)
		return
	}

	client, err := plankUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("plank ws upgrade", "err", err)
		return
	}
	defer client.Close()

	hermesURL := h.mgr.WSURL()
	hermes, _, err := websocket.DefaultDialer.Dial(hermesURL, nil)
	if err != nil {
		_ = client.WriteMessage(websocket.TextMessage, mustJSON(map[string]any{
			"jsonrpc": "2.0",
			"method":  "event",
			"params": map[string]any{
				"type": "error",
				"payload": map[string]any{
					"message": "Cannot. Reason: Hermes WS dial failed. " + err.Error(),
				},
			},
		}))
		return
	}
	defer hermes.Close()

	_ = client.WriteMessage(websocket.TextMessage, mustJSON(map[string]any{
		"jsonrpc": "2.0",
		"method":  "event",
		"params": map[string]any{
			"type": "plank.bridge",
			"payload": map[string]any{
				"ok":     true,
				"source": plank.SourceTag,
			},
		},
	}))

	var wg sync.WaitGroup
	done := make(chan struct{})
	once := sync.Once{}
	closeDone := func() { once.Do(func() { close(done) }) }

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer closeDone()
		for {
			mt, data, err := hermes.ReadMessage()
			if err != nil {
				return
			}
			if err := client.WriteMessage(mt, data); err != nil {
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer closeDone()
		for {
			mt, data, err := client.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.TextMessage {
				data = plank.InjectWebuxSource(data)
			}
			if err := hermes.WriteMessage(mt, data); err != nil {
				return
			}
		}
	}()

	<-done
	_ = client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bridge closed"))
	_ = hermes.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bridge closed"))
	wg.Wait()
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
