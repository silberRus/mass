package game

import (
	"agario-server/internal/events"
	"math"
	"math/rand"
	"sync"
	"time"
)

type World struct {
	Players  map[string]*Player
	Food     map[string]*Food
	Mu       sync.RWMutex
	rand     *rand.Rand
	EventBus *events.EventBus // Event Bus для публикации событий
}

func NewWorld() *World {
	w := &World{
		Players:  make(map[string]*Player),
		Food:     make(map[string]*Food),
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
		EventBus: events.NewEventBus(),
	}
	
	// Инициализируем еду
	w.spawnInitialFood()
	
	return w
}

func (w *World) spawnInitialFood() {
	for i := 0; i < MaxFoodCount; i++ {
		w.spawnFood()
	}
}

func (w *World) spawnFood() *Food {
	x := w.rand.Float64() * WorldWidth
	y := w.rand.Float64() * WorldHeight
	color := randomFoodColor(w.rand)
	
	food := NewFood(Vector2D{X: x, Y: y}, color)
	w.Food[food.ID] = food
	return food
}

// SpawnFoodUnlocked - создание еды БЕЗ лока
func (w *World) SpawnFoodUnlocked() {
	w.spawnFood()
}

func (w *World) AddPlayer(name string, color string, isBot bool) *Player {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	return w.AddPlayerUnlocked(name, color, isBot)
}

// AddPlayerUnlocked - добавление игрока БЕЗ лока (когда лок уже есть)
func (w *World) AddPlayerUnlocked(name string, color string, isBot bool) *Player {
	player := NewPlayer(name, color, isBot)
	w.Players[player.ID] = player
	
	// Публикуем событие PlayerJoined
	if len(player.Cells) > 0 {
		firstCell := player.Cells[0]
		w.EventBus.PublishEvent(events.EventPlayerJoined, &events.PlayerJoinedEvent{
			PlayerID: player.ID,
			Name:     player.Name,
			Color:    player.Color,
			IsBot:    player.IsBot,
			CellID:   firstCell.ID,
			X:        firstCell.Position.X,
			Y:        firstCell.Position.Y,
			Radius:   firstCell.Radius,
		})
	}
	
	return player
}

func (w *World) RemovePlayer(playerID string) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	delete(w.Players, playerID)
}

func (w *World) GetPlayer(playerID string) (*Player, bool) {
	w.Mu.RLock()
	defer w.Mu.RUnlock()
	player, exists := w.Players[playerID]
	return player, exists
}

// Update - основной игровой цикл (публичный метод с локом)
func (w *World) Update(dt float64) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	w.UpdateUnlocked(dt)
}

// UpdateUnlocked - обновление без лока (для вызова когда лок уже есть)
func (w *World) UpdateUnlocked(dt float64) {
	
	// Обновляем движение всех клеток
	for _, player := range w.Players {
		w.updatePlayerMovement(player, dt)
	}
	
	// Применяем деградацию массы для больших клеток
	w.applyMassDegradation(dt)
	
	// Обновляем выброшенную еду
	w.updateFood(dt)
	
	// Проверяем коллизии
	w.checkCollisions()
	
	// Проверяем слияние клеток
	w.checkCellMerging()
	
	// Удаляем мертвых игроков
	w.removeDeadPlayers()
	
	// Пополняем еду
	w.maintainFood()
	
	// ВАЖНО: Публикуем обновления позиций клеток каждые 3 тика (10 раз в секунду)
	// Это дает плавное движение без избыточного трафика
	w.publishCellUpdates()
}

func (w *World) updatePlayerMovement(player *Player, dt float64) {
player.Mu.Lock()
defer player.Mu.Unlock()

for _, cell := range player.Cells {
// Направление к цели
direction := player.TargetPos.Sub(cell.Position).Normalize()

// Скорость зависит от массы
speed := cell.Speed()

// Обновляем позицию
velocity := direction.Mul(speed * dt)
newPos := cell.Position.Add(velocity)

// Ограничиваем мир
newPos.X = math.Max(cell.Radius, math.Min(WorldWidth-cell.Radius, newPos.X))
newPos.Y = math.Max(cell.Radius, math.Min(WorldHeight-cell.Radius, newPos.Y))

cell.Position = newPos
}
}

