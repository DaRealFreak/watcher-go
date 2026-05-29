package http

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// LeaseCoordinator hands out per-module reservations against the global
// connection budget. A Lease guarantees a module up to N concurrent host
// slots within a specific (username, domain) pool; while held, the module's
// requests for that pool use the lease's internal limiter instead of the
// shared global pool — so cross-module rotation only contends at lease
// admission, not at every request.
//
// Leases are keyed by (moduleKey, accountKey). accountKey is "username@domain"
// (see PoolKeyFor). The sum of active leases' slot reservations is bounded
// by the per-domain policy's Max; over-asking blocks until other leases
// release.
type LeaseCoordinator struct {
	mu    sync.Mutex
	cond  *sync.Cond
	pools map[string]*leasePool // accountKey -> pool
}

// leasePool aggregates the active leases for one (username, domain) account.
// inUse is the sum of granted leases' slot counts; bounded by policy.Max.
type leasePool struct {
	policy DomainPolicy
	leases map[string]*Lease // moduleKey -> lease
	inUse  int
}

// GlobalLeases is the process-wide lease coordinator. Nil-safe.
var GlobalLeases *LeaseCoordinator

// InitGlobalLeases (re-)initializes the package-global coordinator.
func InitGlobalLeases() { GlobalLeases = NewLeaseCoordinator() }

// NewLeaseCoordinator returns an empty coordinator.
func NewLeaseCoordinator() *LeaseCoordinator {
	c := &LeaseCoordinator{pools: make(map[string]*leasePool)}
	c.cond = sync.NewCond(&c.mu)
	return c
}

// Lease is a module's reservation of N host slots in one (username, domain)
// pool. While held, the module's requests for that pool go through the
// lease's own ConnectionLimiter (LRU + intra-lease cooldown). Release returns
// the reservation to the coordinator so queued waiters can proceed.
type Lease struct {
	coordinator *LeaseCoordinator
	moduleKey   string
	accountKey  string
	slots       int
	limiter     *ConnectionLimiter
	released    bool
}

// AcquireLease blocks until `slots` host slots are available in the
// accountKey pool, then returns a Lease holding that reservation. policy
// provides the pool cap and the cooldown the lease's internal limiter will
// use between intra-lease evictions.
//
// slots is clamped: negative/zero -> 1; greater than policy.Max -> policy.Max.
// If the module already holds a lease for this account, returns an error
// rather than silently merging — explicit Release is required first.
// Returns ctx.Err() if canceled before a reservation could be granted.
func (c *LeaseCoordinator) AcquireLease(ctx context.Context, moduleKey, accountKey string, slots int, policy DomainPolicy) (*Lease, error) {
	if c == nil {
		return nil, fmt.Errorf("nil lease coordinator")
	}
	if policy.Max <= 0 {
		return nil, fmt.Errorf("policy max must be positive (got %d)", policy.Max)
	}
	if slots <= 0 {
		slots = 1
	}
	if slots > policy.Max {
		slots = policy.Max
	}

	// Watcher to wake cond.Wait()ers on ctx cancel.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			c.mu.Lock()
			c.cond.Broadcast()
			c.mu.Unlock()
		case <-stop:
		}
	}()

	c.mu.Lock()
	defer c.mu.Unlock()

	pool, ok := c.pools[accountKey]
	if !ok {
		pool = &leasePool{policy: policy, leases: make(map[string]*Lease)}
		c.pools[accountKey] = pool
	}
	if _, exists := pool.leases[moduleKey]; exists {
		return nil, fmt.Errorf("module %q already holds a lease for %q", moduleKey, accountKey)
	}

	loggedWait := false
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if pool.inUse+slots <= pool.policy.Max {
			lease := &Lease{
				coordinator: c,
				moduleKey:   moduleKey,
				accountKey:  accountKey,
				slots:       slots,
				limiter:     newConnectionLimiter(slots, pool.policy.Cooldown),
			}
			pool.leases[moduleKey] = lease
			pool.inUse += slots
			slog.Debug(fmt.Sprintf(
				"granted lease module=%s account=%s slots=%d (pool=%d/%d)",
				moduleKey, accountKey, slots, pool.inUse, pool.policy.Max,
			))
			return lease, nil
		}
		if !loggedWait {
			loggedWait = true
			slog.Info(fmt.Sprintf(
				"waiting for proxy lease: module=%s account=%s want=%d available=%d/%d",
				moduleKey, accountKey, slots, pool.policy.Max-pool.inUse, pool.policy.Max,
			))
		}
		c.cond.Wait()
	}
}

// Lookup returns the active lease for (moduleKey, accountKey), or nil if
// none. Used by the transport wrapper to route requests through the lease's
// limiter when one is active.
func (c *LeaseCoordinator) Lookup(moduleKey, accountKey string) *Lease {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	pool, ok := c.pools[accountKey]
	if !ok {
		return nil
	}
	return pool.leases[moduleKey]
}

// Acquire returns a slot for the given host within this lease. Multiple
// concurrent requests to the same host piggyback; different hosts within
// the lease are LRU-managed against the lease's slot count.
func (l *Lease) Acquire(ctx context.Context, host string) (*Slot, error) {
	if l == nil {
		return &Slot{}, nil
	}
	return l.limiter.Acquire(ctx, l.moduleKey+"@"+l.accountKey, host)
}

// Release returns the lease's reservation to the coordinator and wakes any
// queued waiters. Idempotent and nil-safe.
func (l *Lease) Release() {
	if l == nil || l.coordinator == nil {
		return
	}
	l.coordinator.mu.Lock()
	defer l.coordinator.mu.Unlock()
	if l.released {
		return
	}
	l.released = true
	pool, ok := l.coordinator.pools[l.accountKey]
	if !ok {
		return
	}
	delete(pool.leases, l.moduleKey)
	pool.inUse -= l.slots
	slog.Debug(fmt.Sprintf(
		"released lease module=%s account=%s slots=%d (pool=%d/%d)",
		l.moduleKey, l.accountKey, l.slots, pool.inUse, pool.policy.Max,
	))
	l.coordinator.cond.Broadcast()
}

// Slots returns the lease's reserved slot count. For diagnostics.
func (l *Lease) Slots() int {
	if l == nil {
		return 0
	}
	return l.slots
}
