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
	// return &eventBus{
	// 	events: make(map[string][]MessageHandler, 0),
	// }
	panic("")
}

type subscription struct {
	unsubscribe func()
}

func (sb *subscription) Unsubscribe() {
	sb.unsubscribe()
}

type eventBus struct {
	// хэш-таблица со списком хэндлеров для subject
	events   map[string][]MessageHandler
	mx       sync.RWMutex
	isClosed bool
}

func (eb *eventBus) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	eb.mx.Lock()
	defer eb.mx.Unlock()
	if eb.isClosed {
		return nil, errors.New("bus is closed")
	}
	eb.events[subject] = append(eb.events[subject], cb)
	idx := len(eb.events[subject]) - 1
	return &subscription{
		unsubscribe: func() {
			eb.mx.Lock()
			defer eb.mx.Unlock()
			if eb.isClosed {
				return
			}
			eb.events[subject] = append(eb.events[subject][:idx], eb.events[subject][idx+1:]...)
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
	copyHandlers := make([]MessageHandler, len(handlers))
	copy(copyHandlers, handlers)
	go func() {
		for _, handler := range copyHandlers {
			handler(msg)
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
		eb.isClosed = true
		eb.events = nil
		return ctx.Err()
	default:
		go func() {
			select {
			case <-ctx.Done():
				eb.mx.Lock()
				defer eb.mx.Unlock()
				eb.isClosed = true
				eb.events = nil
				close(done)
			case <-done:
				return
			}
		}()
	}
	return nil
}
