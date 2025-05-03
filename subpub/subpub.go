package subpub

import (
	"context"
	"errors"
	"sync"
)

type MessageHandler func(msg interface{})

type Subscription interface {
	Unsubscribe()
}

type SubPub interface {
	//subject - событие
	Subscribe(subject string, cb MessageHandler) (Subscription, error)

	Publish(subject string, msg interface{}) error

	Close(ctx context.Context) error
}

func NewSubPub() SubPub {
	return &eventBus{
		events: make(map[string][]uniqueHandler, 0),
	}
}

type subscription struct {
	unsubscribe func()
}

func (sb *subscription) Unsubscribe() {
	sb.unsubscribe()
}

// Уникальный id обработчика для каждого подписчика
type uniqueHandler struct {
	id      uint64
	execute MessageHandler
}

type eventBus struct {
	nextId uint64
	// хэш-таблица со списком хэндлеров для subject
	events   map[string][]uniqueHandler
	mx       sync.RWMutex
	isClosed bool
}

func (eb *eventBus) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	eb.mx.Lock()
	defer eb.mx.Unlock()
	if eb.isClosed {
		return nil, errors.New("bus is closed")
	}
	unique := uniqueHandler{id: eb.nextId, execute: cb}
	eb.nextId += 1
	eb.events[subject] = append(eb.events[subject], unique)
	return &subscription{
		unsubscribe: func() {
			eb.mx.Lock()
			defer eb.mx.Unlock()
			if eb.isClosed {
				return
			}
			for idx, v := range eb.events[subject] {
				if v.id == unique.id {
					eb.events[subject] = append(eb.events[subject][:idx], eb.events[subject][idx+1:]...)
					break
				}
			}
			// Если список пуст, то удаляем ключ
			if len(eb.events[subject]) == 0 {
				delete(eb.events, subject)
			}
		},
	}, nil
}

func (eb *eventBus) Publish(subject string, msg interface{}) error {
	eb.mx.RLock()
	defer eb.mx.RUnlock()
	if eb.isClosed {
		return errors.New("bus is closed")
	}
	handlers, isExists := eb.events[subject]
	if !isExists {
		return errors.New("event isn't found")
	}
	copyHandlers := make([]uniqueHandler, len(handlers))
	copy(copyHandlers, handlers)
	go func() {
		for _, unique := range copyHandlers {
			unique.execute(msg)
		}
	}()
	return nil
}

func (eb *eventBus) Close(ctx context.Context) error {
	eb.mx.Lock()
	defer eb.mx.Unlock()
	if eb.isClosed {
		return errors.New("bus is already closed")
	}
	done := make(chan struct{})
	select {
	case <-ctx.Done():
		eb.closeAndClear()
		return ctx.Err()
	default:
		go func() {
			select {
			case <-ctx.Done():
				eb.mx.Lock()
				defer eb.mx.Unlock()
				eb.closeAndClear()
				close(done)
			case <-done:
				return
			}
		}()
	}
	return nil
}

func (eb *eventBus) closeAndClear() {
	eb.isClosed = true
	eb.events = nil
	eb.nextId = 0
}
