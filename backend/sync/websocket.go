package sync

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for development
		// In production, you should validate the origin
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WebSocketManager manages WebSocket connections
type WebSocketManager struct {
	clients    map[int]map[*websocket.Conn]bool // userID -> connections
	clientsMux sync.RWMutex
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
}

// Client represents a WebSocket client
type Client struct {
	conn   *websocket.Conn
	userID int
	send   chan Message
}

// Message represents a WebSocket message
type Message struct {
	Type      string      `json:"type"`
	UserID    int         `json:"user_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// Message types
const (
	MsgTypeSyncStart    = "sync_start"
	MsgTypeSyncProgress = "sync_progress"
	MsgTypeSyncComplete = "sync_complete"
	MsgTypeSyncError    = "sync_error"
	MsgTypeRollback     = "rollback"
	MsgTypeNotification = "notification"
	MsgTypeHeartbeat    = "heartbeat"
)

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:    make(map[int]map[*websocket.Conn]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the WebSocket manager
func (wsm *WebSocketManager) Run() {
	for {
		select {
		case client := <-wsm.register:
			wsm.registerClient(client)

		case client := <-wsm.unregister:
			wsm.unregisterClient(client)

		case message := <-wsm.broadcast:
			wsm.broadcastMessage(message)
		}
	}
}

// registerClient registers a new client
func (wsm *WebSocketManager) registerClient(client *Client) {
	wsm.clientsMux.Lock()
	defer wsm.clientsMux.Unlock()

	if wsm.clients[client.userID] == nil {
		wsm.clients[client.userID] = make(map[*websocket.Conn]bool)
	}
	wsm.clients[client.userID][client.conn] = true

	log.Printf("WebSocket client registered for user %d", client.userID)

	// Send welcome message
	welcomeMsg := Message{
		Type:      "connected",
		Data:      map[string]string{"message": "WebSocket connected successfully"},
		Timestamp: time.Now(),
	}

	select {
	case client.send <- welcomeMsg:
	default:
		close(client.send)
		delete(wsm.clients[client.userID], client.conn)
	}
}

// unregisterClient unregisters a client
func (wsm *WebSocketManager) unregisterClient(client *Client) {
	wsm.clientsMux.Lock()
	defer wsm.clientsMux.Unlock()

	if clients, ok := wsm.clients[client.userID]; ok {
		if _, ok := clients[client.conn]; ok {
			delete(clients, client.conn)
			close(client.send)

			if len(clients) == 0 {
				delete(wsm.clients, client.userID)
			}

			log.Printf("WebSocket client unregistered for user %d", client.userID)
		}
	}
}

// broadcastMessage broadcasts a message to appropriate clients
func (wsm *WebSocketManager) broadcastMessage(message Message) {
	wsm.clientsMux.RLock()
	defer wsm.clientsMux.RUnlock()

	// If message has a specific user ID, send only to that user
	if message.UserID > 0 {
		if clients, ok := wsm.clients[message.UserID]; ok {
			for conn := range clients {
				if client := wsm.getClientByConn(conn, message.UserID); client != nil {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(clients, conn)
					}
				}
			}
		}
		return
	}

	// Broadcast to all clients
	for userID, clients := range wsm.clients {
		for conn := range clients {
			if client := wsm.getClientByConn(conn, userID); client != nil {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(clients, conn)
				}
			}
		}
	}
}

// getClientByConn helper function to get client by connection
func (wsm *WebSocketManager) getClientByConn(conn *websocket.Conn, userID int) *Client {
	return &Client{
		conn:   conn,
		userID: userID,
		send:   make(chan Message, 256),
	}
}

// SendToUser sends a message to a specific user
func (wsm *WebSocketManager) SendToUser(userID int, msgType string, data interface{}) {
	message := Message{
		Type:      msgType,
		UserID:    userID,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case wsm.broadcast <- message:
	default:
		log.Printf("Failed to send message to user %d: broadcast channel full", userID)
	}
}

// SendToAll sends a message to all connected clients
func (wsm *WebSocketManager) SendToAll(msgType string, data interface{}) {
	message := Message{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case wsm.broadcast <- message:
	default:
		log.Printf("Failed to broadcast message: broadcast channel full")
	}
}

// HandleWebSocket handles WebSocket connections
func (wsm *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you'd extract userID from JWT token
	// For this example, we'll assume it's passed as a query parameter
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "Missing user_id parameter", http.StatusBadRequest)
		return
	}

	var userID int
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		http.Error(w, "Invalid user_id parameter", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		conn:   conn,
		userID: userID,
		send:   make(chan Message, 256),
	}

	wsm.register <- client

	go wsm.writePump(client)
	go wsm.readPump(client)
}

// writePump pumps messages from the hub to the websocket connection
func (wsm *WebSocketManager) writePump(client *Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub
func (wsm *WebSocketManager) readPump(client *Client) {
	defer func() {
		wsm.unregister <- client
		client.conn.Close()
	}()

	client.conn.SetReadLimit(512)
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var message Message
		err := client.conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages (e.g., heartbeat, client commands)
		wsm.handleClientMessage(client, message)
	}
}

// handleClientMessage handles messages from clients
func (wsm *WebSocketManager) handleClientMessage(client *Client, message Message) {
	switch message.Type {
	case MsgTypeHeartbeat:
		// Respond to heartbeat
		response := Message{
			Type:      MsgTypeHeartbeat,
			Data:      map[string]string{"status": "alive"},
			Timestamp: time.Now(),
		}
		select {
		case client.send <- response:
		default:
			// Client send channel is full, disconnect
			wsm.unregister <- client
		}

	default:
		log.Printf("Unknown message type from client: %s", message.Type)
	}
}

// GetConnectedUsers returns the number of connected users
func (wsm *WebSocketManager) GetConnectedUsers() int {
	wsm.clientsMux.RLock()
	defer wsm.clientsMux.RUnlock()
	return len(wsm.clients)
}

// GetUserConnectionCount returns the number of connections for a specific user
func (wsm *WebSocketManager) GetUserConnectionCount(userID int) int {
	wsm.clientsMux.RLock()
	defer wsm.clientsMux.RUnlock()

	if clients, ok := wsm.clients[userID]; ok {
		return len(clients)
	}
	return 0
}

// NotifyProgress sends sync progress updates
func (wsm *WebSocketManager) NotifyProgress(userID int, operationID int, progress int, message string) {
	data := map[string]interface{}{
		"operation_id": operationID,
		"progress":     progress,
		"message":      message,
	}
	wsm.SendToUser(userID, MsgTypeSyncProgress, data)
}

// NotifyComplete sends sync completion notification
func (wsm *WebSocketManager) NotifyComplete(userID int, operationID int, result interface{}) {
	data := map[string]interface{}{
		"operation_id": operationID,
		"result":       result,
	}
	wsm.SendToUser(userID, MsgTypeSyncComplete, data)
}

// NotifyError sends error notification
func (wsm *WebSocketManager) NotifyError(userID int, operationID int, error string) {
	data := map[string]interface{}{
		"operation_id": operationID,
		"error":        error,
	}
	wsm.SendToUser(userID, MsgTypeSyncError, data)
}
