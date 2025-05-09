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
		events: make(map[string]*uniqueChs, 0),
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
	id   uint64
	next *uniqueCh
	ch   chan interface{}
}

type uniqueChs struct {
	head *uniqueCh
}

func (uChs *uniqueChs) push_back(uCh *uniqueCh) {
	if uChs.head == nil {
		uChs.head = uCh
		return
	}
	for next := uChs.head; ; next = next.next {
		if next.next == nil {
			next.next = uCh
			return
		}
	}
}

func (uChs *uniqueChs) pop(id uint64) {
	if uChs.head.id == id {
		uChs.head = uChs.head.next
		return
	}
	for next := uChs.head; next != nil; next = next.next {
		if next.next.id == id {
			next.next = next.next.next
			return
		}
	}
}

type eventBus struct {
	nextId uint64
	// хэш-таблица со списком каналов для subject
	events   map[string]*uniqueChs
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
	if eb.events[subject] == nil {
		eb.events[subject] = &uniqueChs{}
	}
	eb.events[subject].push_back(unique)
	eb.nextId += 1
	return &subscription{
		unsubscribe: func() {
			eb.mx.Lock()
			defer eb.mx.Unlock()
			if eb.isClosed {
				return
			}
			close(unique.ch)
			eb.events[subject].pop(unique.id)

			// Если список пуст, то удаляем ключ
			if eb.events[subject].head == nil {
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
	for next := eb.events[subject].head; next != nil; next = next.next {
		next.ch <- msg
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
		for next := uniqueLists.head; next != nil; next = next.next {
			close(next.ch)
		}
		delete(eb.events, eventName)
	}
	eb.events = nil

}
