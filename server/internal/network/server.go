package network

import (
	"agario-server/internal/bot"
	"agario-server/internal/events"
	"agario-server/internal/game"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan []byte
	PlayerID string
	Server   *Server
}

type Server struct {
	World      *game.World
	Clients    map[string]*Client
	Register   chan *Client
	Unregister chan *Client
	Commands   chan *PlayerCommand
	mu         sync.RWMutex
	
	// Для периодической синхронизации
	lastSnapshotTime time.Time
	snapshotInterval time.Duration
}

type PlayerCommand struct {
	Type     string
	ClientID string
	Data     interface{}
}

func NewServer(world *game.World) *Server {
	return &Server{
		World:            world,
		Clients:          make(map[string]*Client),
		Register:         make(chan *Client, 10),
		Unregister:       make(chan *Client, 10),
		Commands:         make(chan *PlayerCommand, 100),
		lastSnapshotTime: time.Now(),
		snapshotInterval: 10 * time.Second, // Редкий snapshot для подстраховки (основная синхронизация через cell_updated)
	}
}

func (s *Server) Run(botManager *bot.BotManager) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SERVER] ❌ PANIC in Run(): %v", r)
		}
	}()

	log.Println("[SERVER] Run() started")
	ticker := time.NewTicker(game.TickDuration)
	defer ticker.Stop()

	botUpdateCounter := 0
	botUpdateInterval := 10

	log.Println("[SERVER] Entering main loop...")

	for {
		select {
		case client := <-s.Register:
			log.Printf("[SERVER] Register case triggered for client %s", client.ID)
			s.mu.Lock()
			s.Clients[client.ID] = client
			s.mu.Unlock()
			log.Printf("Client registered: %s", client.ID)

		case client := <-s.Unregister:
			s.mu.Lock()
			if _, ok := s.Clients[client.ID]; ok {
				delete(s.Clients, client.ID)
				
				func() {
					defer func() {
						if r := recover(); r != nil {}
					}()
					close(client.Send)
				}()

				if client.PlayerID != "" {
					s.World.RemovePlayer(client.PlayerID)
					log.Printf("[SERVER] Player %s removed from world", client.PlayerID)
				}
			}
			s.mu.Unlock()
			log.Printf("Client unregistered: %s", client.ID)

		case <-ticker.C:
			processedCmds := 0
			for {
				select {
				case cmd := <-s.Commands:
					s.handleCommand(cmd)
					processedCmds++
				default:
					goto UpdateWorld
				}
			}

		UpdateWorld:
			s.World.Mu.Lock()
			s.World.UpdateUnlocked(game.TickDuration.Seconds())
			botUpdateCounter++
			if botUpdateCounter >= botUpdateInterval {
				botUpdateCounter = 0
				botManager.Update()
			}
			s.World.Mu.Unlock()
			
			// Отправляем события вместо полного состояния!
			s.broadcastEvents()
		}
	}
}

// broadcastEvents - отправка только событий клиентам
func (s *Server) broadcastEvents() {
	// Получаем накопленные события
	events := s.World.EventBus.FlushEvents()
	
	// Проверяем нужен ли snapshot
	needSnapshot := time.Since(s.lastSnapshotTime) >= s.snapshotInterval
	
	if needSnapshot {
		// Создаем и отправляем snapshot
		s.broadcastSnapshot()
		s.lastSnapshotTime = time.Now()
	} else if len(events) > 0 {
		// Отправляем только события
		data, err := s.World.EventBus.SerializeEvents(events)
		if err != nil {
			log.Printf("[BROADCAST] Error serializing events: %v", err)
			return
		}
		
		if data == nil {
			return
		}
		
		// Отправляем events batch всем клиентам
		s.mu.RLock()
		deadClients := []*Client{}
		for _, client := range s.Clients {
			select {
			case client.Send <- data:
			default:
				log.Printf("[BROADCAST] Client %s has full/closed channel, marking for removal", client.ID)
				deadClients = append(deadClients, client)
			}
		}
		s.mu.RUnlock()
		
		// Удаляем мертвые клиенты
		if len(deadClients) > 0 {
			for _, client := range deadClients {
				log.Printf("[BROADCAST] Force unregistering dead client %s", client.ID)
				select {
				case s.Unregister <- client:
				default:
				}
			}
		}
	}
}

// broadcastSnapshot - отправка полного снимка мира для синхронизации
func (s *Server) broadcastSnapshot() {
	s.World.Mu.RLock()
	defer s.World.Mu.RUnlock()

	players := []events.PlayerState{}
	for _, p := range s.World.Players {
		p.Mu.RLock()
		cells := []events.CellState{}
		for _, cell := range p.Cells {
			cells = append(cells, events.CellState{
				ID:     cell.ID,
				X:      cell.Position.X,
				Y:      cell.Position.Y,
				Radius: cell.Radius,
			})
		}
		players = append(players, events.PlayerState{
			ID:    p.ID,
			Name:  p.Name,
			Color: p.Color,
			IsBot: p.IsBot,
			Score: p.GetScore(),
			Cells: cells,
		})
		p.Mu.RUnlock()
	}

	food := []events.FoodState{}
	for _, f := range s.World.Food {
		food = append(food, events.FoodState{
			ID:     f.ID,
			X:      f.Position.X,
			Y:      f.Position.Y,
			Radius: f.Radius,
			Color:  f.Color,
		})
	}

	snapshot := &events.WorldSnapshotEvent{
		Timestamp: time.Now().UnixMilli(),
		Players:   players,
		Food:      food,
	}

	event := events.NewEvent(events.EventWorldSnapshot, snapshot)
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[SNAPSHOT] Error marshaling snapshot: %v", err)
		return
	}

	// Отправляем snapshot всем клиентам
	s.mu.RLock()
	for _, client := range s.Clients {
		select {
		case client.Send <- data:
		default:
			// Пропускаем если канал заполнен
		}
	}
	s.mu.RUnlock()
	
	log.Printf("[SNAPSHOT] Sent snapshot: %d players, %d food", len(players), len(food))
}

