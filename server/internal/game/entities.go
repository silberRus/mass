package game

import (
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Игровые константы
const (
	WorldWidth  = 5000.0
	WorldHeight = 5000.0

	MinCellRadius  = 10.0
	MaxCellRadius  = 2500.0 // x5
	StartRadius    = 20.0
	FoodRadius     = 5.0
	MaxFoodCount   = 3000
	PlayerMaxCells = 16

	// Физика
	BaseSpeed     = 600.0  // базовая скорость (x3)
	SpeedDecay    = 0.3    // замедление от массы
	SplitCooldown = 0.5    // секунды до следующего сплита (было 1.0)
	MergeCooldown = 15.0   // секунды до слияния клеток
	EjectMass     = 12.0   // масса выброшенной еды
	EjectSpeed    = 1200.0 // скорость выброса (x3)

	// Геймплей
	MassToEat    = 1.15 // нужно быть на 15% больше чтобы съесть
	TickRate     = 30   // обновлений в секунду
	TickDuration = time.Second / TickRate
)

// Vector2D - 2D вектор
type Vector2D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (v Vector2D) Add(other Vector2D) Vector2D {
	return Vector2D{X: v.X + other.X, Y: v.Y + other.Y}
}

func (v Vector2D) Sub(other Vector2D) Vector2D {
	return Vector2D{X: v.X - other.X, Y: v.Y - other.Y}
}

func (v Vector2D) Mul(scalar float64) Vector2D {
	return Vector2D{X: v.X * scalar, Y: v.Y * scalar}
}

func (v Vector2D) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

func (v Vector2D) Normalize() Vector2D {
	length := v.Length()
	if length == 0 {
		return Vector2D{X: 0, Y: 0}
	}
	return Vector2D{X: v.X / length, Y: v.Y / length}
}

func Distance(a, b Vector2D) float64 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// Cell - игровая клетка
type Cell struct {
	ID            string
	Position      Vector2D
	Radius        float64
	Velocity      Vector2D
	LastSplitTime time.Time
	LastMergeTime time.Time
}

func NewCell(pos Vector2D, radius float64) *Cell {
	return &Cell{
		ID:            uuid.New().String(),
		Position:      pos,
		Radius:        radius,
		Velocity:      Vector2D{X: 0, Y: 0},
		LastSplitTime: time.Now(),
		LastMergeTime: time.Now(),
	}
}

func (c *Cell) Mass() float64 {
	return c.Radius * c.Radius / 100.0
}

func (c *Cell) SetMass(mass float64) {
	c.Radius = math.Sqrt(mass * 100.0)
	if c.Radius < MinCellRadius {
		c.Radius = MinCellRadius
	}
	if c.Radius > MaxCellRadius {
		c.Radius = MaxCellRadius
	}
}

func (c *Cell) Speed() float64 {
	return BaseSpeed / math.Pow(c.Mass(), SpeedDecay)
}

func (c *Cell) CanSplit() bool {
	return time.Since(c.LastSplitTime).Seconds() >= SplitCooldown
}

func (c *Cell) CanMerge() bool {
	return time.Since(c.LastMergeTime).Seconds() >= MergeCooldown
}

// Player - игрок
type Player struct {
	ID            string
	Name          string
	Color         string
	Cells         []*Cell
	TargetPos     Vector2D
	IsBot         bool
	LastInputTime time.Time
	Mu            sync.RWMutex
}

func NewPlayer(name string, color string, isBot bool) *Player {
	// Случайная стартовая позиция
	startX := MinCellRadius + math.Floor(math.Max(0, math.Min(WorldWidth-MinCellRadius*2, float64(time.Now().UnixNano()%int64(WorldWidth)))))
	startY := MinCellRadius + math.Floor(math.Max(0, math.Min(WorldHeight-MinCellRadius*2, float64((time.Now().UnixNano()/1000)%int64(WorldHeight)))))

	startCell := NewCell(Vector2D{X: startX, Y: startY}, StartRadius)

	return &Player{
		ID:            uuid.New().String(),
		Name:          name,
		Color:         color,
		Cells:         []*Cell{startCell},
		TargetPos:     Vector2D{X: startX, Y: startY},
		IsBot:         isBot,
		LastInputTime: time.Now(),
	}
}

func (p *Player) GetScore() int {
	p.Mu.RLock()
	defer p.Mu.RUnlock()

	totalMass := 0.0
	for _, cell := range p.Cells {
		totalMass += cell.Mass()
	}
	return int(math.Floor(totalMass))
}

func (p *Player) TotalMass() float64 {
	p.Mu.RLock()
	defer p.Mu.RUnlock()

	totalMass := 0.0
	for _, cell := range p.Cells {
		totalMass += cell.Mass()
	}
	return totalMass
}

func (p *Player) IsAlive() bool {
	p.Mu.RLock()
	defer p.Mu.RUnlock()
	return len(p.Cells) > 0
}

func (p *Player) SetTarget(x, y float64) {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	p.TargetPos = Vector2D{X: x, Y: y}
	p.LastInputTime = time.Now()
}

// Food - еда
type Food struct {
	ID        string
	Position  Vector2D
	Color     string
	Radius    float64
	Mass      float64   // Масса которую даёт эта еда
	Velocity  Vector2D  // Скорость движения (для выброшенной еды)
	SpawnTime time.Time // Время создания (чтобы не съедали сразу)
}

func NewFood(pos Vector2D, color string) *Food {
	return &Food{
		ID:        uuid.New().String(),
		Position:  pos,
		Color:     color,
		Radius:    FoodRadius,
		Mass:      1.0,
		Velocity:  Vector2D{X: 0, Y: 0},
		SpawnTime: time.Now(),
	}
}

// NewEjectedFood - создаёт выброшенную игроком еду
func NewEjectedFood(pos Vector2D, color string, mass float64, velocity Vector2D) *Food {
	// Радиус зависит от массы для визуального отличия
	radius := FoodRadius * math.Sqrt(mass)
	return &Food{
		ID:        uuid.New().String(),
		Position:  pos,
		Color:     color,
		Radius:    radius,
		Mass:      mass,
		Velocity:  velocity,
		SpawnTime: time.Now(),
	}
}
