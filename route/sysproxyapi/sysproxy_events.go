package sysproxyapi

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/UruhaLushia/sparkle-service/route/httphelper"
)

const (
	sysproxyEventGuardStarted       = "guard_started"
	sysproxyEventGuardStopped       = "guard_stopped"
	sysproxyEventGuardChanged       = "guard_changed"
	sysproxyEventGuardRestored      = "guard_restored"
	sysproxyEventGuardRestoreFailed = "guard_restore_failed"
	sysproxyEventGuardCheckFailed   = "guard_check_failed"
	sysproxyEventGuardWatchFailed   = "guard_watch_failed"
)

type sysproxyEvent struct {
	Seq     uint64    `json:"seq,omitempty"`
	Type    string    `json:"type"`
	Time    time.Time `json:"time"`
	Guard   bool      `json:"guard"`
	Mode    string    `json:"mode,omitempty"`
	Message string    `json:"message,omitempty"`
	Error   string    `json:"error,omitempty"`
}

type sysproxyEventHub struct {
	mutex       sync.Mutex
	subscribers map[chan sysproxyEvent]struct{}
	last        sysproxyEvent
	nextSeq     uint64
}

var globalSysproxyEvents = &sysproxyEventHub{}

func subscribeSysproxyEvents(buffer int) (<-chan sysproxyEvent, func()) {
	if buffer < 1 {
		buffer = 1
	}

	ch := make(chan sysproxyEvent, buffer)
	globalSysproxyEvents.mutex.Lock()
	if globalSysproxyEvents.subscribers == nil {
		globalSysproxyEvents.subscribers = make(map[chan sysproxyEvent]struct{})
	}
	globalSysproxyEvents.subscribers[ch] = struct{}{}
	last := globalSysproxyEvents.last
	globalSysproxyEvents.mutex.Unlock()

	if last.Type == "" {
		last = newSysproxyEvent(sysproxyEventGuardStopped, "", false, "系统代理守护未运行", nil)
	}
	ch <- last

	unsubscribe := func() {
		globalSysproxyEvents.mutex.Lock()
		if _, ok := globalSysproxyEvents.subscribers[ch]; ok {
			delete(globalSysproxyEvents.subscribers, ch)
			close(ch)
		}
		globalSysproxyEvents.mutex.Unlock()
	}

	return ch, unsubscribe
}

func publishSysproxyGuardEvent(eventType string, mode sysproxyGuardMode, guard bool, message string, err error) {
	event := newSysproxyEvent(eventType, mode, guard, message, err)

	globalSysproxyEvents.mutex.Lock()
	globalSysproxyEvents.nextSeq++
	event.Seq = globalSysproxyEvents.nextSeq
	globalSysproxyEvents.last = event
	for ch := range globalSysproxyEvents.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
	globalSysproxyEvents.mutex.Unlock()
}

func newSysproxyEvent(eventType string, mode sysproxyGuardMode, guard bool, message string, err error) sysproxyEvent {
	event := sysproxyEvent{
		Type:    eventType,
		Time:    time.Now(),
		Guard:   guard,
		Mode:    string(mode),
		Message: message,
	}
	if err != nil {
		event.Error = err.Error()
	}
	return event
}

func sysproxyEvents(w http.ResponseWriter, r *http.Request) {
	conn, _, err := httphelper.AcceptWebSocket(w, r)
	if err != nil {
		httphelper.SendError(w, err)
		return
	}
	defer conn.Close()

	events, unsubscribe := subscribeSysproxyEvents(16)
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
			if err := writeSysproxyEvent(writeFrame, event); err != nil {
				return
			}
		case <-done:
			return
		case <-r.Context().Done():
			return
		}
	}
}

func writeSysproxyEvent(writeFrame func(byte, []byte) error, event sysproxyEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return writeFrame(httphelper.WebSocketOpText, payload)
}
