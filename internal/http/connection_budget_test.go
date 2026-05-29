package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func policy(max int) map[string]DomainPolicy {
	return map[string]DomainPolicy{"nordvpn.com": {Max: max}}
}

func policyWithCooldown(max int, cd time.Duration) map[string]DomainPolicy {
	return map[string]DomainPolicy{"nordvpn.com": {Max: max, Cooldown: cd}}
}

func TestDomainFor(t *testing.T) {
	cases := []struct{ host, want string }{
		{"us8365.proxy.nordvpn.com", "nordvpn.com"},
		{"pl56.protonvpn.net", "protonvpn.net"},
		{"mullvad.net", "mullvad.net"},
		{"192.168.1.10", "192.168.1.10"},
		{"localhost", "localhost"},
	}
	for _, c := range cases {
		if got := DomainFor(c.host); got != c.want {
			t.Errorf("DomainFor(%q) = %q, want %q", c.host, got, c.want)
		}
	}
}

func TestPoolKeyFor(t *testing.T) {
	cases := []struct{ user, host, want string }{
		{"alice", "us8365.proxy.nordvpn.com", "alice@nordvpn.com"},
		{"alice", "us8410.proxy.nordvpn.com", "alice@nordvpn.com"},
		{"bob", "us8410.proxy.nordvpn.com", "bob@nordvpn.com"},
		{"alice", "pl56.protonvpn.net", "alice@protonvpn.net"},
		{"", "192.168.1.10", "@192.168.1.10"},
	}
	for _, c := range cases {
		if got := PoolKeyFor(c.user, c.host); got != c.want {
			t.Errorf("PoolKeyFor(%q,%q)=%q want %q", c.user, c.host, got, c.want)
		}
	}
}

func TestConnectionBudget_UnconfiguredDomainUnlimited(t *testing.T) {
	b := NewConnectionBudget(policy(2))
	ps := &ProxySettings{Enable: true, Host: "proxy.example.org", Username: "u"}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := b.Acquire(context.Background(), "", ps)
			if err != nil {
				t.Errorf("acquire: %v", err)
				return
			}
			s.Release()
		}()
	}
	wg.Wait()
}

// TestConnectionBudget_CapEnforcedOnDistinctActiveHosts: even with LRU keeping
// idle hosts in the pool, at most `cap` hosts may be *actively* in flight at
// the same time. With cap=2 and 10 goroutines on distinct hosts, no more than
// 2 are mid-request simultaneously.
func TestConnectionBudget_CapEnforcedOnDistinctActiveHosts(t *testing.T) {
	b := NewConnectionBudget(policy(2))
	var mu sync.Mutex
	inFlight := map[string]int{}
	var maxObserved int32

	record := func(host string, delta int) {
		mu.Lock()
		inFlight[host] += delta
		if inFlight[host] == 0 {
			delete(inFlight, host)
		}
		cur := int32(len(inFlight))
		mu.Unlock()
		if delta > 0 {
			for {
				prev := atomic.LoadInt32(&maxObserved)
				if cur <= prev || atomic.CompareAndSwapInt32(&maxObserved, prev, cur) {
					break
				}
			}
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		host := fmt.Sprintf("us%d.proxy.nordvpn.com", i+1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			ps := &ProxySettings{Enable: true, Host: host, Username: "alice"}
			slot, err := b.Acquire(context.Background(), "", ps)
			if err != nil {
				t.Errorf("acquire: %v", err)
				return
			}
			record(host, 1)
			time.Sleep(20 * time.Millisecond)
			record(host, -1)
			slot.Release()
		}()
	}
	wg.Wait()
	if got := atomic.LoadInt32(&maxObserved); got > 2 {
		t.Fatalf("observed %d distinct active hosts, cap=2", got)
	}
}

// TestConnectionBudget_SameHostPiggybacks: concurrent requests to the same
// host share a single host-slot, so cap=1 doesn't block any of them.
func TestConnectionBudget_SameHostPiggybacks(t *testing.T) {
	b := NewConnectionBudget(policy(1))
	ps := &ProxySettings{Enable: true, Host: "us1.proxy.nordvpn.com", Username: "alice"}

	const N = 8
	started := make(chan struct{}, N)
	proceed := make(chan struct{})
	var wg sync.WaitGroup
	slots := make([]*Slot, N)
	for i := 0; i < N; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			slot, err := b.Acquire(context.Background(), "", ps)
			if err != nil {
				t.Errorf("acquire: %v", err)
				return
			}
			slots[i] = slot
			started <- struct{}{}
			<-proceed
			slot.Release()
		}()
	}

	deadline := time.After(time.Second)
	for i := 0; i < N; i++ {
		select {
		case <-started:
		case <-deadline:
			t.Fatalf("only %d/%d goroutines acquired within deadline (cap=1 should allow same-host piggyback)", i, N)
		}
	}
	close(proceed)
	wg.Wait()
}

