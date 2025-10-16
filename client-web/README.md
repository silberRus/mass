# Agario Client (Web)

SolidJS-based web client for Agario clone.

## Установка

```bash
cd client-web
npm install
```

## Запуск

```bash
npm run dev
```

Откройте http://localhost:3003 в браузере.

## Требования

- Node.js 18+
- Запущенный сервер на `ws://localhost:8090/ws`

## Управление

- **Мышь** - Направление движения
- **Пробел** - Разделиться (split)
- **W** - Выбросить массу (eject)

## Структура проекта

```
src/
├── components/      # SolidJS компоненты
│   └── Game.tsx    # Главный игровой компонент
├── game/           # Игровой движок
│   └── renderer.ts # Canvas рендеринг
├── network/        # Сеть
│   ├── client.ts   # WebSocket клиент
│   └── protocol.ts # Типы протокола
└── index.tsx       # Точка входа
```

## Особенности

- Client-side рендеринг на Canvas
- Плавная интерполация камеры
- Zoom зависит от размера игрока
- Real-time обновления (30 FPS с сервера)
- Автоматическое переподключение

## Сборка для продакшена

```bash
npm run build
```

Результат будет в папке `dist/`
