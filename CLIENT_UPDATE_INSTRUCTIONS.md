# 🎯 ЭТАП 3: Клиентская часть (State Delta + Client Physics)

## Что нужно сделать:

### 1. Обновить StateManager для state_delta

**Файл:** `client-web/src/game/StateManager.ts`

Добавить обработку нового события:

```typescript
case 'state_delta':
  this.handleStateDelta(data);
  break;
```

И метод:

```typescript
private handleStateDelta(data: any) {
  // Обновляем позиции клеток из delta
  for (const entityDelta of data.entities) {
    // Ищем клетку по ID
    for (const player of this.players.values()) {
      const cell = player.cells.get(entityDelta.id);
      if (cell) {
        // Обновляем позицию и радиус
        cell.x = entityDelta.x;
        cell.y = entityDelta.y;
        cell.radius = entityDelta.radius;
        break;
      }
    }
  }
}
```

### 2. Убрать старые обработчики

Удалить:
- `handlePlayerMoved()`
- `handleCellUpdated()`

### 3. Упростить interpolate

State delta уже дает нам актуальные позиции 10 раз/сек.
Клиенту НЕ нужно симулировать физику между updates!

```typescript
interpolate(dt: number) {
  // ТОЛЬКО velocity для split/eject!
  for (const player of this.players.values()) {
    for (const cell of player.cells.values()) {
      if (cell.velocity) {
        cell.x += cell.velocity.x * dt;
        cell.y += cell.velocity.y * dt;
        
        cell.velocity.x *= 0.95;
        cell.velocity.y *= 0.95;
        
        if (Math.abs(cell.velocity.x) < 0.1 && Math.abs(cell.velocity.y) < 0.1) {
          cell.velocity = undefined;
        }
      }
    }
  }
  
  // То же для еды
  for (const food of this.food.values()) {
    if (food.velocity) {
      food.x += food.velocity.x * dt;
      food.y += food.velocity.y * dt;
      
      food.velocity.x *= 0.95;
      food.velocity.y *= 0.95;
      
      if (Math.abs(food.velocity.x) < 0.1 && Math.abs(food.velocity.y) < 0.1) {
        food.velocity = undefined;
      }
    }
  }
}
```

---

## 🎮 Результат:

После этих изменений:
- ✅ Клиент получает позиции прямо с сервера (10 раз/сек)
- ✅ Нет экстраполяции → нет рассинхронизации!
- ✅ Snapshot каждые 10 сек корректирует мелкие ошибки
- ✅ Плавное движение за счет 60 fps рендеринга

---

## Хочешь чтобы я сделал эти изменения?

Напиши "продолжай" и я обновлю клиент!
