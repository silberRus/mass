package protocol

import "encoding/json"

// MessageType определяет типы сообщений
type MessageType string

const (
	// Client -> Server
	MsgTypeJoin  MessageType = "join"
	MsgTypeMove  MessageType = "move"
	MsgTypeSplit MessageType = "split"
	MsgTypeEject MessageType = "eject"

	// Server -> Client
	MsgTypeInit        MessageType = "init"
	MsgTypeState       MessageType = "state"
	MsgTypePlayerDied  MessageType = "player_died"
	MsgTypeLeaderboard MessageType = "leaderboard"
)

// ClientMessage - базовая структура сообщения от клиента
type ClientMessage struct {
	Type MessageType     `json:"type"`
	Data json.RawMessage `json:"data"`
}

// ServerMessage - базовая структура сообщения от сервера
type ServerMessage struct {
	Type MessageType `json:"type"`
	Data interface{} `json:"data"`
}

// === Client -> Server ===

type JoinData struct {
	Name string `json:"name"`
}

type MoveData struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// === Server -> Client ===

type InitData struct {
	PlayerID  string    `json:"playerId"`
	WorldSize WorldSize `json:"worldSize"`
}

type WorldSize struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type StateData struct {
	Timestamp int64          `json:"timestamp"`
	Players   []PlayerState  `json:"players"`
	Food      []FoodState    `json:"food"`
}

type PlayerState struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Cells  []CellState `json:"cells"`
	Score  int         `json:"score"`
	IsBot  bool        `json:"isBot"`
}

type CellState struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
	Color  string  `json:"color"`
}

type FoodState struct {
	ID     string  `json:"id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Color  string  `json:"color"`
	Radius float64 `json:"radius"`
}

type PlayerDiedData struct {
	PlayerID string `json:"playerId"`
	KillerID string `json:"killerId,omitempty"`
}

type LeaderboardData struct {
	Leaders []LeaderEntry `json:"leaders"`
}

type LeaderEntry struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}
