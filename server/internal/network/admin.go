package network

import (
	"agario-server/internal/bot"
	"agario-server/internal/game"
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type AdminServer struct {
	World      *game.World
	BotManager *bot.BotManager
	Server     *Server
	startTime  time.Time
}

func NewAdminServer(world *game.World, botManager *bot.BotManager, server *Server) *AdminServer {
	return &AdminServer{
		World:      world,
		BotManager: botManager,
		Server:     server,
		startTime:  time.Now(),
	}
}

func (a *AdminServer) Run() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/admin", a.serveAdminPage)
	r.GET("/api/stats", a.getStats)
	r.GET("/api/ws", a.statsWebSocket)
	r.POST("/api/bots/add", a.addBots)
	r.POST("/api/bots/remove", a.removeBots)
	r.POST("/api/player/kick/:id", a.kickPlayer)
	r.POST("/api/food/spawn", a.spawnFood)
	r.POST("/api/gc", a.forceGC)

	log.Println("[ADMIN] Admin panel: http://localhost:8091/admin")
	go r.Run(":8091")
}

func (a *AdminServer) serveAdminPage(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.String(200, adminHTML)
}

func (a *AdminServer) getStats(c *gin.Context) {
	stats := a.collectStats()
	c.JSON(200, stats)
}

func (a *AdminServer) statsWebSocket(c *gin.Context) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		stats := a.collectStats()
		data, _ := json.Marshal(stats)
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			break
		}
	}
}

func (a *AdminServer) collectStats() map[string]interface{} {
	a.World.Mu.RLock()
	defer a.World.Mu.RUnlock()

	playerCount := 0
	botCount := 0
	totalMass := 0.0
	cellCount := 0

	for _, p := range a.World.Players {
		if p.IsBot {
			botCount++
		} else {
			playerCount++
		}
		p.Mu.RLock()
		cellCount += len(p.Cells)
		totalMass += p.TotalMass()
		p.Mu.RUnlock()
	}

	// Собираем статистику памяти
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Количество активных WebSocket соединений
	a.Server.mu.RLock()
	activeConnections := len(a.Server.Clients)
	a.Server.mu.RUnlock()

	return map[string]interface{}{
		"players":   playerCount,
		"bots":      botCount,
		"food":      len(a.World.Food),
		"cells":     cellCount,
		"totalMass": int(totalMass),
		"worldSize": map[string]float64{
			"width":  game.WorldWidth,
			"height": game.WorldHeight,
		},
		"performance": map[string]interface{}{
			"goroutines":    runtime.NumGoroutine(),
			"memoryMB":      m.Alloc / 1024 / 1024,
			"memoryTotalMB": m.TotalAlloc / 1024 / 1024,
			"gcRuns":        m.NumGC,
			"connections":   activeConnections,
		},
		"uptime": int(time.Since(a.startTime).Seconds()),
	}
}

func (a *AdminServer) addBots(c *gin.Context) {
	count, _ := strconv.Atoi(c.DefaultQuery("count", "5"))
	
	// Добавляем ботов с локом
	for i := 0; i < count; i++ {
		name := "Bot" + strconv.Itoa(int(time.Now().UnixNano()%100000)+i)
		newBot := bot.NewBot(name, a.World)
		a.BotManager.Bots = append(a.BotManager.Bots, newBot)
	}
	a.BotManager.MaxBots += count

	c.JSON(200, gin.H{"success": true, "added": count, "total": len(a.BotManager.Bots)})
}

func (a *AdminServer) removeBots(c *gin.Context) {
	count, _ := strconv.Atoi(c.DefaultQuery("count", "5"))
	removed := 0

	a.World.Mu.Lock()
	for i := len(a.BotManager.Bots) - 1; i >= 0 && removed < count; i-- {
		bot := a.BotManager.Bots[i]
		delete(a.World.Players, bot.Player.ID)
		a.BotManager.Bots = append(a.BotManager.Bots[:i], a.BotManager.Bots[i+1:]...)
		removed++
	}
	if a.BotManager.MaxBots > removed {
		a.BotManager.MaxBots -= removed
	}
	a.World.Mu.Unlock()

	c.JSON(200, gin.H{"success": true, "removed": removed, "total": len(a.BotManager.Bots)})
}

