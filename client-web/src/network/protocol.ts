// Типы протокола (синхронизированы с сервером)

export type MessageType =
  | 'join'
  | 'move'
  | 'split'
  | 'eject'
  | 'init'
  | 'state'
  | 'player_died'
  | 'leaderboard';

export interface ClientMessage {
  type: MessageType;
  data: any;
}

export interface ServerMessage {
  type: MessageType;
  data: any;
}

// Client -> Server
export interface JoinData {
  name: string;
}

export interface MoveData {
  x: number;
  y: number;
}

// Server -> Client
export interface InitData {
  playerId: string;
  worldSize: WorldSize;
}

export interface WorldSize {
  width: number;
  height: number;
}

export interface StateData {
  timestamp: number;
  players: PlayerState[];
  food: FoodState[];
}

export interface PlayerState {
  id: string;
  name: string;
  cells: CellState[];
  score: number;
  isBot: boolean;
}

export interface CellState {
  x: number;
  y: number;
  radius: number;
  color: string;
}

export interface FoodState {
  id: string;
  x: number;
  y: number;
  color: string;
  radius: number;
}

export interface PlayerDiedData {
  playerId: string;
  killerId?: string;
}

export interface LeaderboardData {
  leaders: LeaderEntry[];
}

export interface LeaderEntry {
  name: string;
  score: number;
}
