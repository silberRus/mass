import {
  ClientMessage,
  ServerMessage,
  InitData,
  StateData,
  JoinData,
  MoveData,
} from './protocol';

export type GameStateHandler = (message: any) => void;
export type InitHandler = (data: InitData) => void;

export class GameClient {
  private ws: WebSocket | null = null;
  private url: string;
  private onStateUpdate?: GameStateHandler;
  private onInit?: InitHandler;

  constructor(url: string) {
    this.url = url;
  }

  connect(): Promise<void> {
    console.log('[WS] Starting connection to:', this.url);
    
    // ВАЖНО: закрываем старое соединение если оно есть
    if (this.ws) {
      console.log('[WS] Closing existing connection before creating new one');
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
    }
    
    return new Promise((resolve, reject) => {
      try {
        console.log('[WS] Creating WebSocket...');
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
          console.log('[WS] ✅ Connected to server');
          resolve();
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };

        this.ws.onerror = (error) => {
          console.error('[WS] ❌ WebSocket error:', error);
          reject(error);
        };

        this.ws.onclose = (event) => {
          console.log('[WS] Connection closed. Code:', event.code, 'Reason:', event.reason);
          this.ws = null;
        };
      } catch (error) {
        console.error('[WS] ❌ Failed to create WebSocket:', error);
        reject(error);
      }
    });
  }

  private handleMessage(data: string) {
    try {
      const message = JSON.parse(data);

      // Обработка event batch
      if (message.type === 'event_batch') {
        if (this.onStateUpdate) {
          this.onStateUpdate(message);
        }
        return;
      }

      // Обработка одиночных событий (включая world_snapshot)
      if (message.type && message.type !== 'init') {
        if (this.onStateUpdate) {
          this.onStateUpdate(message);
        }
        return;
      }

      // Обработка init сообщения
      if (message.type === 'init') {
        if (this.onInit) {
          this.onInit(message.data as InitData);
        }
        return;
      }

    } catch (error) {
      console.error('[WS] Error parsing message:', error);
    }
  }

  private send(message: ClientMessage) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      const data = JSON.stringify(message);
      this.ws.send(data);
    } else {
      console.warn('[WS] ⚠️ Cannot send - WebSocket not connected');
    }
  }

  join(name: string) {
    console.log('[WS] Sending join request for:', name);
    const data: JoinData = { name };
    this.send({ type: 'join', data });
  }

  move(x: number, y: number) {
    const data: MoveData = { x, y };
    this.send({ type: 'move', data });
  }

  split() {
    this.send({ type: 'split', data: null });
  }

  eject() {
    this.send({ type: 'eject', data: null });
  }

  setStateHandler(handler: GameStateHandler) {
    this.onStateUpdate = handler;
  }

  setInitHandler(handler: InitHandler) {
    this.onInit = handler;
  }

  disconnect() {
    console.log('[WS] Explicit disconnect called');
    
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.onerror = null;
      this.ws.onmessage = null;
      this.ws.onopen = null;
      
      if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
        console.log('[WS] Closing WebSocket connection');
        this.ws.close(1000, 'Client disconnect');
      }
      
      this.ws = null;
    }
    
    console.log('[WS] Disconnect complete');
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}
