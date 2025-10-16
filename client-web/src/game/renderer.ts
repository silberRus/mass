import { GameStateManager, Player, Food } from './StateManager';

export interface Camera {
  x: number;
  y: number;
  zoom: number;
}

export class GameRenderer {
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;
  private camera: Camera;
  private worldWidth: number;
  private worldHeight: number;
  private playerId: string | null = null;
  private targetX: number = 0;
  private targetY: number = 0;

  constructor(canvas: HTMLCanvasElement, worldWidth: number, worldHeight: number) {
    this.canvas = canvas;
    this.ctx = canvas.getContext('2d')!;
    this.worldWidth = worldWidth;
    this.worldHeight = worldHeight;
    
    this.camera = {
      x: worldWidth / 2,
      y: worldHeight / 2,
      zoom: 1,
    };

    this.resizeCanvas();
    window.addEventListener('resize', () => this.resizeCanvas());
  }

  private resizeCanvas() {
    this.canvas.width = window.innerWidth;
    this.canvas.height = window.innerHeight;
  }

  setPlayerId(playerId: string) {
    this.playerId = playerId;
  }

  render(stateManager: GameStateManager) {
    this.clear();
    
    // Обновляем камеру чтобы следовать за игроком
    if (this.playerId) {
      this.updateCamera(stateManager.getPlayers());
    }

    this.ctx.save();
    
    // Применяем трансформацию камеры
    this.ctx.translate(this.canvas.width / 2, this.canvas.height / 2);
    this.ctx.scale(this.camera.zoom, this.camera.zoom);
    this.ctx.translate(-this.camera.x, -this.camera.y);

    // Рисуем сетку
    this.drawGrid();
    
    // Рисуем еду
    stateManager.getFood().forEach(food => this.drawFood(food));
    
    // Рисуем игроков
    stateManager.getPlayers().forEach(player => this.drawPlayer(player));

    // Рисуем курсор/направление
    if (this.playerId) {
      this.drawCursor(stateManager.getPlayers());
    }

    this.ctx.restore();

    // Рисуем UI
    this.drawUI(stateManager);
  }