// applyMassDegradation - применяет деградацию массы для больших клеток
func (w *World) applyMassDegradation(dt float64) {
	const (
		// Минимальная "безопасная" масса - ниже этого порога деградации нет
		safeMassThreshold = 100.0
		
		// Базовый коэффициент деградации (очень малый)
		baseDegradationFactor = 0.0002
		
		// Экспоненциальный коэффициент для очень больших клеток
		exponentialFactor = 0.000005
	)
	
	for _, player := range w.Players {
		player.Mu.Lock()
		
		for _, cell := range player.Cells {
			currentMass := cell.Mass()
			
			// Если масса ниже порога - никакой деградации
			if currentMass <= safeMassThreshold {
				continue
			}
			
			// Рассчитываем потерю массы
			excessMass := currentMass - safeMassThreshold
			
			// Линейная часть: чем больше масса, тем быстрее потеря
			linearLoss := excessMass * baseDegradationFactor * dt
			
			// Экспоненциальная часть для очень больших клеток
			exponentialLoss := excessMass * excessMass * exponentialFactor * dt
			
			totalLoss := linearLoss + exponentialLoss
			
			// Применяем потерю, но не ниже порога
			newMass := math.Max(safeMassThreshold, currentMass-totalLoss)
			cell.SetMass(newMass)
		}
		
		player.Mu.Unlock()
	}
}

// updateFood - обновление движения выброшенной еды
func (w *World) updateFood(dt float64) {
	for _, food := range w.Food {
		// Если еда движется
		if food.Velocity.Length() > 0.1 {
			// Обновляем позицию
			food.Position = food.Position.Add(food.Velocity.Mul(dt))
			
			// Применяем трение (замедление)
			food.Velocity = food.Velocity.Mul(0.95)
			
			// Ограничиваем мир
			if food.Position.X < 0 || food.Position.X > WorldWidth {
				food.Velocity.X *= -0.5
				food.Position.X = math.Max(0, math.Min(WorldWidth, food.Position.X))
			}
			if food.Position.Y < 0 || food.Position.Y > WorldHeight {
				food.Velocity.Y *= -0.5
				food.Position.Y = math.Max(0, math.Min(WorldHeight, food.Position.Y))
			}
		}
	}
}

func (w *World) checkCollisions() {
	// Проверяем столкновения с едой
	for _, player := range w.Players {
		player.Mu.Lock()
		for _, cell := range player.Cells {
			for foodID, food := range w.Food {
				// Не съедаем еду которая только что выброшена (0.2 секунды защиты)
				if time.Since(food.SpawnTime).Seconds() < 0.2 {
					continue
				}
				if Distance(cell.Position, food.Position) < cell.Radius {
					// Клетка съела еду - добавляем массу еды
					cell.SetMass(cell.Mass() + food.Mass)
					delete(w.Food, foodID)
					
					// Публикуем событие
					w.EventBus.PublishEvent(events.EventFoodEaten, &events.FoodEatenEvent{
						FoodID:   foodID,
						PlayerID: player.ID,
						CellID:   cell.ID,
					})
				}
			}
		}
		player.Mu.Unlock()
	}
	
	// Проверяем столкновения между игроками
	players := make([]*Player, 0, len(w.Players))
	for _, p := range w.Players {
		players = append(players, p)
	}
	
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			w.checkPlayerCollision(players[i], players[j])
		}
	}
	
	// Проверяем столкновения клеток одного игрока
	for _, player := range w.Players {
		w.checkSelfCollision(player)
	}
}

func (w *World) checkPlayerCollision(p1, p2 *Player) {
	p1.Mu.Lock()
	defer p1.Mu.Unlock()
	p2.Mu.Lock()
	defer p2.Mu.Unlock()
	
	for i := len(p1.Cells) - 1; i >= 0; i-- {
		for j := len(p2.Cells) - 1; j >= 0; j-- {
			c1 := p1.Cells[i]
			c2 := p2.Cells[j]
			
			dist := Distance(c1.Position, c2.Position)
			if dist < c1.Radius || dist < c2.Radius {
			// Клетки касаются
			if c1.Mass() > c2.Mass()*MassToEat {
			// c1 съедает c2
			c1.SetMass(c1.Mass() + c2.Mass())
			p2.Cells = append(p2.Cells[:j], p2.Cells[j+1:]...)
			 
			// Публикуем событие
			w.EventBus.PublishEvent(events.EventCellEaten, &events.CellEatenEvent{
			 EatenCellID: c2.ID,
			 EatenBy:     p1.ID,
			  EaterCellID: c1.ID,
			  })
					} else if c2.Mass() > c1.Mass()*MassToEat {
						// c2 съедает c1
						c2.SetMass(c2.Mass() + c1.Mass())
						p1.Cells = append(p1.Cells[:i], p1.Cells[i+1:]...)
						
						// Публикуем событие
						w.EventBus.PublishEvent(events.EventCellEaten, &events.CellEatenEvent{
							EatenCellID: c1.ID,
							EatenBy:     p2.ID,
							EaterCellID: c2.ID,
						})
						break
					}
				}
		}
	}
}

