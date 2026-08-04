package closer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestCloser_LIFO_Order(t *testing.T) {
	c := newCloser()

	var order []string

	c.Add("first", func(_ context.Context) error {
		order = append(order, "first")
		return nil
	})
	c.Add("second", func(_ context.Context) error {
		order = append(order, "second")
		return nil
	})
	c.Add("third", func(_ context.Context) error {
		order = append(order, "third")
		return nil
	})

	err := c.CloseAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"third", "second", "first"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %q, got %q", i, v, order[i])
		}
	}
}

func TestCloser_ReturnsFirstError(t *testing.T) {
	c := newCloser()

	errFirst := errors.New("first error")
	errSecond := errors.New("second error")

	c.Add("ok", func(_ context.Context) error {
		return nil
	})
	// This one runs second (LIFO), so its error is reported first
	c.Add("fail-second", func(_ context.Context) error {
		return errSecond
	})
	// This one runs first (LIFO)
	c.Add("fail-first", func(_ context.Context) error {
		return errFirst
	})

	err := c.CloseAll(context.Background())
	if !errors.Is(err, errFirst) {
		t.Fatalf("expected the first error %q, got %q", errFirst, err)
	}
}

func TestCloser_AllFunctionsCalledDespiteErrors(t *testing.T) {
	c := newCloser()

	var called []string

	c.Add("a", func(_ context.Context) error {
		called = append(called, "a")
		return nil
	})
	c.Add("b", func(_ context.Context) error {
		called = append(called, "b")
		return errors.New("error b")
	})
	c.Add("c", func(_ context.Context) error {
		called = append(called, "c")
		return nil
	})

	_ = c.CloseAll(context.Background())

	if len(called) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(called), called)
	}
}

func TestCloser_CloseAllOnce(t *testing.T) {
	c := newCloser()

	callCount := 0
	c.Add("resource", func(_ context.Context) error {
		callCount++
		return nil
	})

	_ = c.CloseAll(context.Background())
	_ = c.CloseAll(context.Background())
	_ = c.CloseAll(context.Background())

	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}
}

func TestCloser_EmptyCloser(t *testing.T) {
	c := newCloser()

	err := c.CloseAll(context.Background())
	if err != nil {
		t.Fatalf("expected a nil error from an empty closer, got %v", err)
	}
}

func TestCloser_RespectsContextCancellation(t *testing.T) {
	c := newCloser()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var receivedCtx context.Context
	c.Add("resource", func(ctx context.Context) error {
		receivedCtx = ctx
		return ctx.Err()
	})

	err := c.CloseAll(ctx)
	if nil == err {
		t.Fatal("expected a context error, got nil")
	}

	if nil == receivedCtx.Err() {
		t.Fatal("expected the cancelled context to reach the close function")
	}
}

func TestCloser_ConcurrentAdd(t *testing.T) {
	c := newCloser()

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Add("resource", func(_ context.Context) error {
				return nil
			})
		}()
	}
	wg.Wait()

	// Check that all 100 were registered
	c.mu.Lock()
	count := len(c.funcs)
	c.mu.Unlock()

	if count != 100 {
		t.Fatalf("expected 100 functions, got %d", count)
	}
}

func TestCloser_ContextTimeout(t *testing.T) {
	c := newCloser()

	c.Add("slow", func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := c.CloseAll(ctx)
	elapsed := time.Since(start)

	if nil == err {
		t.Fatal("expected a timeout error, got nil")
	}

	if elapsed > 1*time.Second {
		t.Fatalf("expected a fast timeout, but it took %v", elapsed)
	}
}
