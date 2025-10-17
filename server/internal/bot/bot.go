package bot

import (
	"agario-server/internal/game"
	"math"
	"math/rand"
	"time"
)

type Bot struct {
	Player         *game.Player
	World          *game.World
	rand           *rand.Rand
	nextDecision   time.Time
	decisionDelay  time.Duration
}

// NewBot - создание бота (с локом для начальной инициализации)
func NewBot(name string, world *game.World) *Bot {
	color := randomColor()
	player := world.AddPlayer(name, color, true)
	
	return &Bot{
		Player:        player,
		World:         world,
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
		nextDecision:  time.Now(),
		decisionDelay: 300 * time.Millisecond,
	}
}

// NewBotUnlocked - создание бота БЕЗ лока (когда world.Mu.Lock уже есть)
func NewBotUnlocked(name string, world *game.World) *Bot {
	color := randomColor()
	player := world.AddPlayerUnlocked(name, color, true)
	
	return &Bot{
		Player:        player,
		World:         world,
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
		nextDecision:  time.Now(),
		decisionDelay: 300 * time.Millisecond,
	}
}

// Update - обновление AI бота
// ВАЖНО: вызывается из world.Update() который уже держит world.Mu.Lock()
// Поэтому НЕ берем никаких локов!
func (b *Bot) Update() {
	if !b.Player.IsAlive() {
		return
	}
	
	if time.Now().Before(b.nextDecision) {
		return
	}
	
	b.nextDecision = time.Now().Add(b.decisionDelay)
	
	// Получаем центр масс БЕЗ ЛОКОВ (world lock уже есть)
	if len(b.Player.Cells) == 0 {
		return
	}
	
	centerX, centerY := 0.0, 0.0
	for _, cell := range b.Player.Cells {
		centerX += cell.Position.X
		centerY += cell.Position.Y
	}
	centerX /= float64(len(b.Player.Cells))
	centerY /= float64(len(b.Player.Cells))
	center := game.Vector2D{X: centerX, Y: centerY}
	
	// Ищем ближайшую еду или слабого противника
	target := b.findTarget(center)
	
	if target != nil {
		b.Player.SetTarget(target.X, target.Y)
		
		// Иногда пытаемся разделиться если противник близко
		if b.shouldSplit(center, *target) {
			b.World.SplitPlayerUnlocked(b.Player)
		}
	} else {
		// Случайное движение если нет цели
		b.wanderRandomly(center)
	}
}

func (b *Bot) findTarget(center game.Vector2D) *game.Vector2D {
	const searchRadius = 400.0
	
	var closestFood *game.Vector2D
	closestFoodDist := math.MaxFloat64
	
	// Ищем ближайшую еду БЕЗ ЛОКОВ
	for _, food := range b.World.Food {
		dist := game.Distance(center, food.Position)
		if dist < searchRadius && dist < closestFoodDist {
			closestFoodDist = dist
			pos := food.Position
			closestFood = &pos
		}
	}
	
	// Ищем слабых противников БЕЗ ЛОКОВ
	var closestEnemy *game.Vector2D
	closestEnemyDist := math.MaxFloat64
	
	// Считаем массу бота БЕЗ ЛОКОВ
	botMass := 0.0
	for _, cell := range b.Player.Cells {
		botMass += cell.Mass()
	}
	
	for _, player := range b.World.Players {
		if player.ID == b.Player.ID || !player.IsAlive() {
			continue
		}
		
		// Считаем массу БЕЗ ЛОКОВ
		enemyMass := 0.0
		for _, cell := range player.Cells {
			enemyMass += cell.Mass()
		}
		
		if len(player.Cells) > 0 {
			enemyCenter := game.Vector2D{X: 0, Y: 0}
			for _, cell := range player.Cells {
				enemyCenter.X += cell.Position.X
				enemyCenter.Y += cell.Position.Y
			}
			enemyCenter.X /= float64(len(player.Cells))
			enemyCenter.Y /= float64(len(player.Cells))
			
			dist := game.Distance(center, enemyCenter)
			
			// Если мы больше на 20% и противник близко - атакуем
			if botMass > enemyMass*1.2 && dist < searchRadius && dist < closestEnemyDist {
				closestEnemyDist = dist
				closestEnemy = &enemyCenter
			}
			
			// Если мы меньше и противник близко - убегаем
			if botMass < enemyMass*0.8 && dist < searchRadius/2 {
				// Убегаем в противоположную сторону
				direction := center.Sub(enemyCenter).Normalize()
				escape := center.Add(direction.Mul(searchRadius))
				return &escape
			}
		}
	}
	
	// Приоритет: противники > еда
	if closestEnemy != nil && closestEnemyDist < closestFoodDist/2 {
		return closestEnemy
	}
	
	if closestFood != nil {
		return closestFood
	}
	
	return nil
}