// TestConnectionBudget_SameHostInstantWhileIdle: once a host is in the pool,
// re-acquiring it after release is instant — the host stays in the pool as
// idle and only gets evicted under pressure from a *different* host.
func TestConnectionBudget_SameHostInstantWhileIdle(t *testing.T) {
	b := NewConnectionBudget(policyWithCooldown(2, 500*time.Millisecond))
	ps := &ProxySettings{Enable: true, Host: "us1.proxy.nordvpn.com", Username: "alice"}

	slot, err := b.Acquire(context.Background(), "", ps)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	slot.Release()

	// Even with cooldown configured, same-host re-acquire while idle should
	// not wait — the host never left the pool.
	start := time.Now()
	for i := 0; i < 5; i++ {
		slot, err = b.Acquire(context.Background(), "", ps)
		if err != nil {
			t.Fatalf("re-acquire %d: %v", i, err)
		}
		slot.Release()
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("same-host re-acquires should be instant; 5 took %s", elapsed)
	}
}

func TestConnectionBudget_PoolsAreIsolated(t *testing.T) {
	b := NewConnectionBudget(policy(1))
	alice := &ProxySettings{Enable: true, Host: "us1.proxy.nordvpn.com", Username: "alice"}
	bob := &ProxySettings{Enable: true, Host: "us1.proxy.nordvpn.com", Username: "bob"}

	held, err := b.Acquire(context.Background(), "", alice)
	if err != nil {
		t.Fatalf("alice acquire: %v", err)
	}
	defer held.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	s, err := b.Acquire(ctx, "", bob)
	if err != nil {
		t.Fatalf("bob acquire should not block on alice: %v", err)
	}
	s.Release()
}

func TestConnectionBudget_NilProxyOrDisabled(t *testing.T) {
	b := NewConnectionBudget(policy(2))
	s, err := b.Acquire(context.Background(), "", nil)
	if err != nil {
		t.Fatalf("nil ps: %v", err)
	}
	s.Release()
	s, err = b.Acquire(context.Background(), "", &ProxySettings{Enable: false})
	if err != nil {
		t.Fatalf("disabled ps: %v", err)
	}
	s.Release()
}

// TestConnectionBudget_ContextCancel: a new-host claim that's stuck at cap
// (every host actively in flight) returns ctx.Err() on cancel.
func TestConnectionBudget_ContextCancel(t *testing.T) {
	b := NewConnectionBudget(policy(1))
	aPs := &ProxySettings{Enable: true, Host: "us1.proxy.nordvpn.com", Username: "alice"}
	bPs := &ProxySettings{Enable: true, Host: "us2.proxy.nordvpn.com", Username: "alice"}
	held, err := b.Acquire(context.Background(), "", aPs)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	defer held.Release()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := b.Acquire(ctx, "", bPs); err == nil {
		t.Fatalf("expected context error")
	}
}

func TestConnectionBudget_NilBudgetSafe(t *testing.T) {
	var b *ConnectionBudget
	s, err := b.Acquire(context.Background(), "", &ProxySettings{Enable: true, Host: "x", Username: "u"})
	if err != nil {
		t.Fatalf("nil budget: %v", err)
	}
	s.Release()
}

