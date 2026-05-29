package http

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/publicsuffix"
)

// Wait-state codes published by Acquire so the diagnostic warn goroutine can
// distinguish "blocked at cap" (potential leak) from "in cooldown" (expected,
// configured behavior).
const (
	waitStateUnknown  int32 = 0
	waitStateAtCap    int32 = 1
	waitStateCooldown int32 = 2
)

// DomainPolicy is the per-domain budget policy. Max caps the number of
// *distinct hosts* simultaneously held in the (username, domain) pool —
// including idle hosts that no longer have in-flight requests but haven't
// been evicted yet. Same-host requests share one slot; idle hosts stay in
// the pool until a different new host pressures one out (LRU eviction).
//
// Cooldown, if >0, is the minimum spacing between consecutive evictions:
// admitting a new host that requires evicting an idle one waits until
// cooldown has elapsed since the most recent eviction. This gives the VPN's
// server-side connection counter time to decrement between host transitions.
type DomainPolicy struct {
	Max      int
	Cooldown time.Duration
}

// ConnectionBudget caps simultaneous in-flight HTTP requests per
// (proxy username, host eTLD+1) pool. policies maps domains (e.g.
// "nordvpn.com") to the per-account cap and cooldown for that service.
// Domains absent from the map are unlimited.
type ConnectionBudget struct {
	policies map[string]DomainPolicy
	pools    sync.Map // poolKey -> *ConnectionLimiter
}

// Global is the process-wide budget. Initialized once at startup via
// InitGlobalBudget. Nil before initialization; nil-safe to call.
var Global *ConnectionBudget

// InitGlobalBudget (re-)initializes the package-global budget.
func InitGlobalBudget(policies map[string]DomainPolicy) {
	Global = NewConnectionBudget(policies)
}

// NewConnectionBudget constructs a budget. The policies map is copied so later
// callers can't mutate it. Zero or negative Max values are treated as unlimited.
func NewConnectionBudget(policies map[string]DomainPolicy) *ConnectionBudget {
	cp := make(map[string]DomainPolicy, len(policies))
	for k, v := range policies {
		cp[k] = v
	}
	return &ConnectionBudget{policies: cp}
}

// Acquire blocks until a slot is available for the (username, domain(host))
// pool, or ctx is canceled. If ps is nil, disabled, or its domain is not in
// the policies map, a no-op slot is returned immediately.
//
// If moduleKey is non-empty and the module holds an active lease for this
// pool, the request is routed through the lease's internal limiter and the
// shared global pool is bypassed. This gives leased modules guaranteed
// throughput against their reserved slot count, isolated from cross-module
// contention.
func (b *ConnectionBudget) Acquire(ctx context.Context, moduleKey string, ps *ProxySettings) (*Slot, error) {
	if b == nil || ps == nil || !ps.Enable || ps.Host == "" {
		return &Slot{}, nil
	}
	domain := DomainFor(ps.Host)
	pol, ok := b.policies[domain]
	if !ok || pol.Max <= 0 {
		return &Slot{}, nil
	}
	key := ps.Username + "@" + domain
	if moduleKey != "" {
		if lease := GlobalLeases.Lookup(moduleKey, key); lease != nil {
			return lease.Acquire(ctx, ps.Host)
		}
	}
	return b.getOrCreate(key, pol).Acquire(ctx, key, ps.Host)
}

// PolicyFor returns the configured per-domain policy for the given domain.
// Exposed for the watcher's lease-sizing logic so it can size a lease to the
// pool's actual cap and cooldown.
func (b *ConnectionBudget) PolicyFor(domain string) (DomainPolicy, bool) {
	if b == nil {
		return DomainPolicy{}, false
	}
	pol, ok := b.policies[domain]
	return pol, ok
}

func (b *ConnectionBudget) getOrCreate(key string, pol DomainPolicy) *ConnectionLimiter {
	if v, ok := b.pools.Load(key); ok {
		return v.(*ConnectionLimiter)
	}
	fresh := newConnectionLimiter(pol.Max, pol.Cooldown)
	actual, _ := b.pools.LoadOrStore(key, fresh)
	return actual.(*ConnectionLimiter)
}

// PoolKeyFor returns the budget pool key for a (username, host) pair.
// Exposed for diagnostics and testing.
func PoolKeyFor(username, host string) string { return username + "@" + DomainFor(host) }

// DomainFor returns the eTLD+1 of host (e.g. "us1.proxy.nordvpn.com" ->
// "nordvpn.com"). On parse failure (IP literal, unqualified name), the host
// is returned unchanged so the value still uniquely identifies a pool.
func DomainFor(host string) string {
	d, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil || d == "" {
		return host
	}
	return d
}