func (b *Bot) shouldSplit(center game.Vector2D, target game.Vector2D) bool {
	dist := game.Distance(center, target)
	
	// БЕЗ ЛОКОВ - world lock уже есть
	if len(b.Player.Cells) >= game.PlayerMaxCells/2 {
		return false
	}
	
	// Считаем массу БЕЗ ЛОКОВ
	totalMass := 0.0
	for _, cell := range b.Player.Cells {
		totalMass += cell.Mass()
	}
	
	// Сплитаемся если цель близко и у нас достаточно массы
	if dist < 100 && totalMass > 80 {
		if b.rand.Float64() < 0.3 { // 30% шанс
			for _, cell := range b.Player.Cells {
				if cell.CanSplit() {
					return true
				}
			}
		}
	}
	
	return false
}

func (b *Bot) wanderRandomly(center game.Vector2D) {
	// Движемся к случайной точке недалеко от текущей позиции
	angle := b.rand.Float64() * 2 * math.Pi
	distance := 200.0 + b.rand.Float64()*300.0
	
	targetX := center.X + math.Cos(angle)*distance
	targetY := center.Y + math.Sin(angle)*distance
	
	// Ограничиваем мир
	targetX = math.Max(50, math.Min(game.WorldWidth-50, targetX))
	targetY = math.Max(50, math.Min(game.WorldHeight-50, targetY))
	
	b.Player.SetTarget(targetX, targetY)
	// Позиция обновляется через state delta, события не нужны
}

func randomColor() string {
	colors := []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#FFA07A",
		"#98D8C8", "#F7DC6F", "#BB8FCE", "#85C1E2",
		"#F8B739", "#52BE80", "#EC7063", "#5DADE2",
	}
	return colors[rand.Intn(len(colors))]
}

// BotManager - управление ботами
type BotManager struct {
	Bots      []*Bot
	World     *game.World
	MaxBots   int
	botNames  []string
	nameIndex int
}

func NewBotManager(world *game.World, maxBots int) *BotManager {
	return &BotManager{
		Bots:    make([]*Bot, 0),
		World:   world,
		MaxBots: maxBots,
		botNames: []string{
			"BotAlpha", "BotBeta", "BotGamma", "BotDelta",
			"BotEpsilon", "BotZeta", "BotEta", "BotTheta",
			"BotIota", "BotKappa", "BotLambda", "BotMu",
			"BotNu", "BotXi", "BotOmicron", "BotPi",
		},
		nameIndex: 0,
	}
}

// SpawnBots - создание ботов (с локом для начальной инициализации)
func (bm *BotManager) SpawnBots() {
	for len(bm.Bots) < bm.MaxBots {
		name := bm.botNames[bm.nameIndex%len(bm.botNames)]
		bm.nameIndex++
		
		bot := NewBot(name, bm.World)
		bm.Bots = append(bm.Bots, bot)
	}
}

// SpawnBotsUnlocked - создание ботов БЕЗ лока (когда world.Mu.Lock уже есть)
func (bm *BotManager) SpawnBotsUnlocked() {
	for len(bm.Bots) < bm.MaxBots {
		name := bm.botNames[bm.nameIndex%len(bm.botNames)]
		bm.nameIndex++
		
		bot := NewBotUnlocked(name, bm.World)
		bm.Bots = append(bm.Bots, bot)
	}
}

// Update - обновление ботов БЕЗ локов (вызывается когда world.Mu.Lock уже есть)
func (bm *BotManager) Update() {
	// Обновляем всех ботов
	for i := len(bm.Bots) - 1; i >= 0; i-- {
		bot := bm.Bots[i]
		
		if !bot.Player.IsAlive() {
			// Удаляем мертвого бота
			bm.Bots = append(bm.Bots[:i], bm.Bots[i+1:]...)
			continue
		}
		
		bot.Update()
	}
	
	// Пополняем ботов если их мало (БЕЗ лока - он уже есть!)
	bm.SpawnBotsUnlocked()
}