func (a *AdminServer) kickPlayer(c *gin.Context) {
	playerID := c.Param("id")
	
	a.World.Mu.Lock()
	delete(a.World.Players, playerID)
	a.World.Mu.Unlock()

	c.JSON(200, gin.H{"success": true})
}

func (a *AdminServer) spawnFood(c *gin.Context) {
	count, _ := strconv.Atoi(c.DefaultQuery("count", "100"))
	
	a.World.Mu.Lock()
	for i := 0; i < count; i++ {
		a.World.SpawnFoodUnlocked()
	}
	a.World.Mu.Unlock()

	c.JSON(200, gin.H{"success": true, "spawned": count})
}

func (a *AdminServer) forceGC(c *gin.Context) {
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	before := m1.Alloc / 1024 / 1024

	runtime.GC()

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	after := m2.Alloc / 1024 / 1024

	c.JSON(200, gin.H{
		"success": true,
		"beforeMB": before,
		"afterMB": after,
		"freedMB": before - after,
	})
}

const adminHTML = `<!DOCTYPE html>
<html><head>
<meta charset="UTF-8">
<title>Agario Admin Panel</title>
<style>
body{font-family:Arial;margin:20px;background:#1a1a1a;color:#fff}
.panel{background:#2a2a2a;padding:20px;margin:10px;border-radius:8px;box-shadow:0 4px 6px rgba(0,0,0,0.3)}
h2{color:#4ECDC4;margin-top:0}
button{background:#4ECDC4;color:#000;border:none;padding:10px 20px;margin:5px;cursor:pointer;border-radius:5px;font-weight:bold;transition:all 0.3s}
button:hover{background:#45B7D1;transform:translateY(-2px)}
button.danger{background:#ff6b6b}
button.danger:hover{background:#ff5252}
.stat{display:inline-block;margin:10px 20px;font-size:16px}
.stat-label{color:#aaa;display:block;font-size:12px}
.stat-value{color:#4ECDC4;font-size:28px;font-weight:bold;display:block}
.warning{color:#ffeb3b}
.danger-value{color:#ff6b6b}
input{padding:8px;margin:5px;border-radius:5px;border:1px solid #444;background:#333;color:#fff}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:20px}
.uptime{color:#98D8C8;font-size:14px;margin-top:10px}
</style>
</head><body>
<h1>🎮 Agario Admin Panel</h1>
<div class="uptime">Uptime: <span id="uptime">0s</span></div>

<div class="panel">
<h2>📊 Real-time Statistics</h2>
<div class="grid">
  <div class="stat">
    <span class="stat-label">Players</span>
    <span id="players" class="stat-value">0</span>
  </div>
  <div class="stat">
    <span class="stat-label">Bots</span>
    <span id="bots" class="stat-value">0</span>
  </div>
  <div class="stat">
    <span class="stat-label">Food</span>
    <span id="food" class="stat-value">0</span>
  </div>
  <div class="stat">
    <span class="stat-label">Cells</span>
    <span id="cells" class="stat-value">0</span>
  </div>
  <div class="stat">
    <span class="stat-label">Total Mass</span>
    <span id="mass" class="stat-value">0</span>
  </div>
</div>
</div>

<div class="panel">
<h2>⚡ Performance Metrics</h2>
<div class="grid">
  <div class="stat">
    <span class="stat-label">Memory Usage</span>
    <span id="memory" class="stat-value">0 MB</span>
  </div>
  <div class="stat">
    <span class="stat-label">Total Allocated</span>
    <span id="memoryTotal" class="stat-value">0 MB</span>
  </div>
  <div class="stat">
    <span class="stat-label">Goroutines</span>
    <span id="goroutines" class="stat-value">0</span>
  </div>
  <div class="stat">
    <span class="stat-label">GC Runs</span>
    <span id="gc" class="stat-value">0</span>
  </div>
  <div class="stat">
    <span class="stat-label">WebSocket Connections</span>
    <span id="connections" class="stat-value">0</span>
  </div>
</div>
<button onclick="forceGC()" style="margin-top:15px">🗑️ Force Garbage Collection</button>
</div>

<div class="panel">
<h2>🤖 Bot Management</h2>
<button onclick="addBots(1)">+1 Bot</button>
<button onclick="addBots(5)">+5 Bots</button>
<button onclick="addBots(10)">+10 Bots</button>
<button onclick="addBots(20)">+20 Bots</button>
<br>
<button class="danger" onclick="removeBots(1)">-1 Bot</button>
<button class="danger" onclick="removeBots(5)">-5 Bots</button>
<button class="danger" onclick="removeBots(10)">-10 Bots</button>
<button class="danger" onclick="removeBots(20)">-20 Bots</button>
</div>

<div class="panel">
<h2>🍕 Food Management</h2>
<button onclick="spawnFood(100)">+100 Food</button>
<button onclick="spawnFood(500)">+500 Food</button>
<button onclick="spawnFood(1000)">+1000 Food</button>
<button onclick="spawnFood(5000)">+5000 Food</button>
</div>

<script>
const ws = new WebSocket('ws://localhost:8091/api/ws');
ws.onmessage = (e) => {
  const data = JSON.parse(e.data);
  document.getElementById('players').textContent = data.players;
  document.getElementById('bots').textContent = data.bots;
  document.getElementById('food').textContent = data.food;
  document.getElementById('cells').textContent = data.cells || 0;
  document.getElementById('mass').textContent = data.totalMass;
  
  const perf = data.performance || {};
  document.getElementById('memory').textContent = perf.memoryMB + ' MB';
  document.getElementById('memoryTotal').textContent = perf.memoryTotalMB + ' MB';
  document.getElementById('goroutines').textContent = perf.goroutines;
  document.getElementById('gc').textContent = perf.gcRuns;
  document.getElementById('connections').textContent = perf.connections;
  
  // Предупреждения
  const memEl = document.getElementById('memory');
  if (perf.memoryMB > 200) {
    memEl.className = 'stat-value danger-value';
  } else if (perf.memoryMB > 100) {
    memEl.className = 'stat-value warning';
  } else {
    memEl.className = 'stat-value';
  }
  
  const gorEl = document.getElementById('goroutines');
  if (perf.goroutines > 100) {
    gorEl.className = 'stat-value warning';
  } else {
    gorEl.className = 'stat-value';
  }
  
  // Uptime
  const uptime = data.uptime || 0;
  const hours = Math.floor(uptime / 3600);
  const mins = Math.floor((uptime % 3600) / 60);
  const secs = uptime % 60;
  document.getElementById('uptime').textContent = 
    (hours > 0 ? hours + 'h ' : '') + 
    (mins > 0 ? mins + 'm ' : '') + 
    secs + 's';
};

ws.onerror = () => {
  document.body.innerHTML = '<h1 style="color:#ff6b6b">❌ Connection Failed</h1><p>Make sure the server is running on port 8091</p>';
};

function addBots(n) { 
  fetch('/api/bots/add?count='+n, {method:'POST'})
    .then(r=>r.json())
    .then(d=>console.log('✅ Added',d.added,'bots. Total:',d.total)); 
}

function removeBots(n) { 
  fetch('/api/bots/remove?count='+n, {method:'POST'})
    .then(r=>r.json())
    .then(d=>console.log('✅ Removed',d.removed,'bots. Total:',d.total)); 
}

function spawnFood(n) { 
  fetch('/api/food/spawn?count='+n, {method:'POST'})
    .then(r=>r.json())
    .then(d=>console.log('✅ Spawned',d.spawned,'food')); 
}

function forceGC() {
  fetch('/api/gc', {method:'POST'})
    .then(r=>r.json())
    .then(d=>{
      console.log('🗑️ GC: Freed',d.freedMB,'MB');
      alert('Garbage Collection\\nBefore: '+d.beforeMB+'MB\\nAfter: '+d.afterMB+'MB\\nFreed: '+d.freedMB+'MB');
    });
}
</script>
</body></html>`