// TestReleasingBody_ReleasesOnClose: closing the wrapped response body of
// host A's request makes A idle. A different host B then evicts A and is
// admitted (no cooldown configured in this test).
func TestReleasingBody_ReleasesOnClose(t *testing.T) {
	b := NewConnectionBudget(policy(1))
	aPs := &ProxySettings{Enable: true, Host: "us1.proxy.nordvpn.com", Username: "alice"}
	bPs := &ProxySettings{Enable: true, Host: "us2.proxy.nordvpn.com", Username: "alice"}
	slot, err := b.Acquire(context.Background(), "", aPs)
	if err != nil {
		t.Fatalf("acquire A: %v", err)
	}

	blocked := make(chan struct{})
	acquired := make(chan *Slot, 1)
	go func() {
		close(blocked)
		s, err := b.Acquire(context.Background(), "", bPs)
		if err != nil {
			t.Errorf("acquire B: %v", err)
			return
		}
		acquired <- s
	}()
	<-blocked
	time.Sleep(20 * time.Millisecond)
	select {
	case s := <-acquired:
		s.Release()
		t.Fatal("B acquired while A still active (cap=1, A in flight)")
	default:
	}

	wrapped := WrapBodyWithSlot(io.NopCloser(bytes.NewReader([]byte("x"))), slot)
	if err := wrapped.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case s := <-acquired:
		s.Release()
	case <-time.After(time.Second):
		t.Fatal("B did not unblock after A's body closed")
	}
}

func TestWrapBodyWithSlot_NilSlotPassthrough(t *testing.T) {
	body := io.NopCloser(bytes.NewReader([]byte("x")))
	if WrapBodyWithSlot(body, nil) != body {
		t.Fatalf("expected pass-through when slot is nil")
	}
}

// TestConnectionBudget_CooldownBetweenEvictions: the first eviction has no
// prior to space against and proceeds instantly. The next eviction must wait
// the configured cooldown after that first one.
func TestConnectionBudget_CooldownBetweenEvictions(t *testing.T) {
	cd := 150 * time.Millisecond
	b := NewConnectionBudget(policyWithCooldown(1, cd))
	psA := &ProxySettings{Enable: true, Host: "us1.proxy.nordvpn.com", Username: "alice"}
	psB := &ProxySettings{Enable: true, Host: "us2.proxy.nordvpn.com", Username: "alice"}
	psC := &ProxySettings{Enable: true, Host: "us3.proxy.nordvpn.com", Username: "alice"}

	slotA, err := b.Acquire(context.Background(), "", psA)
	if err != nil {
		t.Fatalf("acquire A: %v", err)
	}
	slotA.Release() // A now idle in the pool

	// First eviction: A is the LRU idle, B comes in. No prior eviction.
	t0 := time.Now()
	slotB, err := b.Acquire(context.Background(), "", psB)
	if err != nil {
		t.Fatalf("acquire B: %v", err)
	}
	if elapsed := time.Since(t0); elapsed > 50*time.Millisecond {
		t.Fatalf("first eviction should be instant; took %s", elapsed)
	}
	slotB.Release() // B now idle

	// Second eviction: B is LRU idle, C comes in. Should wait cooldown
	// (less the time elapsed since the first eviction).
	t1 := time.Now()
	slotC, err := b.Acquire(context.Background(), "", psC)
	if err != nil {
		t.Fatalf("acquire C: %v", err)
	}
	defer slotC.Release()
	if elapsed := time.Since(t1); elapsed < cd-50*time.Millisecond {
		t.Fatalf("second eviction should wait cooldown; took %s (cooldown=%s)", elapsed, cd)
	}
}

// TestConnectionBudget_NoEvictionWhilePoolHasRoom: with cap=2 and only one
// host ever used, the second host's claim doesn't trigger an eviction and
// doesn't pay cooldown.
func TestConnectionBudget_NoEvictionWhilePoolHasRoom(t *testing.T) {
	b := NewConnectionBudget(policyWithCooldown(2, 500*time.Millisecond))
	a := &ProxySettings{Enable: true, Host: "us1.proxy.nordvpn.com", Username: "alice"}
	bHost := &ProxySettings{Enable: true, Host: "us2.proxy.nordvpn.com", Username: "alice"}

	slotA, err := b.Acquire(context.Background(), "", a)
	if err != nil {
		t.Fatalf("acquire A: %v", err)
	}
	defer slotA.Release()

	start := time.Now()
	slotB, err := b.Acquire(context.Background(), "", bHost)
	if err != nil {
		t.Fatalf("acquire B: %v", err)
	}
	defer slotB.Release()

	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("admit-without-eviction should be instant; took %s", elapsed)
	}
}