// hostState tracks one host's place in the LRU pool. refcount > 0 means the
// host is currently serving requests; refcount == 0 means it is idle and
// eligible for LRU eviction. lastUsed is updated on every refcount change so
// the oldest-idle entry can be identified for eviction.
type hostState struct {
	refcount int
	lastUsed time.Time
}

// ConnectionLimiter implements an LRU-managed host pool for one
// (username, domain) bucket. Hosts stay in the pool as idle entries after
// their last request finishes, so same-host re-use is always free. Only
// admitting a *different* new host while the pool is full triggers an
// eviction, and consecutive evictions are spaced by the configured cooldown
// so the VPN's server-side counter can settle between host transitions.
type ConnectionLimiter struct {
	cap      int
	cooldown time.Duration

	mu           sync.Mutex
	cond         *sync.Cond
	pool         map[string]*hostState
	lastEviction time.Time
}

func newConnectionLimiter(cap int, cooldown time.Duration) *ConnectionLimiter {
	l := &ConnectionLimiter{
		cap:      cap,
		cooldown: cooldown,
		pool:     make(map[string]*hostState),
	}
	l.cond = sync.NewCond(&l.mu)
	return l
}

// findOldestIdleLocked returns the host name of the oldest idle entry in the
// pool, or "" if every entry has in-flight requests. Caller must hold l.mu.
func (l *ConnectionLimiter) findOldestIdleLocked() string {
	var oldest string
	var oldestT time.Time
	for h, hs := range l.pool {
		if hs.refcount > 0 {
			continue
		}
		if oldest == "" || hs.lastUsed.Before(oldestT) {
			oldest = h
			oldestT = hs.lastUsed
		}
	}
	return oldest
}

// Acquire returns a Slot for the given host in the pool keyed by poolKey.
// Same-host re-acquires are immediate; new-host admissions either fit into
// a free slot, wait an inter-eviction cooldown before evicting the LRU idle
// host, or block until any pool entry becomes idle.
//
// On ctx cancel, returns ctx.Err() without holding a slot.
func (l *ConnectionLimiter) Acquire(ctx context.Context, poolKey, host string) (*Slot, error) {
	// Wake cond.Wait()ers on ctx cancel so they can re-check ctx.Err().
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			l.mu.Lock()
			l.cond.Broadcast()
			l.mu.Unlock()
		case <-stop:
		}
	}()

	// Diagnostic heartbeat. waitState lets the goroutine distinguish "blocked
	// at cap" (every host has in-flight requests — potential leak) from "in
	// cooldown" (expected, configured spacing between evictions).
	var (
		waitState   atomic.Int32
		cooldownEnd atomic.Int64 // unix nanos; valid while waitState == waitStateCooldown
	)
	start := time.Now()
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				l.mu.Lock()
				hosts := len(l.pool)
				l.mu.Unlock()
				elapsed := time.Since(start).Round(time.Second)
				switch waitState.Load() {
				case waitStateCooldown:
					remain := time.Until(time.Unix(0, cooldownEnd.Load())).Round(time.Second)
					slog.Info(fmt.Sprintf(
						"proxy slot acquire blocked on cooldown (waited=%s, remaining=%s, pool=%s, host=%s, hosts=%d/%d)",
						elapsed, remain, poolKey, host, hosts, l.cap,
					))
				case waitStateAtCap:
					slog.Warn(fmt.Sprintf(
						"proxy slot acquire blocked at cap (waited=%s, pool=%s, host=%s, hosts=%d/%d) - check for leaked slots",
						elapsed, poolKey, host, hosts, l.cap,
					))
				default:
					slog.Warn(fmt.Sprintf(
						"proxy slot acquire still pending (waited=%s, pool=%s, host=%s, hosts=%d/%d)",
						elapsed, poolKey, host, hosts, l.cap,
					))
				}
			}
		}
	}()

	// cooldownWaitLocked sleeps until lastEviction + cooldown elapses,
	// releasing l.mu during the sleep. Returns (false, nil) when no wait
	// was needed, (true, nil) when a wait happened (caller should re-check
	// pool state), or (_, ctx.Err()) when canceled mid-wait. The caller
	// must hold l.mu on entry and will hold it again on return.
	cooldownWaitLocked := func() (bool, error) {
		if l.cooldown <= 0 || l.lastEviction.IsZero() {
			return false, nil
		}
		deadline := l.lastEviction.Add(l.cooldown)
		wait := time.Until(deadline)
		if wait <= 0 {
			return false, nil
		}
		cooldownEnd.Store(deadline.UnixNano())
		waitState.Store(waitStateCooldown)
		l.mu.Unlock()
		timer := time.NewTimer(wait)
		var ctxErr error
		select {
		case <-timer.C:
		case <-ctx.Done():
			ctxErr = ctx.Err()
		}
		timer.Stop()
		l.mu.Lock()
		waitState.Store(waitStateUnknown)
		return true, ctxErr
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	atCapLogged := false
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Fast path: host is already in the pool (active or idle). Just
		// bump its refcount — no eviction, no cooldown, no slot consumed.
		if hs, ok := l.pool[host]; ok {
			hs.refcount++
			hs.lastUsed = time.Now()
			return l.makeSlot(host), nil
		}

		// Host is new to this pool. If there's room without eviction, take
		// it — but still respect any cooldown left over from a recent
		// eviction (the evicted host's connection may still be cooling on
		// the VPN's side).
		if len(l.pool) < l.cap {
			waited, err := cooldownWaitLocked()
			if err != nil {
				return nil, err
			}
			if waited {
				continue // re-check; another goroutine may have raced ahead
			}
			l.pool[host] = &hostState{refcount: 1, lastUsed: time.Now()}
			return l.makeSlot(host), nil
		}

		// Pool is full. Find the oldest idle host to evict.
		victim := l.findOldestIdleLocked()
		if victim == "" {
			// Every host has in-flight requests. Wait for a release.
			if !atCapLogged {
				atCapLogged = true
				slog.Debug(fmt.Sprintf(
					"waiting for free proxy connection slot (pool=%s, host=%s, hosts=%d/%d)",
					poolKey, host, len(l.pool), l.cap,
				))
			}
			waitState.Store(waitStateAtCap)
			l.cond.Wait()
			waitState.Store(waitStateUnknown)
			continue
		}

		// Respect the inter-eviction cooldown before kicking the LRU host.
		waited, err := cooldownWaitLocked()
		if err != nil {
			return nil, err
		}
		if waited {
			continue // re-check: victim or pool state may have shifted
		}

		// Evict the oldest idle host and admit the new one. lastEviction is
		// updated so subsequent admissions wait one cooldown apart.
		delete(l.pool, victim)
		l.lastEviction = time.Now()
		l.pool[host] = &hostState{refcount: 1, lastUsed: time.Now()}
		slog.Debug(fmt.Sprintf(
			"evicted idle host %s for new host %s in pool %s",
			victim, host, poolKey,
		))
		return l.makeSlot(host), nil
	}
}

