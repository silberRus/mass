# 🎯 ОПТИМАЛЬНАЯ АРХИТЕКТУРА - План реализации

## Философия:
**"Server is authoritative, Client predicts and interpolates"**

---

## 🎮 Протокол коммуникации

### Client → Server (Input только!)
```
Частота: каждый клиентский frame (но throttled до 30/сек)
Размер: ~50 байт

{
  seq: uint32,           // Sequence для reconciliation
  timestamp: uint64,     // Client time
  input: {
    targetX: float32,
    targetY: float32,
    split: bool,
    eject: bool
  }
}
```

### Server → Client

#### 1. State Delta (10 раз/сек, ~100ms)
```
Размер: ~5-20 KB (зависит от изменений)

{
  type: "delta",
  tick: uint32,
  timestamp: uint64,
  entities: [
    {
      id: string,
      x?: float32,        // Только если изменилось > threshold
      y?: float32,
      radius?: float32,
      velX?: float32,     // Для экстраполяции
      velY?: float32
    }
  ]
}
```

#### 2. Events (когда происходят)
```
Размер: ~100-500 байт на событие

Только механики:
- player_joined: { id, name, color, cellId, x, y, radius }
- player_died: { id }
- cell_eaten: { eatenCellId, eaterCellId, eaterPlayerId }
- food_eaten: { foodId, cellId }
- player_split: { playerId, newCells: [{id, x, y, r, velX, velY}] }
- player_ejected: { playerId, foods: [{id, x, y, r, color, velX, velY}] }
- food_spawned: { foods: [{id, x, y, r, color}] }
```

#### 3. Snapshot (раз в 5 сек)
```
Размер: ~800 KB (полное состояние)

{
  type: "snapshot",
  tick: uint32,
  full_state: {
    players: [...],
    food: [...]
  }
}
```

---

## 🏗️ Этапы реализации

### ✅ Этап 1: Очистка событий (30 мин)
**Цель:** Убрать избыточные события позиций

**Действия:**
1. Удалить: player_moved, cell_updated из events
2. Оставить только: механики (split, eject, eat, join, die)
3. Убрать publishCellUpdates из world.go
4. Убрать публикацию движения из bot.go

**Результат:** События только для важных механик

---

### ✅ Этап 2: State Delta Updates (1 час)
**Цель:** Отправлять только изменившиеся позиции

**Серверная часть (Go):**

```go
// world.go - tracking changes
type EntityState struct {
    LastX      float64
    LastY      float64
    LastRadius float64
}

var entityStates = make(map[string]*EntityState)

// publishStateDelta - вызывается каждые 3 тика (100ms)
func (w *World) publishStateDelta() {
    const POSITION_THRESHOLD = 5.0  // Игнорируем изменения < 5 единиц
    const RADIUS_THRESHOLD = 0.5
    
    deltas := []EntityDelta{}
    
    for _, player := range w.Players {
        for _, cell := range player.Cells {
            lastState := entityStates[cell.ID]
            if lastState == nil {
                lastState = &EntityState{}
                entityStates[cell.ID] = lastState
            }
            
            // Проверяем изменения
            dx := math.Abs(cell.Position.X - lastState.LastX)
            dy := math.Abs(cell.Position.Y - lastState.LastY)
            dr := math.Abs(cell.Radius - lastState.LastRadius)
            
            if dx > POSITION_THRESHOLD || dy > POSITION_THRESHOLD || dr > RADIUS_THRESHOLD {
                deltas = append(deltas, EntityDelta{
                    ID:     cell.ID,
                    X:      cell.Position.X,
                    Y:      cell.Position.Y,
                    Radius: cell.Radius,
                })
                
                // Обновляем last state
                lastState.LastX = cell.Position.X
                lastState.LastY = cell.Position.Y
                lastState.LastRadius = cell.Radius
            }
        }
    }
    
    if len(deltas) > 0 {
        w.EventBus.PublishEvent(events.EventStateDelta, &events.StateDeltaEvent{
            Tick:     w.CurrentTick,
            Entities: deltas,
        })
    }
}
```

**Результат:** 
- Трафик: ~5-20 KB каждые 100ms = 50-200 KB/сек
- Вместо: 800 KB каждые 100ms = 8 MB/сек
- **Экономия: 40-160x!**

---

### ✅ Этап 3: Клиентская физика (1 час)
**Цель:** Клиент симулирует ТУ ЖЕ физику что сервер

**Клиентская часть (TypeScript):**