func (w *World) checkSelfCollision(player *Player) {
	player.Mu.Lock()
	defer player.Mu.Unlock()
	
	// Клетки одного игрока не едят друг друга, только сливаются при cooldown
}

func (w *World) checkCellMerging() {
	for _, player := range w.Players {
		player.Mu.Lock()
		
		for i := 0; i < len(player.Cells); i++ {
			for j := i + 1; j < len(player.Cells); j++ {
				c1 := player.Cells[i]
				c2 := player.Cells[j]
				
				if !c1.CanMerge() || !c2.CanMerge() {
					continue
				}
				
				dist := Distance(c1.Position, c2.Position)
				if dist < (c1.Radius+c2.Radius)/2 {
					// Сливаем клетки
					c2ID := c2.ID // Сохраняем ID перед удалением
					c1.SetMass(c1.Mass() + c2.Mass())
					c1.LastMergeTime = time.Now()
					player.Cells = append(player.Cells[:j], player.Cells[j+1:]...)
					j--
					
					// Публикуем событие
					w.EventBus.PublishEvent(events.EventCellMerged, &events.CellMergedEvent{
						PlayerID:  player.ID,
						Cell1ID:   c1.ID,
						Cell2ID:   c2ID,
						NewCellID: c1.ID,
						X:         c1.Position.X,
						Y:         c1.Position.Y,
						Radius:    c1.Radius,
					})
				}
			}
		}
		
		player.Mu.Unlock()
	}
}

func (w *World) removeDeadPlayers() {
	for id, player := range w.Players {
		if !player.IsAlive() {
			delete(w.Players, id)
			
			// Публикуем событие
			w.EventBus.PublishEvent(events.EventPlayerDied, &events.PlayerDiedEvent{
				PlayerID: id,
			})
		}
	}
}

func (w *World) maintainFood() {
	currentFood := len(w.Food)
	if currentFood < MaxFoodCount {
		toSpawn := MaxFoodCount - currentFood
		
		// Собираем информацию о созданной еде
		if toSpawn > 0 {
			newFoods := []events.FoodInfo{}
			
			for i := 0; i < toSpawn; i++ {
				food := w.spawnFood()
				newFoods = append(newFoods, events.FoodInfo{
					FoodID: food.ID,
					X:      food.Position.X,
					Y:      food.Position.Y,
					Radius: food.Radius,
					Color:  food.Color,
					VelX:   0,
					VelY:   0,
				})
			}
			
			// Публикуем событие
			w.EventBus.PublishEvent(events.EventFoodSpawned, &events.FoodSpawnedEvent{
				Foods: newFoods,
			})
		}
	}
}

// Split - разделить клетку игрока (публичный метод с локами)
func (w *World) Split(playerID string) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	
	player, exists := w.Players[playerID]
	if !exists {
		return
	}
	
	w.splitPlayer(player)
}

// SplitPlayerUnlocked - разделение БЕЗ лока (когда лок уже есть)
func (w *World) SplitPlayerUnlocked(player *Player) {
	w.splitPlayer(player)
}

