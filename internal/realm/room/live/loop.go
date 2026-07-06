package live

import (
	"context"
	"time"
)

// startLoop starts the room owner goroutine.
func (room *Room) startLoop(ctx context.Context, interval time.Duration, publisher MovementPublisher) {
	if publisher == nil || interval <= 0 {
		return
	}

	room.mutex.Lock()
	if room.loopCancel != nil {
		room.mutex.Unlock()
		return
	}
	loopCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	room.loopCancel = cancel
	room.loopDone = done
	room.mutex.Unlock()

	go room.runLoop(loopCtx, interval, publisher, done)
}

// stopLoop stops the room owner goroutine.
func (room *Room) stopLoop() {
	room.mutex.Lock()
	cancel := room.loopCancel
	done := room.loopDone
	room.loopCancel = nil
	room.loopDone = nil
	room.mutex.Unlock()

	if cancel == nil {
		return
	}
	cancel()
	if done != nil {
		<-done
	}
}

// runLoop runs room ticks until stopped.
func (room *Room) runLoop(ctx context.Context, interval time.Duration, publisher MovementPublisher, done chan<- struct{}) {
	defer close(done)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			movements := room.Tick()
			if len(movements) == 0 {
				continue
			}
			_ = publisher(ctx, room, movements)
		}
	}
}
