// State Manager - управление состоянием игры на основе событий
export interface Vector2D {
  x: number;
  y: number;
}

export interface Cell {
  id: string;
  x: number;
  y: number;
  radius: number;
  velocity?: Vector2D; // Для split/eject
  targetX?: number;    // Куда движется (от сервера)
  targetY?: number;
  
  // Для плавной интерполяции
  serverX?: number;    // Последняя позиция от сервера
  serverY?: number;
  serverRadius?: number;
  lerpFactor?: number; // Скорость интерполяции к серверной позиции
}

export interface Player {
  id: string;
  name: string;
  color: string;
  isBot: boolean;
  score: number;
  cells: Map<string, Cell>;
}

export interface Food {
  id: string;
  x: number;
  y: number;
  radius: number;
  color: string;
  velocity?: Vector2D;
}

export class GameStateManager {
  private players: Map<string, Player> = new Map();
  private food: Map<string, Food> = new Map();
  private lastUpdateTime: number = 0;

  constructor() {
    console.log('[STATE] GameStateManager initialized');
  }

  // Обработка событий
  handleEvent(event: any) {
    const eventType = event.type;
    const data = event.data;

    console.log(`[STATE] Handling event: ${eventType}`);

    switch (eventType) {
      case 'player_joined':
        this.handlePlayerJoined(data);
        break;
      case 'state_delta':
        this.handleStateDelta(data);
        break;
      case 'player_split':
        this.handlePlayerSplit(data);
        break;
      case 'player_died':
        this.handlePlayerDied(data);
        break;
      case 'food_spawned':
        this.handleFoodSpawned(data);
        break;
      case 'food_eaten':
        this.handleFoodEaten(data);
        break;
      case 'cell_eaten':
        this.handleCellEaten(data);
        break;
      case 'cell_merged':
        this.handleCellMerged(data);
        break;
      case 'world_snapshot':
        this.handleWorldSnapshot(data);
        break;
      default:
        console.warn(`[STATE] Unknown event type: ${eventType}`);
    }
  }

  // Обработка batch событий
  handleEventBatch(events: any[]) {
    for (const event of events) {
      this.handleEvent(event);
    }
  }

  private handlePlayerJoined(data: any) {
    const player: Player = {
      id: data.playerId,
      name: data.name,
      color: data.color,
      isBot: data.isBot,
      score: 0,
      cells: new Map(),
    };

    const cell: Cell = {
      id: data.cellId,
      x: data.x,
      y: data.y,
      radius: data.radius,
    };

    player.cells.set(cell.id, cell);
    this.players.set(player.id, player);
    
    console.log(`[STATE] Player joined: ${player.name} (${player.id})`);
  }

  private handleStateDelta(data: any) {
    // Обновляем позиции клеток из delta (приходит с сервера 10 раз/сек)
    for (const entityDelta of data.entities) {
      // Ищем клетку по ID
      let found = false;
      for (const player of this.players.values()) {
        const cell = player.cells.get(entityDelta.id);
        if (cell) {
          // Сохраняем серверные позиции отдельно
          cell.serverX = entityDelta.x;
          cell.serverY = entityDelta.y;
          cell.serverRadius = entityDelta.radius;
          cell.targetX = entityDelta.targetX;
          cell.targetY = entityDelta.targetY;
          
          // Начинаем плавную интерполяцию
          cell.lerpFactor = 0;
          
          found = true;
          break;
        }
      }
      
      if (!found) {
        // Возможно это еда, но мы её не трекаем через delta
      }
    }
  }

  private handlePlayerSplit(data: any) {
    const player = this.players.get(data.playerId);
    if (!player) return;

    // Добавляем новые клетки
    for (const cellData of data.newCells) {
      const cell: Cell = {
        id: cellData.cellId,
        x: cellData.x,
        y: cellData.y,
        radius: cellData.radius,
        velocity: { x: cellData.velX, y: cellData.velY },
      };
      player.cells.set(cell.id, cell);
    }
  }

  private handlePlayerDied(data: any) {
    this.players.delete(data.playerId);
    console.log(`[STATE] Player died: ${data.playerId}`);
  }

  private handleFoodSpawned(data: any) {
    for (const foodData of data.foods) {
      const food: Food = {
        id: foodData.foodId,
        x: foodData.x,
        y: foodData.y,
        radius: foodData.radius,
        color: foodData.color,
        velocity: foodData.velX ? { x: foodData.velX, y: foodData.velY } : undefined,
      };
      this.food.set(food.id, food);
    }
  }

  private handleFoodEaten(data: any) {
    this.food.delete(data.foodId);
    
    // Обновляем размер клетки (будет в snapshot)
    const player = this.players.get(data.playerId);
    if (player) {
      const cell = player.cells.get(data.cellId);
      if (cell) {
        // Увеличиваем радиус немного для плавности
        cell.radius += 0.5;
      }
    }
  }

