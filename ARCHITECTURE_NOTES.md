# Architecture Notes: Go Trading System

## Problem Statement

Build a demonstration trading system that showcases:
1. Real-time data streaming capabilities
2. Low-latency order processing
3. Go concurrency patterns relevant to financial systems
4. Production-quality code for senior engineer interview

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Go Trading System                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │
│  │   Market     │───▶│    Price     │───▶│  WebSocket   │──▶ Clients│
│  │   Feed       │    │   Channel    │    │    Hub       │          │
│  │ (goroutines) │    │ (buffered)   │    │  (pub-sub)   │          │
│  └──────────────┘    └──────────────┘    └──────────────┘          │
│        │                                                             │
│        ▼                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │
│  │   REST       │───▶│   Trading    │───▶│    Order     │          │
│  │   API        │    │   Engine     │    │    Book      │          │
│  │              │    │              │    │ (heap-based) │          │
│  └──────────────┘    └──────────────┘    └──────────────┘          │
│        │                    │                                        │
│        │                    ▼                                        │
│        │             ┌──────────────┐    ┌──────────────┐          │
│        │             │  Position    │───▶│    Risk      │          │
│        │             │  Manager     │    │   Manager    │          │
│        │             │              │    │  (atomic)    │          │
│        │             └──────────────┘    └──────────────┘          │
│        │                                                             │
└────────┴─────────────────────────────────────────────────────────────┘
```

## Concurrency Design

### 1. Market Data Feed (Fan-Out Pattern)

**Problem**: Generate realistic price updates for multiple symbols concurrently.

**Solution**: One goroutine per symbol, aggregating into a central channel with fan-out to subscribers.

```go
// One producer goroutine per symbol
for _, symbol := range symbols {
    go f.generatePrices(symbol, basePrice)
}

// Broadcaster goroutine fans out to subscribers
go f.broadcast()
```

**Trade-offs**:
- (+) Symbols are independent - no coordination needed
- (+) Easy to add/remove symbols
- (-) More goroutines = more scheduling overhead
- Mitigation: Symbols are bounded (typically <100)

### 2. Order Book (Heap with RWMutex)

**Problem**: Support fast order matching while allowing concurrent reads.

**Solution**: Min/max heaps for asks/bids with sync.RWMutex for thread safety.

```go
type OrderBook struct {
    bids   *orderHeap  // Max-heap (highest price first)
    asks   *orderHeap  // Min-heap (lowest price first)
    mu     sync.RWMutex
}
```

**Why Heap?**:
- O(log n) insertion
- O(1) best price lookup
- O(log n) extraction
- Price-time priority naturally maintained

**Trade-offs**:
- (+) Excellent read performance with RWMutex
- (-) Writers block readers during matching
- Mitigation: Matching is fast (sub-millisecond)

### 3. WebSocket Hub (Pub-Sub Pattern)

**Problem**: Broadcast messages to multiple clients efficiently.

**Solution**: Central hub with register/unregister channels and broadcast distribution.

```go
type Hub struct {
    clients    map[*Client]bool
    register   chan *Client
    unregister chan *Client
    broadcast  chan []byte
}
```

**Non-blocking Sends**:
```go
select {
case client.send <- message:
default:
    // Client buffer full - disconnect
    close(client.send)
    delete(h.clients, client)
}
```

**Trade-offs**:
- (+) Slow clients don't block fast ones
- (+) Automatic cleanup of unresponsive clients
- (-) Messages can be dropped
- Acceptable for market data (stale prices are worse than missing one)

### 4. Risk Manager (Atomic Operations)

**Problem**: Check risk limits on every order without lock contention.

**Solution**: Atomic values for frequently-read state.

```go
type RiskManager struct {
    dailyPnL atomic.Value  // float64
    breached atomic.Bool   // Trading halted flag
}
```

**Why Atomic?**:
- Lock-free reads (hot path for order validation)
- Only writers need synchronization
- Risk checks happen on every order

### 5. Graceful Shutdown (Context Propagation)

**Problem**: Clean shutdown without data loss or orphaned goroutines.

**Solution**: Context cancellation tree propagated through all components.

```go
func (e *Engine) Start() error {
    e.ctx, e.cancel = context.WithCancel(context.Background())

    go func() {
        for {
            select {
            case <-e.ctx.Done():
                return  // Clean exit
            case order := <-e.orderQueue:
                // Process order
            }
        }
    }()
}
```

## Data Flow

### Order Lifecycle

```
1. Client submits order (REST POST /api/orders)
        │
        ▼
2. API handler validates request
        │
        ▼
3. Risk manager checks limits (atomic read)
        │
        ├── Rejected? Return error
        ▼
4. Order book matches against resting orders
        │
        ├── Generates trades
        ▼
5. Position manager updates holdings
        │
        ▼
6. WebSocket broadcasts updates to clients
```

### Market Data Flow

```
1. Simulated feed generates quotes (per-symbol goroutine)
        │
        ▼
2. Quotes sent to aggregation channel (buffered, 1000)
        │
        ▼
3. Broadcaster fans out to subscribers
        │
        ├── Position manager (updates unrealized P&L)
        ├── WebSocket hub (broadcasts to clients)
        └── (Future: persistence, analytics)
```

## Performance Considerations

### Backpressure Handling

All channels use non-blocking sends:
```go
select {
case ch <- msg:
    // Sent
default:
    // Channel full - drop or handle
}
```

This prevents:
- Slow consumers blocking fast producers
- Memory growth from unbounded queues
- Deadlocks from full channels

### Memory Allocation

Current: New objects per order/trade (simple, correct)

Production optimization (not implemented):
- `sync.Pool` for order objects
- Pre-allocated trade slices
- Zero-allocation JSON encoding

### Latency Budget

| Operation | Target | Notes |
|-----------|--------|-------|
| Quote generation | <1ms | Simple math |
| Order validation | <100μs | Atomic reads |
| Order matching | <1ms | Heap operations |
| WebSocket broadcast | <10ms | Batched writes |
| End-to-end order | <50ms | Total round-trip |

## Testing Strategy

### Unit Tests
- Order book matching logic
- Position P&L calculations
- Risk limit enforcement

### Benchmark Tests
- Order matching throughput (ops/sec)
- Memory allocations per operation

### Integration Tests
- End-to-end order flow
- WebSocket message delivery

## Future Enhancements

1. **Persistence**: Add PostgreSQL for order/trade history
2. **Authentication**: JWT tokens for API access
3. **Metrics**: Prometheus metrics export
4. **Tracing**: OpenTelemetry for distributed tracing
5. **Real Market Data**: Polygon.io or Alpaca integration
6. **Order Types**: Stop-loss, trailing stop, OCO

## Key Learnings

1. **Channels are coordination, not queues**: Use buffered channels for decoupling, not as unlimited queues
2. **Atomic vs Mutex**: Use atomics for single values read frequently, mutexes for complex state
3. **Context propagation**: Essential for clean shutdown in concurrent systems
4. **Non-blocking sends**: Critical for preventing cascade failures
