// Package ws implements the real-time broadcast layer that fans MQTT metrics
// out to every connected browser WebSocket client.
//
// Design:
//   - A single Hub goroutine owns the client set. All mutations (register,
//     unregister, broadcast) funnel through channels to avoid locks.
//   - Per-client send buffers prevent a slow reader from blocking the hub.
//     When a client's buffer fills up, we disconnect it rather than dropping
//     messages silently.
//   - Ingestion feeds the Hub via Broadcast(metric); the hub pushes to every
//     registered client's send channel.
package ws

import (
	"log/slog"
	"sync/atomic"
)

// Event is the JSON envelope broadcast to every connected client. Kept small
// and flat so the SvelteKit store can route it with a single dispatch.
type Event struct {
	DeviceID string  `json:"device_id"`
	Sensor   string  `json:"sensor"`
	Value    float64 `json:"value"`
	Time     string  `json:"time"` // RFC3339Nano
}

// Client is anything that can receive events. The concrete WebSocket
// implementation lives in ws_routes.go; this interface keeps the hub testable.
//
// IMPORTANT: implementations MUST treat Send() as a non-blocking fire-and-forget
// channel. The hub uses non-blocking sends and evicts clients whose buffer is
// full, so the client MUST NOT close the send channel itself — doing so creates
// a send-on-closed-channel panic. The writer goroutine exits when the
// underlying connection fails, not when the channel closes.
type Client interface {
	ID() uint64
	Send() chan<- Event
}

// Hub coordinates registration and broadcast. Construct with NewHub and run
// with Run(ctx) in its own goroutine.
type Hub struct {
	register   chan Client
	unregister chan Client
	broadcast  chan Event
	idGen      atomic.Uint64
}

// NewHub constructs an empty hub. Call Run in a goroutine to start accepting
// registrations and broadcasts.
func NewHub() *Hub {
	return &Hub{
		register:   make(chan Client),
		unregister: make(chan Client),
		broadcast:  make(chan Event, 256),
	}
}

// NextID returns a unique identifier for a new client. Exposed so ws_routes.go
// can tag each connection before registering.
func (h *Hub) NextID() uint64 { return h.idGen.Add(1) }

// Register adds a client. Blocks until the hub's goroutine accepts it.
func (h *Hub) Register(c Client) { h.register <- c }

// Unregister removes a client. BLOCKING — callers rely on the client being
// removed from the hub's map before they tear down the underlying connection,
// otherwise the hub may send on a channel whose writer is gone.
func (h *Hub) Unregister(c Client) { h.unregister <- c }

// Broadcast delivers an event to every connected client. If the hub's input
// channel is full (ingestion pushing faster than the hub can fan out), the
// event is dropped and logged — CLAUDE.md explicitly accepts this trade-off.
func (h *Hub) Broadcast(ev Event) {
	select {
	case h.broadcast <- ev:
	default:
		slog.Warn("ws: hub broadcast channel full, dropping event",
			"device_id", ev.DeviceID, "sensor", ev.Sensor)
	}
}

// Run owns the client set and drives fanout. Returns when the input channels
// are closed or the context used by the embedded signals is cancelled
// (signalled by closing h.broadcast upstream).
func (h *Hub) Run(stop <-chan struct{}) {
	clients := make(map[uint64]Client)

	for {
		select {
		case <-stop:
			return

		case c := <-h.register:
			clients[c.ID()] = c

		case c := <-h.unregister:
			if _, ok := clients[c.ID()]; ok {
				delete(clients, c.ID())
			}

		case ev := <-h.broadcast:
			for id, c := range clients {
				select {
				case c.Send() <- ev:
				default:
					// Client too slow — drop it rather than block the whole hub.
					slog.Warn("ws: slow client, disconnecting", "client_id", id)
					delete(clients, id)
				}
			}
		}
	}
}
