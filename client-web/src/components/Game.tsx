import { createSignal, onMount, onCleanup, Show } from 'solid-js';
import { GameClient } from '../network/client';
import { GameRenderer } from '../game/renderer';
import { GameStateManager } from '../game/StateManager';
import { InitData } from '../network/protocol';

export default function Game() {
  const [connected, setConnected] = createSignal(false);
  const [playerName, setPlayerName] = createSignal('');
  const [showJoin, setShowJoin] = createSignal(true);
  const [error, setError] = createSignal('');
  const [playerId, setPlayerId] = createSignal<string | null>(null);

  let canvasRef: HTMLCanvasElement | undefined;
  let client: GameClient | null = null;
  let renderer: GameRenderer | null = null;
  let stateManager: GameStateManager | null = null;

  const WS_URL = 'ws://localhost:8090/ws';

  const handleJoin = async () => {
    const name = playerName().trim();
    if (!name) {
      setError('Please enter your name');
      return;
    }

    console.log('[JOIN] Starting join process with name:', name);

    try {
      setError('');
      client = new GameClient(WS_URL);
      stateManager = new GameStateManager();

      client.setInitHandler((data: InitData) => {
        console.log('Init received:', data);
        renderer = new GameRenderer(canvasRef!, data.worldSize.width, data.worldSize.height);
        renderer.setPlayerId(data.playerId);
        setPlayerId(data.playerId);
        setConnected(true);
        setShowJoin(false);
      });

      client.setStateHandler((message: any) => {
        if (!stateManager) return;
        
        // Event batch
        if (message.type === 'event_batch') {
          stateManager.handleEventBatch(message.events);
        } else {
          // Single event (including world_snapshot)
          stateManager.handleEvent(message);
        }
        
        // Проверяем жив ли игрок
        const player = stateManager.getPlayer(playerId());
        if (!player && playerId()) {
          console.log('[GAME] Player died, cleaning up...');
          setConnected(false);
          setShowJoin(true);
          setPlayerId(null);
          if (client) {
            client.disconnect();
            client = null;
          }
          if (renderer) renderer = null;
          if (stateManager) {
            stateManager.clear();
            stateManager = null;
          }
        }
      });

      await client.connect();
      client.join(name);

    } catch (err) {
      console.error('[JOIN] Error:', err);
      setError('Failed to connect to server. Is it running?');
    }
  };

  onMount(() => {
    if (!canvasRef) return;

    // Обработка мыши
    const handleMouseMove = (e: MouseEvent) => {
      if (!renderer || !client || !connected()) return;
      
      const worldPos = renderer.screenToWorld(e.clientX, e.clientY);
      client.move(worldPos.x, worldPos.y);
    };

    // Обработка клавиатуры
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!client || !connected()) return;

      switch (e.key.toLowerCase()) {
        case ' ':
          e.preventDefault();
          client.split();
          break;
        case 'w':
          e.preventDefault();
          client.eject();
          break;
      }
    };

    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('keydown', handleKeyDown);

    // Game loop для интерполяции
    let lastTime = performance.now();
    let animationFrameId: number;
    
    const gameLoop = () => {
      const now = performance.now();
      const dt = (now - lastTime) / 1000;
      lastTime = now;
      
      if (stateManager && renderer && connected()) {
        stateManager.interpolate(dt);
        renderer.render(stateManager);
      }
      
      animationFrameId = requestAnimationFrame(gameLoop);
    };
    
    gameLoop();

    onCleanup(() => {
      console.log('[CLEANUP] Removing event listeners and disconnecting');
      cancelAnimationFrame(animationFrameId);
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('keydown', handleKeyDown);
      if (client) {
        console.log('[CLEANUP] Disconnecting client');
        client.disconnect();
        client = null;
      }
      if (renderer) {
        console.log('[CLEANUP] Cleaning up renderer');
        renderer = null;
      }
      if (stateManager) {
        stateManager.clear();
        stateManager = null;
      }
    });
  });

  return (
    <div style={{
      width: '100vw',
      height: '100vh',
      margin: 0,
      padding: 0,
      overflow: 'hidden',
      position: 'relative',
    }}>
      <canvas
        ref={canvasRef}
        style={{
          display: 'block',
          cursor: showJoin() ? 'default' : 'none',
        }}
      />

      <Show when={showJoin()}>
        <div style={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          background: 'rgba(0, 0, 0, 0.8)',
          padding: '40px',
          'border-radius': '10px',
          'text-align': 'center',
        }}>
          <h1 style={{ color: '#fff', 'margin-bottom': '20px' }}>Agario Clone</h1>
          
          <input
            type="text"
            placeholder="Enter your name"
            value={playerName()}
            onInput={(e) => setPlayerName(e.currentTarget.value)}
            onKeyPress={(e) => {
              if (e.key === 'Enter') {
                handleJoin();
              }
            }}
            style={{
              padding: '10px',
              'font-size': '16px',
              width: '200px',
              border: '2px solid #4ECDC4',
              'border-radius': '5px',
              'margin-bottom': '15px',
            }}
          />

          <Show when={error()}>
            <div style={{ color: '#ff6b6b', 'margin-bottom': '10px' }}>
              {error()}
            </div>
          </Show>

          <button
            onClick={handleJoin}
            style={{
              padding: '10px 30px',
              'font-size': '16px',
              background: '#4ECDC4',
              color: '#fff',
              border: 'none',
              'border-radius': '5px',
              cursor: 'pointer',
              'font-weight': 'bold',
            }}
          >
            Play
          </button>

          <div style={{
            'margin-top': '20px',
            color: '#aaa',
            'font-size': '14px',
          }}>
            <p>Controls:</p>
            <p>Mouse - Move</p>
            <p>Space - Split</p>
            <p>W - Eject mass</p>
          </div>
        </div>
      </Show>
    </div>
  );
}
