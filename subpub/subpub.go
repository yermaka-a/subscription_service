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
	Subscribe(subject string, cb MessageHandler) (Subscription, error)

	Publish(subject string, msg interface{}) error

	Close(ctx context.Context) error
}

func NewSubPub() SubPub {
	return &eventBus{
		handlers: make(map[string][]MessageHandler),
	}
}

type subscription struct {
	unsubscribe func()
}

func (s *subscription) Unsubscribe() {
	s.unsubscribe()
}

type eventBus struct {
	handlers map[string][]MessageHandler
	lock     sync.RWMutex
	closed   bool
}

func (eb *eventBus) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	eb.lock.Lock()
	defer eb.lock.Unlock()

	if eb.closed {
		return nil,
			errors.New("bus is cloed")
	}

	eb.handlers[subject] = append(eb.handlers[subject], cb)

	idx := len(eb.handlers[subject]) - 1

	return &subscription{
		unsubscribe: func() {
			eb.lock.Lock()
			defer eb.lock.Unlock()
			if eb.closed {
				return
			}

			handlers := eb.handlers[subject]
			eb.handlers[subject] = append(handlers[:idx], handlers[idx+1:]...)
		},
	}, nil
}

func (eb *eventBus) Publish(subject string, msg interface{}) error {
	eb.lock.RLock()
	defer eb.lock.RUnlock()
	if eb.closed {
		return errors.New("bus is closed")
	}

	if handlers, found := eb.handlers[subject]; found {
		for _, handler := range handlers {
			go handler(msg)
		}
	}
	return nil
}

func (eb *eventBus) Close(ctx context.Context) error {
	eb.lock.Lock()
	defer eb.lock.Unlock()

	if eb.closed {
		return errors.New("bus is already closed")
	}
	eb.closed = true
	return nil
}
