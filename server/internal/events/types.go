package events

import "time"

// EventType - тип события
type EventType string

const (
	// События игроков
	EventPlayerJoined  EventType = "player_joined"
	EventPlayerSplit   EventType = "player_split"
	EventPlayerEjected EventType = "player_ejected"
	EventPlayerDied    EventType = "player_died"
	
	// События клеток
	EventCellMerged  EventType = "cell_merged"
	EventCellEaten   EventType = "cell_eaten"
	
	// События еды
	EventFoodSpawned EventType = "food_spawned"
	EventFoodEaten   EventType = "food_eaten"
	
	// State updates
	EventStateDelta    EventType = "state_delta" // НОВОЕ: delta updates
	EventWorldSnapshot EventType = "world_snapshot"
)

// Event - базовое событие
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// PlayerJoinedEvent - игрок подключился
type PlayerJoinedEvent struct {
	PlayerID string  `json:"playerId"`
	Name     string  `json:"name"`
	Color    string  `json:"color"`
	IsBot    bool    `json:"isBot"`
	CellID   string  `json:"cellId"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Radius   float64 `json:"radius"`
}

// EntityDelta - изменение entity (клетки или еды)
type EntityDelta struct {
	ID      string  `json:"id"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Radius  float64 `json:"radius"`
	TargetX float64 `json:"targetX"` // Куда движется клетка
	TargetY float64 `json:"targetY"`
}

// StateDeltaEvent - delta update (только изменения)
type StateDeltaEvent struct {
	Tick      int64          `json:"tick"`
	Timestamp int64          `json:"timestamp"`
	Entities  []EntityDelta  `json:"entities"`
}

// PlayerSplitEvent - игрок разделился
type PlayerSplitEvent struct {
	PlayerID string   `json:"playerId"`
	NewCells []CellInfo `json:"newCells"`
}

type CellInfo struct {
	CellID string  `json:"cellId"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
	VelX   float64 `json:"velX"`
	VelY   float64 `json:"velY"`
}

// PlayerEjectedEvent - игрок выбросил массу
type PlayerEjectedEvent struct {
	PlayerID string     `json:"playerId"`
	Food     []FoodInfo `json:"food"`
}

type FoodInfo struct {
	FoodID string  `json:"foodId"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
	Color  string  `json:"color"`
	VelX   float64 `json:"velX"`
	VelY   float64 `json:"velY"`
}

// CellMergedEvent - клетки слились
type CellMergedEvent struct {
	PlayerID      string  `json:"playerId"`
	Cell1ID       string  `json:"cell1Id"`
	Cell2ID       string  `json:"cell2Id"`
	NewCellID     string  `json:"newCellId"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	Radius        float64 `json:"radius"`
}

// FoodEatenEvent - еда съедена
type FoodEatenEvent struct {
	FoodID   string `json:"foodId"`
	PlayerID string `json:"playerId"`
	CellID   string `json:"cellId"`
}

// CellEatenEvent - клетка съедена
type CellEatenEvent struct {
	EatenCellID string `json:"eatenCellId"`
	EatenBy     string `json:"eatenBy"`
	EaterCellID string `json:"eaterCellId"`
}

// FoodSpawnedEvent - еда создана
type FoodSpawnedEvent struct {
	Foods []FoodInfo `json:"foods"`
}

// PlayerDiedEvent - игрок умер
type PlayerDiedEvent struct {
	PlayerID string `json:"playerId"`
}

// WorldSnapshotEvent - полный снимок мира для синхронизации
type WorldSnapshotEvent struct {
	Timestamp int64          `json:"timestamp"`
	Players   []PlayerState  `json:"players"`
	Food      []FoodState    `json:"food"`
}

type PlayerState struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Color string      `json:"color"`
	IsBot bool        `json:"isBot"`
	Score int         `json:"score"`
	Cells []CellState `json:"cells"`
}

type CellState struct {
	ID     string  `json:"id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
}

type FoodState struct {
	ID     string  `json:"id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"radius"`
	Color  string  `json:"color"`
}

// NewEvent - создать событие
func NewEvent(eventType EventType, data interface{}) *Event {
	return &Event{
		Type:      eventType,
		Timestamp: time.Now().UnixMilli(),
		Data:      data,
	}
}
