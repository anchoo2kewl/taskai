package collab

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Client represents a WebSocket client connection
type Client struct {
	ID            string
	UserID        int64
	PageID        int64
	Conn          *websocket.Conn
	Send          chan []byte
	manager       *Manager
	roomID        string
	closedMu      sync.Mutex
	closed        bool
	HandleMessage func([]byte) // Custom message handler
}

// Message represents a WebSocket message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Manager manages WebSocket connections and rooms
type Manager struct {
	// Rooms maps room ID (page ID) to connected clients
	rooms map[string]map[*Client]bool

	// Broadcast sends messages to all clients in a room
	broadcast chan *BroadcastMessage

	// Register new clients
	register chan *Client

	// Unregister clients
	unregister chan *Client

	// Mutex for thread-safe access
	mu sync.RWMutex

	// Logger
	logger *zap.Logger

	// Context for shutdown
	ctx context.Context
}

// BroadcastMessage represents a message to broadcast to a room
type BroadcastMessage struct {
	RoomID  string
	Message []byte
	Exclude *Client // Optional client to exclude from broadcast
}

// NewManager creates a new WebSocket manager
func NewManager(ctx context.Context, logger *zap.Logger) *Manager {
	m := &Manager{
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan *BroadcastMessage, 256),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		logger:     logger,
		ctx:        ctx,
	}

	// Start the manager's event loop
	go m.run()

	return m
}

// run processes manager events
func (m *Manager) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Info("WebSocket manager shutting down")
			m.closeAllConnections()
			return

		case client := <-m.register:
			m.registerClient(client)

		case client := <-m.unregister:
			m.unregisterClient(client)

		case msg := <-m.broadcast:
			m.broadcastToRoom(msg)

		case <-ticker.C:
			// Periodic cleanup - can be extended for health checks
			m.mu.RLock()
			roomCount := len(m.rooms)
			clientCount := 0
			for _, clients := range m.rooms {
				clientCount += len(clients)
			}
			m.mu.RUnlock()

			m.logger.Debug("WebSocket stats",
				zap.Int("rooms", roomCount),
				zap.Int("clients", clientCount),
			)
		}
	}
}

// RegisterClient registers a new client to a room
func (m *Manager) RegisterClient(client *Client, roomID string) {
	client.manager = m
	client.roomID = roomID
	m.register <- client
}

// UnregisterClient unregisters a client
func (m *Manager) UnregisterClient(client *Client) {
	m.unregister <- client
}

// Broadcast sends a message to all clients in a room
func (m *Manager) Broadcast(roomID string, message []byte, exclude *Client) {
	m.broadcast <- &BroadcastMessage{
		RoomID:  roomID,
		Message: message,
		Exclude: exclude,
	}
}

// registerClient adds a client to a room
func (m *Manager) registerClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create room if it doesn't exist
	if m.rooms[client.roomID] == nil {
		m.rooms[client.roomID] = make(map[*Client]bool)
		m.logger.Info("Created new room",
			zap.String("room_id", client.roomID),
		)
	}

	// Add client to room
	m.rooms[client.roomID][client] = true

	m.logger.Info("Client joined room",
		zap.String("client_id", client.ID),
		zap.Int64("user_id", client.UserID),
		zap.String("room_id", client.roomID),
		zap.Int("room_size", len(m.rooms[client.roomID])),
	)

	// Start client read/write pumps
	go client.writePump()
	go client.readPump()
}

// unregisterClient removes a client from a room
func (m *Manager) unregisterClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if client is in a room
	if clients, ok := m.rooms[client.roomID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)

			// Close the client's send channel
			client.closedMu.Lock()
			if !client.closed {
				close(client.Send)
				client.closed = true
			}
			client.closedMu.Unlock()

			// Remove room if empty
			if len(clients) == 0 {
				delete(m.rooms, client.roomID)
				m.logger.Info("Removed empty room",
					zap.String("room_id", client.roomID),
				)
			}

			m.logger.Info("Client left room",
				zap.String("client_id", client.ID),
				zap.Int64("user_id", client.UserID),
				zap.String("room_id", client.roomID),
				zap.Int("room_size", len(clients)),
			)
		}
	}
}

// broadcastToRoom sends a message to all clients in a room
func (m *Manager) broadcastToRoom(msg *BroadcastMessage) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients, exists := m.rooms[msg.RoomID]
	if !exists {
		return
	}

	for client := range clients {
		// Skip the excluded client (usually the sender)
		if msg.Exclude != nil && client == msg.Exclude {
			continue
		}

		// Non-blocking send
		select {
		case client.Send <- msg.Message:
		default:
			// Client's send buffer is full, skip
			m.logger.Warn("Client send buffer full, dropping message",
				zap.String("client_id", client.ID),
			)
		}
	}
}

// closeAllConnections closes all client connections
func (m *Manager) closeAllConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for roomID, clients := range m.rooms {
		for client := range clients {
			client.closedMu.Lock()
			if !client.closed {
				close(client.Send)
				client.closed = true
				client.Conn.Close()
			}
			client.closedMu.Unlock()
		}
		delete(m.rooms, roomID)
	}
}

// GetRoomSize returns the number of clients in a room
func (m *Manager) GetRoomSize(roomID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if clients, exists := m.rooms[roomID]; exists {
		return len(clients)
	}
	return 0
}

// Constants for WebSocket configuration
const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512 KB
)

// readPump pumps messages from the WebSocket connection to the manager
func (c *Client) readPump() {
	defer func() {
		c.manager.UnregisterClient(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.manager.logger.Error("WebSocket read error",
					zap.String("client_id", c.ID),
					zap.Error(err),
				)
			}
			break
		}

		// Handle the message (to be implemented in wiki_ws_handlers.go)
		c.handleMessage(message)
	}
}

// writePump pumps messages from the manager to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *Client) handleMessage(message []byte) {
	// Use custom handler if set, otherwise echo back
	if c.HandleMessage != nil {
		c.HandleMessage(message)
	} else {
		// Default: echo back to the room for testing
		c.manager.Broadcast(c.roomID, message, c)
	}
}