// TestConnectionBudget_CooldownZeroBypasses: zero cooldown disables the
// grace period entirely — back-to-back evictions are instant.
func TestConnectionBudget_CooldownZeroBypasses(t *testing.T) {
	b := NewConnectionBudget(policyWithCooldown(1, 0))
	psA := &ProxySettings{Enable: true, Host: "us1.proxy.nordvpn.com", Username: "alice"}
	psB := &ProxySettings{Enable: true, Host: "us2.proxy.nordvpn.com", Username: "alice"}
	psC := &ProxySettings{Enable: true, Host: "us3.proxy.nordvpn.com", Username: "alice"}

	slot, _ := b.Acquire(context.Background(), "", psA)
	slot.Release()
	slot, _ = b.Acquire(context.Background(), "", psB) // evicts A
	slot.Release()

	start := time.Now()
	slot, err := b.Acquire(context.Background(), "", psC) // evicts B
	if err != nil {
		t.Fatalf("acquire C: %v", err)
	}
	defer slot.Release()
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("cooldown=0 should not delay; took %s", elapsed)
	}
}

// TestConnectionBudget_CooldownContextCancel: a context canceled while
// waiting out the cooldown before the second eviction returns ctx.Err() and
// leaves the pool consistent.
func TestConnectionBudget_CooldownContextCancel(t *testing.T) {
	b := NewConnectionBudget(policyWithCooldown(1, time.Second))
	psA := &ProxySettings{Enable: true, Host: "us1.proxy.nordvpn.com", Username: "alice"}
	psB := &ProxySettings{Enable: true, Host: "us2.proxy.nordvpn.com", Username: "alice"}
	psC := &ProxySettings{Enable: true, Host: "us3.proxy.nordvpn.com", Username: "alice"}

	slot, _ := b.Acquire(context.Background(), "", psA)
	slot.Release()
	slot, _ = b.Acquire(context.Background(), "", psB) // first eviction (no wait)
	slot.Release()

	// Second eviction would need cooldown. Cancel mid-wait.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := b.Acquire(ctx, "", psC); err == nil {
		t.Fatalf("expected ctx error during cooldown wait")
	}

	// B should still be idle in the pool — same-host re-acquire is instant.
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	slot, err := b.Acquire(ctx2, "", psB)
	if err != nil {
		t.Fatalf("post-cancel B re-acquire: %v", err)
	}
	slot.Release()
}

// TestConnectionBudget_LRUEvictsOldestIdle: with cap=2, after A and B are
// both released (idle), admitting C should evict whichever became idle
// *first* (LRU), keeping the more recently used one.
func TestConnectionBudget_LRUEvictsOldestIdle(t *testing.T) {
	b := NewConnectionBudget(policy(2))
	psA := &ProxySettings{Enable: true, Host: "us1.proxy.nordvpn.com", Username: "alice"}
	psB := &ProxySettings{Enable: true, Host: "us2.proxy.nordvpn.com", Username: "alice"}
	psC := &ProxySettings{Enable: true, Host: "us3.proxy.nordvpn.com", Username: "alice"}

	slotA, _ := b.Acquire(context.Background(), "", psA)
	slotA.Release()
	// A is the older idle entry.
	time.Sleep(5 * time.Millisecond)
	slotB, _ := b.Acquire(context.Background(), "", psB)
	slotB.Release()
	// B is the more recent idle entry.

	// Admitting C should evict A (oldest idle). B stays. Verify by checking
	// that B re-acquire is instant (still in pool) and A re-acquire requires
	// another eviction (and would pay cooldown — but cooldown=0 here, so we
	// can't observe a wait; instead we verify state via behavior).
	slotC, _ := b.Acquire(context.Background(), "", psC)
	defer slotC.Release()

	// B should still be in the pool as idle: re-acquire instant.
	start := time.Now()
	slotB2, err := b.Acquire(context.Background(), "", psB)
	if err != nil {
		t.Fatalf("re-acquire B: %v", err)
	}
	defer slotB2.Release()
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("B should still be in pool (most-recently-idle); re-acquire took %s", elapsed)
	}
}

// TestConnectionBudget_EvictsOnlyIdle: at cap with every host actively in
// flight (no idle), a new-host claim blocks until one releases — it does NOT
// evict an active host.
func TestConnectionBudget_EvictsOnlyIdle(t *testing.T) {
	b := NewConnectionBudget(policy(1))
	psA := &ProxySettings{Enable: true, Host: "us1.proxy.nordvpn.com", Username: "alice"}
	psB := &ProxySettings{Enable: true, Host: "us2.proxy.nordvpn.com", Username: "alice"}

	slotA, _ := b.Acquire(context.Background(), "", psA)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := b.Acquire(ctx, "", psB); err == nil {
		t.Fatalf("B should have blocked while A is active")
	}
	slotA.Release()
}