  private clear() {
    this.ctx.fillStyle = '#f0f0f0';
    this.ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);
  }

  private drawGrid() {
    const gridSize = 50;
    const startX = Math.floor(this.camera.x / gridSize) * gridSize;
    const startY = Math.floor(this.camera.y / gridSize) * gridSize;
    
    const endX = startX + this.canvas.width / this.camera.zoom + gridSize * 2;
    const endY = startY + this.canvas.height / this.camera.zoom + gridSize * 2;

    this.ctx.strokeStyle = '#e0e0e0';
    this.ctx.lineWidth = 1 / this.camera.zoom;

    for (let x = startX; x < endX; x += gridSize) {
      this.ctx.beginPath();
      this.ctx.moveTo(x, startY);
      this.ctx.lineTo(x, endY);
      this.ctx.stroke();
    }

    for (let y = startY; y < endY; y += gridSize) {
      this.ctx.beginPath();
      this.ctx.moveTo(startX, y);
      this.ctx.lineTo(endX, y);
      this.ctx.stroke();
    }

    // Рисуем границы мира
    this.ctx.strokeStyle = '#333';
    this.ctx.lineWidth = 5 / this.camera.zoom;
    this.ctx.strokeRect(0, 0, this.worldWidth, this.worldHeight);
  }

  private drawFood(food: Food) {
    this.ctx.fillStyle = food.color;
    this.ctx.beginPath();
    this.ctx.arc(food.x, food.y, food.radius, 0, Math.PI * 2);
    this.ctx.fill();
  }

  private drawPlayer(player: Player) {
    for (const cell of player.cells.values()) {
      this.drawCell(cell, player);
    }
  }

  private drawCell(cell: any, player: Player) {
    const isOwnCell = player.id === this.playerId;

    // Основной круг
    this.ctx.fillStyle = player.color;
    this.ctx.beginPath();
    this.ctx.arc(cell.x, cell.y, cell.radius, 0, Math.PI * 2);
    this.ctx.fill();

    // Обводка
    this.ctx.strokeStyle = isOwnCell ? '#fff' : '#333';
    this.ctx.lineWidth = (isOwnCell ? 4 : 2) / this.camera.zoom;
    this.ctx.stroke();

    // Имя игрока
    if (cell.radius > 15) {
      const fontSize = Math.max(12, cell.radius / 3);
      this.ctx.font = `bold ${fontSize}px Arial`;
      this.ctx.fillStyle = '#fff';
      this.ctx.textAlign = 'center';
      this.ctx.textBaseline = 'middle';
      this.ctx.strokeStyle = '#000';
      this.ctx.lineWidth = 3 / this.camera.zoom;
      
      const name = player.isBot ? `🤖 ${player.name}` : player.name;
      this.ctx.strokeText(name, cell.x, cell.y);
      this.ctx.fillText(name, cell.x, cell.y);
      
      // Масса
      const mass = Math.floor((cell.radius * cell.radius) / 100);
      const massText = `${mass}`;
      this.ctx.font = `${fontSize * 0.7}px Arial`;
      this.ctx.strokeText(massText, cell.x, cell.y + fontSize);
      this.ctx.fillText(massText, cell.x, cell.y + fontSize);
    }
  }

  private updateCamera(players: Player[]) {
    const player = players.find(p => p.id === this.playerId);
    if (!player || player.cells.size === 0) return;

    // Находим центр масс всех клеток игрока
    let totalX = 0;
    let totalY = 0;
    let totalMass = 0;

    for (const cell of player.cells.values()) {
      const mass = cell.radius * cell.radius;
      totalX += cell.x * mass;
      totalY += cell.y * mass;
      totalMass += mass;
    }

    const centerX = totalX / totalMass;
    const centerY = totalY / totalMass;

    // Плавное движение камеры
    const smoothing = 0.1;
    this.camera.x += (centerX - this.camera.x) * smoothing;
    this.camera.y += (centerY - this.camera.y) * smoothing;

    // Зум зависит от размера игрока
    const avgRadius = Math.sqrt(totalMass / player.cells.size);
    const targetZoom = Math.max(0.3, Math.min(1.5, 1000 / (avgRadius + 500)));
    this.camera.zoom += (targetZoom - this.camera.zoom) * smoothing;
  }

  private drawUI(stateManager: GameStateManager) {
    const players = stateManager.getPlayers();
    
    // Топ-5 игроков
    const sorted = [...players]
      .sort((a, b) => b.score - a.score)
      .slice(0, 5);

    this.ctx.fillStyle = 'rgba(0, 0, 0, 0.5)';
    this.ctx.fillRect(10, 10, 200, 30 + sorted.length * 25);

    this.ctx.font = 'bold 16px Arial';
    this.ctx.fillStyle = '#fff';
    this.ctx.textAlign = 'left';
    this.ctx.fillText('Leaderboard', 20, 30);

    this.ctx.font = '14px Arial';
    sorted.forEach((player, i) => {
      const y = 55 + i * 25;
      const isMe = player.id === this.playerId;
      this.ctx.fillStyle = isMe ? '#ffeb3b' : '#fff';
      const name = player.isBot ? `🤖 ${player.name}` : player.name;
      this.ctx.fillText(`${i + 1}. ${name}: ${player.score}`, 20, y);
    });

    // Счет игрока
    const player = players.find(p => p.id === this.playerId);
    if (player) {
      this.ctx.fillStyle = 'rgba(0, 0, 0, 0.5)';
      this.ctx.fillRect(this.canvas.width - 160, 10, 150, 50);
      
      this.ctx.font = 'bold 16px Arial';
      this.ctx.fillStyle = '#fff';
      this.ctx.textAlign = 'right';
      this.ctx.fillText(`Score: ${player.score}`, this.canvas.width - 20, 35);
      this.ctx.fillText(`Cells: ${player.cells.size}`, this.canvas.width - 20, 55);
    }

    // Подсказки управления
    this.ctx.fillStyle = 'rgba(0, 0, 0, 0.5)';
    this.ctx.fillRect(10, this.canvas.height - 80, 200, 70);
    
    this.ctx.font = '12px Arial';
    this.ctx.fillStyle = '#fff';
    this.ctx.textAlign = 'left';
    this.ctx.fillText('Mouse: Move', 20, this.canvas.height - 60);
    this.ctx.fillText('Space: Split', 20, this.canvas.height - 40);
    this.ctx.fillText('W: Eject mass', 20, this.canvas.height - 20);
  }

  screenToWorld(screenX: number, screenY: number): { x: number; y: number } {
    const x = (screenX - this.canvas.width / 2) / this.camera.zoom + this.camera.x;
    const y = (screenY - this.canvas.height / 2) / this.camera.zoom + this.camera.y;
    this.targetX = x;
    this.targetY = y;
    return { x, y };
  }

  getTarget(): { x: number; y: number } {
    return { x: this.targetX, y: this.targetY };
  }

  private drawCursor(players: Player[]) {
    const player = players.find(p => p.id === this.playerId);
    if (!player || player.cells.size === 0) return;

    let centerX = 0;
    let centerY = 0;
    for (const cell of player.cells.values()) {
      centerX += cell.x;
      centerY += cell.y;
    }
    centerX /= player.cells.size;
    centerY /= player.cells.size;

    // Курсор всегда впереди от игрока (минимум 300 единиц)
    const dx = this.targetX - centerX;
    const dy = this.targetY - centerY;
    const distance = Math.sqrt(dx * dx + dy * dy);
    
    const minDistance = 300;
    if (distance < minDistance) {
      const dir = distance > 0 ? { x: dx / distance, y: dy / distance } : { x: 1, y: 0 };
      this.targetX = centerX + dir.x * minDistance;
      this.targetY = centerY + dir.y * minDistance;
    }

    // Черная пунктирная линия
    this.ctx.strokeStyle = 'rgba(0, 0, 0, 0.6)';
    this.ctx.lineWidth = 4 / this.camera.zoom;
    this.ctx.setLineDash([10 / this.camera.zoom, 10 / this.camera.zoom]);
    this.ctx.beginPath();
    this.ctx.moveTo(centerX, centerY);
    this.ctx.lineTo(this.targetX, this.targetY);
    this.ctx.stroke();
    this.ctx.setLineDash([]);

    // Курсор-кольцо
    this.ctx.fillStyle = 'rgba(255, 0, 0, 0.3)';
    this.ctx.strokeStyle = '#000';
    this.ctx.lineWidth = 4 / this.camera.zoom;
    this.ctx.beginPath();
    this.ctx.arc(this.targetX, this.targetY, 20 / this.camera.zoom, 0, Math.PI * 2);
    this.ctx.fill();
    this.ctx.stroke();

    this.ctx.strokeStyle = '#fff';
    this.ctx.lineWidth = 2 / this.camera.zoom;
    this.ctx.beginPath();
    this.ctx.arc(this.targetX, this.targetY, 15 / this.camera.zoom, 0, Math.PI * 2);
    this.ctx.stroke();

    // Стрелка
    const finalDist = Math.sqrt((this.targetX - centerX) ** 2 + (this.targetY - centerY) ** 2);
    if (finalDist > 50) {
      const angle = Math.atan2(this.targetY - centerY, this.targetX - centerX);
      const arrowSize = 15 / this.camera.zoom;
      
      this.ctx.fillStyle = '#000';
      this.ctx.beginPath();
      this.ctx.moveTo(
        this.targetX + Math.cos(angle) * (arrowSize + 2),
        this.targetY + Math.sin(angle) * (arrowSize + 2)
      );
      this.ctx.lineTo(
        this.targetX + Math.cos(angle + 2.5) * (arrowSize * 0.6 + 2),
        this.targetY + Math.sin(angle + 2.5) * (arrowSize * 0.6 + 2)
      );
      this.ctx.lineTo(
        this.targetX + Math.cos(angle - 2.5) * (arrowSize * 0.6 + 2),
        this.targetY + Math.sin(angle - 2.5) * (arrowSize * 0.6 + 2)
      );
      this.ctx.closePath();
      this.ctx.fill();

      this.ctx.fillStyle = '#ff0000';
      this.ctx.beginPath();
      this.ctx.moveTo(
        this.targetX + Math.cos(angle) * arrowSize,
        this.targetY + Math.sin(angle) * arrowSize
      );
      this.ctx.lineTo(
        this.targetX + Math.cos(angle + 2.5) * arrowSize * 0.6,
        this.targetY + Math.sin(angle + 2.5) * arrowSize * 0.6
      );
      this.ctx.lineTo(
        this.targetX + Math.cos(angle - 2.5) * arrowSize * 0.6,
        this.targetY + Math.sin(angle - 2.5) * arrowSize * 0.6
      );
      this.ctx.closePath();
      this.ctx.fill();
    }
  }
}
