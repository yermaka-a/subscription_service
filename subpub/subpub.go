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
		events: make(map[string][]*uniqueCh, 0),
	}
}

type subscription struct {
	unsubscribe func()
}

func (sb *subscription) Unsubscribe() {
	sb.unsubscribe()
}

// Уникальный id канала для каждого канала подписчика
type uniqueCh struct {
	id uint64
	ch chan interface{}
}

type eventBus struct {
	nextId uint64
	// хэш-таблица со списком каналов для subject
	events   map[string][]*uniqueCh
	mx       sync.RWMutex
	isClosed bool
}

func (eb *eventBus) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	eb.mx.Lock()
	defer eb.mx.Unlock()
	if eb.isClosed {
		return nil, errors.New("bus is closed")
	}

	eventCh := make(chan interface{}, 1)
	go func(c <-chan interface{}) {
		for {
			msg, ok := <-c
			if !ok {
				break
			}
			cb(msg)
		}
	}(eventCh)
	unique := &uniqueCh{id: eb.nextId, ch: eventCh}
	eb.events[subject] = append(eb.events[subject], unique)
	eb.nextId += 1
	return &subscription{
		unsubscribe: func() {
			eb.mx.Lock()
			defer eb.mx.Unlock()
			if eb.isClosed {
				return
			}
			close(unique.ch)
			for idx, ch := range eb.events[subject] {
				if ch.id == unique.id {
					eb.events[subject] = append(eb.events[subject][:idx], eb.events[subject][idx+1:]...)
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
	_, isExists := eb.events[subject]
	if !isExists {
		return errors.New("event isn't found")
	}

	for _, unique := range eb.events[subject] {
		unique.ch <- msg
	}
	return nil
}

func (eb *eventBus) Close(ctx context.Context) error {
	if eb.isClosed {
		return errors.New("bus is already closed")
	}
	select {
	case <-ctx.Done():
		eb.closeAndClear()
		return ctx.Err()
	default:
		go func() {
			done := make(chan struct{})
			select {
			case <-ctx.Done():
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
	eb.mx.Lock()
	defer eb.mx.Unlock()
	if eb.isClosed {
		return
	}
	eb.isClosed = true
	for eventName, uniqueLists := range eb.events {
		for _, uniqueCh := range uniqueLists {
			close(uniqueCh.ch)
		}
		delete(eb.events, eventName)
	}
	eb.events = nil

}