func (l *ConnectionLimiter) makeSlot(host string) *Slot {
	return &Slot{release: func() {
		l.mu.Lock()
		hs := l.pool[host]
		if hs == nil || hs.refcount == 0 {
			// defensive — should not happen given Slot's sync.Once guard
			l.mu.Unlock()
			return
		}
		hs.refcount--
		hs.lastUsed = time.Now()
		if hs.refcount == 0 {
			// Host went idle. Wake any cap-waiters so they can evict it.
			l.cond.Broadcast()
		}
		l.mu.Unlock()
	}}
}

// Slot represents a held connection budget. Release is idempotent and nil-safe.
type Slot struct {
	release func()
	once    sync.Once
}

// Release returns the slot to its pool. Safe to call multiple times and on nil.
func (s *Slot) Release() {
	if s == nil {
		return
	}
	s.once.Do(func() {
		if s.release != nil {
			s.release()
		}
	})
}

// releasingBody wraps a response body so Close releases the budget slot.
type releasingBody struct {
	io.ReadCloser
	slot *Slot
}

// Close delegates to the underlying body and then releases the slot.
// Idempotent via the Slot's sync.Once.
func (r *releasingBody) Close() error {
	err := r.ReadCloser.Close()
	r.slot.Release()
	return err
}

// WrapBodyWithSlot returns a ReadCloser that releases slot on Close.
// If slot is nil, body is returned unchanged.
//
// As a safety net against callers that forget to Close the response body
// (a benign GC-eventual leak pre-budget, but a hard block under budget),
// a finalizer is registered that releases the slot when the wrapper is
// garbage-collected. The Slot's sync.Once makes the double-release safe.
// This converts "permanent slot leak" into "delayed release on next GC".
func WrapBodyWithSlot(body io.ReadCloser, slot *Slot) io.ReadCloser {
	if slot == nil {
		return body
	}
	rb := &releasingBody{ReadCloser: body, slot: slot}
	runtime.SetFinalizer(rb, func(r *releasingBody) { r.slot.Release() })
	return rb
}