```typescript
// ClientPhysics.ts - та же логика что на сервере!
export class ClientPhysics {
  // КОПИЯ серверной логики движения
  static updateCellPosition(
    cell: Cell, 
    targetX: number, 
    targetY: number, 
    dt: number
  ) {
    // Направление к цели
    const dx = targetX - cell.x;
    const dy = targetY - cell.y;
    const distance = Math.sqrt(dx * dx + dy * dy);
    
    if (distance < 1) return;
    
    const ndx = dx / distance;
    const ndy = dy / distance;
    
    // ТА ЖЕ формула скорости что на сервере!
    const mass = (cell.radius * cell.radius) / 100;
    const speed = 600 / Math.pow(mass, 0.3);
    
    // Движение
    cell.x += ndx * speed * dt;
    cell.y += ndy * speed * dt;
    
    // Границы мира
    cell.x = Math.max(cell.radius, Math.min(5000 - cell.radius, cell.x));
    cell.y = Math.max(cell.radius, Math.min(5000 - cell.radius, cell.y));
  }
}
```

**Результат:** Клиент движется плавно между server updates

---

### ✅ Этап 4: Entity Interpolation (30 мин)
**Цель:** Плавная интерполяция между server updates

```typescript
// StateManager.ts
class InterpolationBuffer {
  private buffer: StateUpdate[] = [];
  private renderTime: number = 0;
  
  add(update: StateUpdate) {
    this.buffer.push(update);
    // Держим только последние 3 updates (300ms буфер)
    if (this.buffer.length > 3) {
      this.buffer.shift();
    }
  }
  
  getInterpolatedState(now: number): StateUpdate {
    // Render time отстает на 100ms для плавности
    this.renderTime = now - 100;
    
    // Находим два ближайших updates
    let before = null;
    let after = null;
    
    for (let i = 0; i < this.buffer.length - 1; i++) {
      if (this.buffer[i].timestamp <= this.renderTime && 
          this.buffer[i + 1].timestamp > this.renderTime) {
        before = this.buffer[i];
        after = this.buffer[i + 1];
        break;
      }
    }
    
    if (!before || !after) {
      return this.buffer[this.buffer.length - 1];
    }
    
    // Linear interpolation
    const t = (this.renderTime - before.timestamp) / 
              (after.timestamp - before.timestamp);
              
    return this.interpolate(before, after, t);
  }
}
```

**Результат:** Плавное движение даже при 100ms updates

---

### ✅ Этап 5: Snapshot Reconciliation (30 мин)
**Цель:** Коррекция рассинхронизации

```typescript
handleSnapshot(snapshot: Snapshot) {
  const RECONCILIATION_THRESHOLD = 50; // пикселей
  
  for (const serverPlayer of snapshot.players) {
    const localPlayer = this.players.get(serverPlayer.id);
    if (!localPlayer) continue;
    
    for (const serverCell of serverPlayer.cells) {
      const localCell = localPlayer.cells.get(serverCell.id);
      if (!localCell) continue;
      
      // Проверяем расстояние
      const dx = serverCell.x - localCell.x;
      const dy = serverCell.y - localCell.y;
      const distance = Math.sqrt(dx * dx + dy * dy);
      
      if (distance > RECONCILIATION_THRESHOLD) {
        // Значительный рассинхрон - корректируем плавно
        console.warn(`[RECONCILIATION] Cell ${serverCell.id} off by ${distance}px`);
        localCell.targetX = serverCell.x;
        localCell.targetY = serverCell.y;
        // Smooth correction over 500ms
        localCell.correctionSpeed = distance / 0.5;
      }
    }
  }
}
```

---

## 📊 Ожидаемые результаты

### Трафик:
```
Client → Server: 50 bytes × 30/sec = 1.5 KB/sec
Server → Client:
  - State deltas: 10 KB × 10/sec = 100 KB/sec
  - Events: ~5 KB/sec (спорадически)
  - Snapshot: 800 KB / 5 sec = 160 KB/sec

TOTAL: ~270 KB/sec (было 24 MB/sec!)
Улучшение: 90x!
```

### Качество:
```
✅ Плавное движение (клиентская физика + interpolation)
✅ Нет телепортации (smooth correction)
✅ Минимальный рассинхрон (< 50px)
✅ Работает при потере пакетов (buffer + reconciliation)
```

---

## 🔧 Порядок реализации

1. **Сейчас:** Этап 1 - Очистка событий
2. **+30 мин:** Этап 2 - State delta
3. **+1 час:** Этап 3 - Клиентская физика
4. **+30 мин:** Этап 4 - Interpolation
5. **+30 мин:** Этап 5 - Reconciliation

**Итого: ~3 часа чистой работы**

---

## ⚡ Дальнейшие оптимизации (опционально)

### После того как все работает:

1. **Бинарный протокол** (Protocol Buffers / MessagePack)
   - Уменьшение размера на 50-70%
   - ~100 KB/sec → ~30 KB/sec

2. **Interest Management** (Area of Interest)
   - Отправлять только видимые объекты
   - Для 100+ игроков

3. **Adaptive tick rate**
   - Динамическая частота updates в зависимости от нагрузки

4. **Lag compensation**
   - Rewind для hit detection

---

## 🎯 Следующий шаг

Начинаем с **Этапа 1**: Очищаем события и убираем избыточность!

Готов? 🚀
