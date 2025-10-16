package events

import (
	"encoding/json"
	"log"
	"sync"
)

// EventHandler - функция-обработчик события
type EventHandler func(*Event)

// EventBus - шина событий
type EventBus struct {
	handlers map[EventType][]EventHandler
	mu       sync.RWMutex
	
	// Буфер событий для батчинга
	eventBuffer []*Event
	bufferMu    sync.Mutex
}

// NewEventBus - создать новую шину событий
func NewEventBus() *EventBus {
	return &EventBus{
		handlers:    make(map[EventType][]EventHandler),
		eventBuffer: make([]*Event, 0, 100),
	}
}

// Subscribe - подписаться на событие
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Publish - опубликовать событие
func (eb *EventBus) Publish(event *Event) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()
	
	// Вызываем обработчики
	for _, handler := range handlers {
		go handler(event)
	}
	
	// Добавляем в буфер для батчинга
	eb.bufferMu.Lock()
	eb.eventBuffer = append(eb.eventBuffer, event)
	eb.bufferMu.Unlock()
}

// PublishEvent - создать и опубликовать событие
func (eb *EventBus) PublishEvent(eventType EventType, data interface{}) {
	event := NewEvent(eventType, data)
	eb.Publish(event)
}

// FlushEvents - получить и очистить буфер событий
func (eb *EventBus) FlushEvents() []*Event {
	eb.bufferMu.Lock()
	defer eb.bufferMu.Unlock()
	
	if len(eb.eventBuffer) == 0 {
		return nil
	}
	
	// Копируем события
	events := make([]*Event, len(eb.eventBuffer))
	copy(events, eb.eventBuffer)
	
	// Очищаем буфер
	eb.eventBuffer = eb.eventBuffer[:0]
	
	return events
}

// SerializeEvents - сериализовать события в JSON
func (eb *EventBus) SerializeEvents(events []*Event) ([]byte, error) {
	if len(events) == 0 {
		return nil, nil
	}
	
	// Создаем batch сообщение
	batch := map[string]interface{}{
		"type":   "event_batch",
		"events": events,
	}
	
	data, err := json.Marshal(batch)
	if err != nil {
		log.Printf("[EVENT_BUS] Error serializing events: %v", err)
		return nil, err
	}
	
	return data, nil
}

// GetBufferSize - размер текущего буфера
func (eb *EventBus) GetBufferSize() int {
	eb.bufferMu.Lock()
	defer eb.bufferMu.Unlock()
	return len(eb.eventBuffer)
}