// Остальные методы остаются без изменений...
func (s *Server) handleCommand(cmd *PlayerCommand) {
	switch cmd.Type {
	case "join":
		s.processJoin(cmd)
	case "move":
		s.processMove(cmd)
	case "split":
		s.processSplit(cmd)
	case "eject":
		s.processEject(cmd)
	}
}

func (s *Server) processJoin(cmd *PlayerCommand) {
	joinData := cmd.Data.(map[string]interface{})
	name := joinData["name"].(string)
	log.Printf("[SERVER] Processing join for %s, name: %s", cmd.ClientID, name)

	s.World.Mu.Lock()
	color := randomPlayerColor()
	player := s.World.AddPlayerUnlocked(name, color, false)
	s.World.Mu.Unlock()

	s.mu.Lock()
	if client, ok := s.Clients[cmd.ClientID]; ok {
		client.PlayerID = player.ID
	}
	s.mu.Unlock()

	// Отправляем init сообщение (старый формат для обратной совместимости)
	initData := map[string]interface{}{
		"playerId": player.ID,
		"worldSize": map[string]float64{
			"width":  game.WorldWidth,
			"height": game.WorldHeight,
		},
	}

	msg := map[string]interface{}{
		"type": "init",
		"data": initData,
	}

	msgData, _ := json.Marshal(msg)

	s.mu.RLock()
	if client, ok := s.Clients[cmd.ClientID]; ok {
		client.Send <- msgData
	}
	s.mu.RUnlock()

	log.Printf("Player joined: %s (%s)", name, player.ID)
}

func (s *Server) processMove(cmd *PlayerCommand) {
	moveData := cmd.Data.(map[string]interface{})
	x := moveData["x"].(float64)
	y := moveData["y"].(float64)
	
	s.mu.RLock()
	client, ok := s.Clients[cmd.ClientID]
	s.mu.RUnlock()

	if !ok || client.PlayerID == "" {
		return
	}

	player, exists := s.World.GetPlayer(client.PlayerID)
	if !exists {
		return
	}

	player.SetTarget(x, y)
	// Позиции обновляются через state delta
}

func (s *Server) processSplit(cmd *PlayerCommand) {
	s.mu.RLock()
	client, ok := s.Clients[cmd.ClientID]
	s.mu.RUnlock()

	if !ok || client.PlayerID == "" {
		return
	}
	s.World.Split(client.PlayerID)
}

func (s *Server) processEject(cmd *PlayerCommand) {
	s.mu.RLock()
	client, ok := s.Clients[cmd.ClientID]
	s.mu.RUnlock()

	if !ok || client.PlayerID == "" {
		return
	}
	s.World.Eject(client.PlayerID)
}

func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("[WEBSOCKET] New connection request from %s", r.RemoteAddr)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WEBSOCKET] ❌ Upgrade error: %v", err)
		return
	}
	log.Printf("[WEBSOCKET] ✅ WebSocket upgraded successfully")

	client := &Client{
		ID:     generateClientID(),
		Conn:   conn,
		Send:   make(chan []byte, 16),
		Server: s,
	}

	log.Printf("[WEBSOCKET] Created client with ID: %s", client.ID)
	s.Register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		log.Printf("[CLIENT %s] readPump closing", c.ID)
		c.Server.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		c.handleMessage(message)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(10 * time.Second)
	defer func() {
		log.Printf("[CLIENT %s] writePump closing", c.ID)
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
			if !ok {
				log.Printf("[CLIENT %s] Send channel closed", c.ID)
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[CLIENT %s] Write error: %v", c.ID, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[CLIENT %s] Ping error: %v", c.ID, err)
				return
			}
		}
	}
}

func (c *Client) handleMessage(message []byte) {
	var clientMsg map[string]interface{}
	if err := json.Unmarshal(message, &clientMsg); err != nil {
		log.Printf("[CLIENT %s] Error unmarshaling message: %v", c.ID, err)
		return
	}

	msgType, ok := clientMsg["type"].(string)
	if !ok {
		return
	}

	data := clientMsg["data"]

	switch msgType {
	case "join":
		c.Server.Commands <- &PlayerCommand{
			Type:     "join",
			ClientID: c.ID,
			Data:     data,
		}

	case "move":
		c.Server.Commands <- &PlayerCommand{
			Type:     "move",
			ClientID: c.ID,
			Data:     data,
		}

	case "split":
		c.Server.Commands <- &PlayerCommand{
			Type:     "split",
			ClientID: c.ID,
			Data:     nil,
		}

	case "eject":
		c.Server.Commands <- &PlayerCommand{
			Type:     "eject",
			ClientID: c.ID,
			Data:     nil,
		}
	}
}

func generateClientID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

func randomPlayerColor() string {
	colors := []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#FFA07A",
		"#98D8C8", "#F7DC6F", "#BB8FCE", "#85C1E2",
	}
	return colors[time.Now().UnixNano()%int64(len(colors))]
}
