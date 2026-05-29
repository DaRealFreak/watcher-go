package http

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestLeaseCoordinator_GrantsWhenFits(t *testing.T) {
	c := NewLeaseCoordinator()
	pol := DomainPolicy{Max: 10}

	a, err := c.AcquireLease(context.Background(), "modA", "alice@nordvpn.com", 4, pol)
	if err != nil {
		t.Fatalf("modA: %v", err)
	}
	defer a.Release()
	if a.Slots() != 4 {
		t.Fatalf("modA slots: got %d want 4", a.Slots())
	}

	b, err := c.AcquireLease(context.Background(), "modB", "alice@nordvpn.com", 5, pol)
	if err != nil {
		t.Fatalf("modB: %v", err)
	}
	defer b.Release()
	if b.Slots() != 5 {
		t.Fatalf("modB slots: got %d want 5", b.Slots())
	}
}

func TestLeaseCoordinator_QueuesWhenFull(t *testing.T) {
	c := NewLeaseCoordinator()
	pol := DomainPolicy{Max: 10}

	a, err := c.AcquireLease(context.Background(), "modA", "alice@nordvpn.com", 6, pol)
	if err != nil {
		t.Fatalf("modA: %v", err)
	}

	// modB needs 5 but only 4 are free -> blocks until modA releases.
	got := make(chan *Lease, 1)
	go func() {
		b, err := c.AcquireLease(context.Background(), "modB", "alice@nordvpn.com", 5, pol)
		if err != nil {
			t.Errorf("modB: %v", err)
			return
		}
		got <- b
	}()

	// Ensure modB is actually blocked.
	select {
	case <-got:
		t.Fatal("modB should not have been granted while modA holds 6 slots")
	case <-time.After(50 * time.Millisecond):
	}

	a.Release()

	select {
	case b := <-got:
		defer b.Release()
		if b.Slots() != 5 {
			t.Fatalf("modB slots: got %d want 5", b.Slots())
		}
	case <-time.After(time.Second):
		t.Fatal("modB never unblocked after modA released")
	}
}

func TestLeaseCoordinator_PoolsAreIndependent(t *testing.T) {
	c := NewLeaseCoordinator()
	pol := DomainPolicy{Max: 5}

	a, err := c.AcquireLease(context.Background(), "modA", "alice@nordvpn.com", 5, pol)
	if err != nil {
		t.Fatalf("alice: %v", err)
	}
	defer a.Release()

	// Different account -> separate pool -> not blocked.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	b, err := c.AcquireLease(ctx, "modA", "bob@nordvpn.com", 5, pol)
	if err != nil {
		t.Fatalf("bob: %v", err)
	}
	defer b.Release()
}

func TestLeaseCoordinator_DuplicateLeaseRejected(t *testing.T) {
	c := NewLeaseCoordinator()
	pol := DomainPolicy{Max: 10}

	a, err := c.AcquireLease(context.Background(), "modA", "alice@nordvpn.com", 3, pol)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	defer a.Release()

	if _, err := c.AcquireLease(context.Background(), "modA", "alice@nordvpn.com", 3, pol); err == nil {
		t.Fatalf("expected error for duplicate lease, got nil")
	}
}

func TestLeaseCoordinator_ContextCancelWhileQueued(t *testing.T) {
	c := NewLeaseCoordinator()
	pol := DomainPolicy{Max: 5}

	a, err := c.AcquireLease(context.Background(), "modA", "alice@nordvpn.com", 5, pol)
	if err != nil {
		t.Fatalf("modA: %v", err)
	}
	defer a.Release()

	ctx, cancel := context.WithCancel(context.Background())
	got := make(chan error, 1)
	go func() {
		_, err := c.AcquireLease(ctx, "modB", "alice@nordvpn.com", 1, pol)
		got <- err
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-got:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("queued lease did not return on ctx cancel")
	}
}

func TestLeaseCoordinator_SlotCountClamped(t *testing.T) {
	c := NewLeaseCoordinator()
	pol := DomainPolicy{Max: 5}

	a, err := c.AcquireLease(context.Background(), "modA", "alice@nordvpn.com", 0, pol)
	if err != nil {
		t.Fatalf("zero slots: %v", err)
	}
	defer a.Release()
	if a.Slots() != 1 {
		t.Fatalf("zero slots clamped: got %d want 1", a.Slots())
	}

	b, err := c.AcquireLease(context.Background(), "modB", "bob@nordvpn.com", 100, pol)
	if err != nil {
		t.Fatalf("over-max slots: %v", err)
	}
	defer b.Release()
	if b.Slots() != 5 {
		t.Fatalf("over-max slots clamped: got %d want 5", b.Slots())
	}
}

func TestLease_AcquireRoutesThroughLimiter(t *testing.T) {
	c := NewLeaseCoordinator()
	pol := DomainPolicy{Max: 10}

	lease, err := c.AcquireLease(context.Background(), "modA", "alice@nordvpn.com", 2, pol)
	if err != nil {
		t.Fatalf("acquire lease: %v", err)
	}
	defer lease.Release()

	// Within the lease, cap is the lease's own slot count (2).
	s1, err := lease.Acquire(context.Background(), "us1.proxy.nordvpn.com")
	if err != nil {
		t.Fatalf("acquire 1: %v", err)
	}
	defer s1.Release()
	s2, err := lease.Acquire(context.Background(), "us2.proxy.nordvpn.com")
	if err != nil {
		t.Fatalf("acquire 2: %v", err)
	}
	defer s2.Release()
	// Same-host piggyback within lease is instant even at slot cap.
	s3, err := lease.Acquire(context.Background(), "us1.proxy.nordvpn.com")
	if err != nil {
		t.Fatalf("acquire 3 (piggyback): %v", err)
	}
	defer s3.Release()
}

func TestLease_ReleaseWakesAllWaiters(t *testing.T) {
	c := NewLeaseCoordinator()
	pol := DomainPolicy{Max: 4}

	hog, err := c.AcquireLease(context.Background(), "modHog", "alice@nordvpn.com", 4, pol)
	if err != nil {
		t.Fatalf("hog: %v", err)
	}

	const peers = 3
	results := make(chan *Lease, peers)
	var wg sync.WaitGroup
	for i := 0; i < peers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			lease, err := c.AcquireLease(context.Background(), modKey(i), "alice@nordvpn.com", 1, pol)
			if err != nil {
				t.Errorf("peer %d: %v", i, err)
				return
			}
			results <- lease
		}()
	}

	time.Sleep(50 * time.Millisecond)
	if len(results) != 0 {
		t.Fatal("peers should be blocked while hog holds all 4 slots")
	}

	hog.Release()
	wg.Wait()
	close(results)

	count := 0
	for l := range results {
		count++
		l.Release()
	}
	if count != peers {
		t.Fatalf("expected %d leases granted after hog release; got %d", peers, count)
	}
}

func modKey(i int) string {
	return "peer-" + string(rune('A'+i))
}
