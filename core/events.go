package core

import (
	"sync"
	"time"
)

const (
	CoreEventStarting      = "starting"
	CoreEventStarted       = "started"
	CoreEventStopping      = "stopping"
	CoreEventStopped       = "stopped"
	CoreEventExited        = "exited"
	CoreEventRestarting    = "restarting"
	CoreEventRestartFailed = "restart_failed"
	CoreEventTakeover      = "takeover"
	CoreEventReady         = "ready"
	CoreEventFailed        = "failed"
	CoreEventLog           = "log"
)

type CoreEvent struct {
	Seq     uint64            `json:"seq,omitempty"`
	Type    string            `json:"type"`
	Time    time.Time         `json:"time"`
	Running bool              `json:"running"`
	PID     int32             `json:"pid,omitempty"`
	OldPID  int32             `json:"old_pid,omitempty"`
	Message string            `json:"message,omitempty"`
	Error   string            `json:"error,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
}

type coreEventHub struct {
	mutex       sync.Mutex
	subscribers map[chan CoreEvent]struct{}
	last        CoreEvent
	nextSeq     uint64
}

func (cm *CoreManager) SubscribeEvents(buffer int) (<-chan CoreEvent, func()) {
	if buffer < 1 {
		buffer = 1
	}

	ch := make(chan CoreEvent, buffer)
	cm.eventHub.mutex.Lock()
	if cm.eventHub.subscribers == nil {
		cm.eventHub.subscribers = make(map[chan CoreEvent]struct{})
	}
	cm.eventHub.subscribers[ch] = struct{}{}
	last := cm.eventHub.last
	cm.eventHub.mutex.Unlock()

	if last.Type == "" {
		last = cm.newCoreEvent(CoreEventStopped, "核心未运行", nil, 0, 0)
	}
	ch <- last

	unsubscribe := func() {
		cm.eventHub.mutex.Lock()
		if _, ok := cm.eventHub.subscribers[ch]; ok {
			delete(cm.eventHub.subscribers, ch)
			close(ch)
		}
		cm.eventHub.mutex.Unlock()
	}

	return ch, unsubscribe
}

func (cm *CoreManager) publishCoreEvent(event CoreEvent) {
	if event.Time.IsZero() {
		event.Time = time.Now()
	}
	if event.PID == 0 {
		event.PID = cm.pid.Load()
	}
	event.Running = cm.isRunning.Load() && event.PID > 0

	cm.eventHub.mutex.Lock()
	cm.eventHub.nextSeq++
	event.Seq = cm.eventHub.nextSeq
	if event.Type != CoreEventLog {
		cm.eventHub.last = event
	}
	for ch := range cm.eventHub.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
	cm.eventHub.mutex.Unlock()
}

func (cm *CoreManager) newCoreEvent(eventType string, message string, err error, pid int32, oldPID int32) CoreEvent {
	event := CoreEvent{
		Type:    eventType,
		Time:    time.Now(),
		Running: cm.isRunning.Load(),
		PID:     pid,
		OldPID:  oldPID,
		Message: message,
	}
	if event.PID == 0 {
		event.PID = cm.pid.Load()
	}
	if err != nil {
		event.Error = err.Error()
	}
	event.Running = event.Running && event.PID > 0
	return event
}

func (cm *CoreManager) emitCoreEvent(eventType string, message string, err error) {
	cm.publishCoreEvent(cm.newCoreEvent(eventType, message, err, 0, 0))
}
