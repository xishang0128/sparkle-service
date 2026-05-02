package coreapi

import (
	"encoding/json"
	"net/http"
	"sync"

	corepkg "github.com/UruhaLushia/sparkle-service/core"
	"github.com/UruhaLushia/sparkle-service/route/httphelper"
)

func coreEvents(w http.ResponseWriter, r *http.Request) {
	conn, _, err := httphelper.AcceptWebSocket(w, r)
	if err != nil {
		httphelper.SendError(w, err)
		return
	}
	defer conn.Close()

	events, unsubscribe := cm.SubscribeEvents(16)
	defer unsubscribe()

	done := make(chan struct{})
	var writeMu sync.Mutex
	writeFrame := func(opcode byte, payload []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return httphelper.WriteWebSocketFrame(conn, opcode, payload)
	}

	go func() {
		defer close(done)
		for {
			opcode, payload, err := httphelper.ReadWebSocketFrame(conn)
			if err != nil {
				return
			}
			switch opcode {
			case httphelper.WebSocketOpClose:
				_ = writeFrame(httphelper.WebSocketOpClose, nil)
				return
			case httphelper.WebSocketOpPing:
				_ = writeFrame(httphelper.WebSocketOpPong, payload)
			}
		}
	}()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writeCoreEvent(writeFrame, event); err != nil {
				return
			}
		case <-done:
			return
		case <-r.Context().Done():
			return
		}
	}
}

func writeCoreEvent(writeFrame func(byte, []byte) error, event corepkg.CoreEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return writeFrame(httphelper.WebSocketOpText, payload)
}