  private handleCellEaten(data: any) {
    // Находим игрока у которого съели клетку
    for (const player of this.players.values()) {
      if (player.cells.has(data.eatenCellId)) {
        player.cells.delete(data.eatenCellId);
        
        // Если у игрока не осталось клеток - он мертв
        if (player.cells.size === 0) {
          this.players.delete(player.id);
        }
        break;
      }
    }
  }

  private handleCellMerged(data: any) {
    const player = this.players.get(data.playerId);
    if (!player) return;

    // Удаляем старые клетки
    player.cells.delete(data.cell1Id);
    player.cells.delete(data.cell2Id);

    // Добавляем новую объединенную клетку
    const cell: Cell = {
      id: data.newCellId,
      x: data.x,
      y: data.y,
      radius: data.radius,
    };
    player.cells.set(cell.id, cell);
  }

  private handleWorldSnapshot(data: any) {
    console.log('[STATE] Applying world snapshot');
    
    // Полная синхронизация
    this.players.clear();
    this.food.clear();

    // Загружаем игроков
    for (const playerData of data.players) {
      const player: Player = {
        id: playerData.id,
        name: playerData.name,
        color: playerData.color,
        isBot: playerData.isBot,
        score: playerData.score,
        cells: new Map(),
      };

      for (const cellData of playerData.cells) {
        const cell: Cell = {
          id: cellData.id,
          x: cellData.x,
          y: cellData.y,
          radius: cellData.radius,
        };
        player.cells.set(cell.id, cell);
      }

      this.players.set(player.id, player);
    }

    // Загружаем еду
    for (const foodData of data.food) {
      const food: Food = {
        id: foodData.id,
        x: foodData.x,
        y: foodData.y,
        radius: foodData.radius,
        color: foodData.color,
      };
      this.food.set(food.id, food);
    }
    
    this.lastUpdateTime = data.timestamp;
  }

  // Интерполяция позиций (вызывается каждый кадр)
  interpolate(dt: number) {
    // CLIENT-SIDE PREDICTION с SMOOTH RECONCILIATION:
    // 1. Клиент предсказывает движение через физику
    // 2. Сервер присылает позицию + target 
    // 3. Клиент ПЛАВНО корректирует позицию к серверной
    
    // Симулируем движение к target
    for (const player of this.players.values()) {
      for (const cell of player.cells.values()) {
        // Priority 1: Velocity (для split/eject)
        if (cell.velocity) {
          cell.x += cell.velocity.x * dt;
          cell.y += cell.velocity.y * dt;
          
          cell.velocity.x *= 0.95;
          cell.velocity.y *= 0.95;
          
          if (Math.abs(cell.velocity.x) < 0.1 && Math.abs(cell.velocity.y) < 0.1) {
            cell.velocity = undefined;
          }
        }
        // Priority 2: Предсказание движения через физику
        else if (cell.targetX !== undefined && cell.targetY !== undefined) {
          const dx = cell.targetX - cell.x;
          const dy = cell.targetY - cell.y;
          const distance = Math.sqrt(dx * dx + dy * dy);
          
          if (distance > 1) {
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
        
        // Priority 3: Плавная коррекция к серверной позиции
        if (cell.serverX !== undefined && cell.serverY !== undefined) {
          // Постепенно увеличиваем лерп фактор
          if (cell.lerpFactor !== undefined) {
            cell.lerpFactor = Math.min(1, cell.lerpFactor + dt * 5); // 5 = скорость коррекции
          } else {
            cell.lerpFactor = 0;
          }
          
          // Рассчитываем разницу между предсказанной и серверной позицией
          const errorX = cell.serverX - cell.x;
          const errorY = cell.serverY - cell.y;
          const errorDist = Math.sqrt(errorX * errorX + errorY * errorY);
          
          // Если рассинхронизация большая, корректируем быстрее
          const correctionSpeed = errorDist > 50 ? 8 : 3;
          const lerpSpeed = dt * correctionSpeed;
          
          // Плавная коррекция позиции
          if (errorDist > 1) {
            cell.x += errorX * lerpSpeed;
            cell.y += errorY * lerpSpeed;
          }
          
          // Плавная коррекция радиуса
          if (cell.serverRadius !== undefined) {
            const radiusError = cell.serverRadius - cell.radius;
            cell.radius += radiusError * lerpSpeed;
          }
        }
      }
    }

    // Интерполируем выброшенную еду
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

  // Геттеры
  getPlayers(): Player[] {
    return Array.from(this.players.values());
  }

  getFood(): Food[] {
    return Array.from(this.food.values());
  }

  getPlayer(playerId: string): Player | undefined {
    return this.players.get(playerId);
  }

  clear() {
    this.players.clear();
    this.food.clear();
  }
}