// splitPlayer - внутренний метод без локов
func (w *World) splitPlayer(player *Player) {
	player.Mu.Lock()
	defer player.Mu.Unlock()
	
	if len(player.Cells) >= PlayerMaxCells {
		return
	}
	
	newCells := []*Cell{}
	
	for _, cell := range player.Cells {
		if !cell.CanSplit() || cell.Mass() < 20 { // уменьшили с 40 до 20
			continue
		}
		
		// Делим клетку пополам
		newMass := cell.Mass() / 2
		cell.SetMass(newMass)
		cell.LastSplitTime = time.Now()
		
		// Направление split
		direction := player.TargetPos.Sub(cell.Position).Normalize()
		
		// Новая клетка появляется рядом и получает импульс
		offset := direction.Mul(cell.Radius * 1.2)
		newPos := cell.Position.Add(offset)
		
		newCell := NewCell(newPos, 0)
		newCell.SetMass(newMass)
		newCell.LastSplitTime = time.Now()
		newCell.LastMergeTime = time.Now()
		
		// Небольшой импульс вперед (не далеко!)
		impulseSpeed := 800.0 // Фиксированная скорость
		newCell.Velocity = direction.Mul(impulseSpeed)
		
		newCells = append(newCells, newCell)
	}
	
	player.Cells = append(player.Cells, newCells...)
	
	// Публикуем событие если были созданы новые клетки
	if len(newCells) > 0 {
		newCellsInfo := []events.CellInfo{}
		for _, cell := range newCells {
			newCellsInfo = append(newCellsInfo, events.CellInfo{
				CellID: cell.ID,
				X:      cell.Position.X,
				Y:      cell.Position.Y,
				Radius: cell.Radius,
				VelX:   cell.Velocity.X,
				VelY:   cell.Velocity.Y,
			})
		}
		
		w.EventBus.PublishEvent(events.EventPlayerSplit, &events.PlayerSplitEvent{
			PlayerID: player.ID,
			NewCells: newCellsInfo,
		})
	}
}

// Eject - выбросить массу (публичный метод с локами)
func (w *World) Eject(playerID string) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	
	player, exists := w.Players[playerID]
	if !exists {
		return
	}
	
	w.ejectMass(player)
}

// ejectMass - внутренний метод без локов
func (w *World) ejectMass(player *Player) {
	player.Mu.Lock()
	defer player.Mu.Unlock()
	
	ejectedFoods := []events.FoodInfo{}
	
	for _, cell := range player.Cells {
		if cell.Mass() < EjectMass+10 {
			continue
		}
		
		// Уменьшаем массу клетки
		cell.SetMass(cell.Mass() - EjectMass)
		
		// Направление выброса
		direction := player.TargetPos.Sub(cell.Position).Normalize()
		
		// Дистанция выброса зависит от массы клетки (больше масса = дальше бросок)
		throwDistance := cell.Radius * 1.5
		offset := direction.Mul(throwDistance)
		foodPos := cell.Position.Add(offset)
		
		// Скорость выброса пропорциональна массе (но не слишком быстро)
		throwSpeed := EjectSpeed * math.Sqrt(cell.Mass()) / 10.0
		if throwSpeed > EjectSpeed * 2 {
			throwSpeed = EjectSpeed * 2
		}
		velocity := direction.Mul(throwSpeed)
		
		// Добавляем еду напрямую (мир уже залочен)
		food := NewEjectedFood(foodPos, player.Color, EjectMass, velocity)
		w.Food[food.ID] = food
		
		// Собираем информацию для события
		ejectedFoods = append(ejectedFoods, events.FoodInfo{
			FoodID: food.ID,
			X:      food.Position.X,
			Y:      food.Position.Y,
			Radius: food.Radius,
			Color:  food.Color,
			VelX:   food.Velocity.X,
			VelY:   food.Velocity.Y,
		})
	}
	
	// Публикуем событие если была выброшена еда
	if len(ejectedFoods) > 0 {
		w.EventBus.PublishEvent(events.EventPlayerEjected, &events.PlayerEjectedEvent{
			PlayerID: player.ID,
			Food:     ejectedFoods,
		})
	}
}

func randomFoodColor(r *rand.Rand) string {
	colors := []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#FFA07A",
		"#98D8C8", "#F7DC6F", "#BB8FCE", "#85C1E2",
		"#F8B739", "#52BE80", "#EC7063", "#5DADE2",
	}
	return colors[r.Intn(len(colors))]
}

// publishCellUpdates - публикует обновления позиций клеток (вызывается каждый тик)
var cellUpdateCounter = 0

func (w *World) publishCellUpdates() {
	cellUpdateCounter++
	
	// Отправляем обновления каждые 3 тика (10 раз в секунду)
	if cellUpdateCounter < 3 {
		return
	}
	cellUpdateCounter = 0
	
	// Собираем все обновления
	for _, player := range w.Players {
		player.Mu.RLock()
		for _, cell := range player.Cells {
			w.EventBus.PublishEvent(events.EventCellUpdated, &events.CellUpdatedEvent{
				CellID:   cell.ID,
				PlayerID: player.ID,
				X:        cell.Position.X,
				Y:        cell.Position.Y,
				Radius:   cell.Radius,
			})
		}
		player.Mu.RUnlock()
	}
}
