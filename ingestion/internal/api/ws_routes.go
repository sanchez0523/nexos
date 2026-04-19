package api

import (
	"log/slog"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"github.com/nexos-io/nexos/ingestion/internal/api/ws"
)

// wsClient adapts a *websocket.Conn into the ws.Client interface. The send
// channel is owned by the hub (read-side) and this client (write-side).
//
// Lifecycle & safety contract:
//   - The writer goroutine exits when `stop` is closed, NOT when `send` is
//     closed. We never close(send) because the hub may still hold a reference
//     and its non-blocking send would panic on a closed channel.
//   - On disconnect we (a) block on Unregister so the hub drops us from its
//     map, (b) close `stop` so the writer exits, (c) let GC reclaim `send`.
type wsClient struct {
	id   uint64
	send chan ws.Event
	stop chan struct{}
}

func (c *wsClient) ID() uint64            { return c.id }
func (c *wsClient) Send() chan<- ws.Event { return c.send }

// writePumpTimeout bounds each WriteJSON call so a dead TCP connection cannot
// wedge the writer goroutine forever (ReadMessage handles reader-side TCP
// zombies via the socket's own error surface).
const writePumpTimeout = 10 * time.Second

// handleWebSocket validates the access token from the httpOnly cookie that
// the browser attaches automatically to the WebSocket handshake, flags the
// request as allowed for gofiber/contrib/websocket, and delegates to the
// upgrade handler.
//
// Because the handshake is a regular HTTP GET before the upgrade, the same
// SameSite=Strict cookie used by REST requests is sent along — no query
// parameter token required, no URL leakage in logs.
func (s *Server) handleWebSocket() fiber.Handler {
	upgrade := websocket.New(func(conn *websocket.Conn) {
		defer conn.Close()

		client := &wsClient{
			id:   s.deps.Hub.NextID(),
			send: make(chan ws.Event, 64),
			stop: make(chan struct{}),
		}
		s.deps.Hub.Register(client)
		defer s.deps.Hub.Unregister(client) // blocking — ensures hub drops us

		writerDone := make(chan struct{})
		go func() {
			defer close(writerDone)
			for {
				select {
				case <-client.stop:
					return
				case ev := <-client.send:
					_ = conn.SetWriteDeadline(time.Now().Add(writePumpTimeout))
					if err := conn.WriteJSON(ev); err != nil {
						return
					}
				}
			}
		}()

		// Reader blocks on ReadMessage; returns on disconnect, bad frame, or
		// client-side close. We don't accept any payloads from the client.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}

		close(client.stop) // signal writer to exit
		<-writerDone
	})

	return func(c *fiber.Ctx) error {
		raw := c.Cookies(cookieAccessToken)
		if raw == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
		}
		if _, err := s.deps.Issuer.RequireAccess(raw); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}

		// gofiber/contrib/websocket (>=v1) consults this flag before upgrading.
		c.Locals("allowed", true)
		slog.Debug("ws: upgrading connection")
		return upgrade(c)
	}
}
