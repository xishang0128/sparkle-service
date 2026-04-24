package route

import (
	"encoding/json"
	"net/http"
	"sync"

	"sparkle-service/core"
)

func coreEvents(w http.ResponseWriter, r *http.Request) {
	conn, _, err := acceptWebSocket(w, r)
	if err != nil {
		sendError(w, err)
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
		return writeWebSocketFrame(conn, opcode, payload)
	}

	go func() {
		defer close(done)
		for {
			opcode, payload, err := readWebSocketFrame(conn)
			if err != nil {
				return
			}
			switch opcode {
			case webSocketOpClose:
				_ = writeFrame(webSocketOpClose, nil)
				return
			case webSocketOpPing:
				_ = writeFrame(webSocketOpPong, payload)
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

func writeCoreEvent(writeFrame func(byte, []byte) error, event core.CoreEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return writeFrame(webSocketOpText, payload)
}
