package subpub

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSubscribeAndPublish(t *testing.T) {
	bus := NewSubPub()
	received := false
	sub, err := bus.Subscribe("test", func(msg interface{}) {
		if msg == "hello" {
			received = true
		}
	})

	if err != nil {
		t.Fatalf("unexpected error during subscription: %v", err)
	}

	err = bus.Publish("test", "hello")
	if err != nil {
		t.Fatalf("unexpected error during publish: %v", err)
	}

	// wait for message to be processed
	time.Sleep(100 * time.Millisecond)
	if !received {
		t.Fatal("exptected to receive a message but didn't")
	}
	sub.Unsubscribe()
}

func TestUnsubscribe(t *testing.T) {
	bus := NewSubPub()
	received := false
	sub, _ := bus.Subscribe("test", func(msg interface{}) {
		received = true
	})

	sub.Unsubscribe()
	err := bus.Publish("test", "hello")
	if err == nil || err.Error() != "event isn't found" {
		t.Fatalf("expcted error 'event isn't found', got: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if received {
		t.Fatal("did not expect to receive a message after unsubsrcibe")
	}
}

func TestPublishNoSubscribers(t *testing.T) {
	bus := NewSubPub()
	err := bus.Publish("no_subs", "hello")
	if err == nil {
		t.Fatal("expected rror when publishing to a subject with no subscribers")
	}
}

func TestTimeoutClose(t *testing.T) {
	bus := NewSubPub()

	_, err := bus.Subscribe("test", func(msg interface{}) {})
	if err != nil {
		t.Fatalf("unexpected during subscription: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = bus.Close(ctx)
	if err != nil {
		t.Fatalf("unexpcted error during close %v", err)
	}

	// Проверяем что шина закрыта
	err = bus.Publish("test", "hello")
	if err != nil {
		t.Fatalf("unexpected error when publishing to a bus by using timeout: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	err = bus.Publish("test", "hello")
	if err == nil {
		t.Fatal("expected error when publishing to a closed bus")
	}
}

func TestCloseWithContextCancel(t *testing.T) {
	bus := NewSubPub()

	_, err := bus.Subscribe("test", func(msg interface{}) {})

	if err != nil {
		t.Fatalf("unexpcted error during subscription: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = bus.Close(ctx)
	if err == nil {
		t.Fatalf("expected error due to context cancellation, got nil")
	}
}

func TestConcurrentAcces(t *testing.T) {
	bus := NewSubPub()
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = bus.Subscribe("test", func(msg interface{}) {})

	}()
	go func() {
		defer wg.Done()
		_ = bus.Publish("test", "hello")
	}()
	wg.Wait()
}

func TestHeavyConcurrentAccess(t *testing.T) {
	bus := NewSubPub()

	var wg sync.WaitGroup
	numPublishers := 100
	numSubscribers := 100
	numMessages := 1000

	// Создаем подписчиков
	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func(subID int) {
			defer wg.Done()
			sub, err := bus.Subscribe("concurrent", func(msg interface{}) {
			})
			if err != nil {
				t.Errorf("unexpected error during subscription: %v", err)
			}
			time.Sleep(1 * time.Millisecond)
			sub.Unsubscribe()
		}(i)
	}

	// Создаем публикаторов
	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(pubID int) {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				err := bus.Publish("concurrent", fmt.Sprintf("message-%d-%d", pubID, j))
				if err != nil && err.Error() != "event isn't found" {
					t.Errorf("unexpected error during publish: %v", err)
				}
			}
		}(i)
	}

	// Отписываем подписчиков
	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func(subID int) {
			defer wg.Done()
			sub, err := bus.Subscribe("concurrent", func(msg interface{}) {})
			if err != nil {
				t.Errorf("unexpected error during subscription: %v", err)
			}
			time.Sleep(1 * time.Millisecond)
			sub.Unsubscribe()
		}(i)
	}

	wg.Wait()
}

func TestGetAllDataWithUnsubscribes(t *testing.T) {
	bus := NewSubPub()
	const numSubscribers = 100

	var wg sync.WaitGroup
	var receivedArr [numSubscribers]bool
	// Создаем подписчиков
	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func(subID int) {
			defer wg.Done()
			_, err := bus.Subscribe("concurrent", func(msg interface{}) {
				receivedArr[subID] = true
			})
			if err != nil {
				t.Errorf("unexpected error during subscription: %v", err)
			}
		}(i)
	}
	wg.Wait()

	bus.Publish("concurrent", struct{}{})

	time.Sleep(200 * time.Millisecond)
	for i := 0; i < numSubscribers; i++ {
		if receivedArr[i] != true {
			t.Fatalf("expected all true receivedArr values but by index %d got %v", i, receivedArr[i])
		}
	}
}
