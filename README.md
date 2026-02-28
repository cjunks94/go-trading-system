# Go Trading System

A real-time stock trading simulation built in Go, demonstrating WebSocket price streaming, order matching with price-time priority, position/P&L tracking, and risk management.

Built for **Bloomberg Senior Staff Engineer** interview preparation.

## Features

- **Real-time Market Data**: Simulated price feeds using Brownian motion with goroutine-per-symbol architecture
- **Order Book**: Price-time priority matching engine supporting market and limit orders
- **WebSocket Streaming**: Live price updates pushed to connected clients
- **Position Tracking**: Real-time P&L calculation (realized + unrealized)
- **Risk Management**: Position limits, order size limits, and daily loss limits
- **REST API**: Full order management and position querying
- **Web Dashboard**: Interactive trading interface

## Architecture Highlights

| Component | Go Pattern | Purpose |
|-----------|------------|---------|
| Market Feed | Fan-out (goroutine per symbol) | Parallel price generation |
| WebSocket Hub | Pub-sub (register/unregister channels) | Client connection management |
| Order Engine | Worker pool (bounded concurrency) | Order processing |
| Risk Manager | Atomic operations | Lock-free limit checking |
| Graceful Shutdown | Context cancellation | Clean resource cleanup |

## Quick Start

### Prerequisites
- Go 1.22+
- Docker (optional)

### Run Locally

```bash
# Clone and navigate
cd projects/go-trading-system

# Download dependencies
go mod download

# Run the server
go run ./cmd/server

# Open dashboard
open http://localhost:8080
```

### Run with Docker

```bash
docker-compose up --build
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Service health status |
| GET | `/api/quotes` | All current quotes |
| GET | `/api/quotes/{symbol}` | Quote for specific symbol |
| POST | `/api/orders` | Submit new order |
| GET | `/api/orders/{id}` | Get order by ID |
| DELETE | `/api/orders/{id}` | Cancel order |
| GET | `/api/positions` | All positions |
| GET | `/api/positions/{symbol}` | Position for symbol |
| GET | `/api/pnl` | P&L summary |
| WS | `/ws` | WebSocket for live updates |

### Submit Order Example

```bash
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{"symbol":"AAPL","side":"BUY","type":"MARKET","quantity":100}'
```

## Project Structure

```
go-trading-system/
├── cmd/server/main.go          # Application entry point
├── internal/
│   ├── api/                    # HTTP handlers and router
│   ├── config/                 # Configuration management
│   ├── market/                 # Market data feed (simulated)
│   ├── orderbook/              # Order book with matching engine
│   ├── trading/                # Trading engine, positions, risk
│   └── websocket/              # WebSocket hub
├── pkg/models/                 # Domain models
├── web/                        # Dashboard UI
├── docker-compose.yml
├── Dockerfile
└── Makefile
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `SYMBOLS` | `AAPL,GOOGL,MSFT,AMZN,TSLA` | Symbols to track |
| `WORKERS` | `4` | Order processing workers |
| `MAX_POSITION_SIZE` | `10000` | Max shares per symbol |
| `MAX_ORDER_SIZE` | `1000` | Max shares per order |
| `MAX_DAILY_LOSS` | `50000` | Daily loss limit (halts trading) |

## Testing

```bash
# Run all tests
make test

# Run benchmarks
make bench

# Generate coverage report
make coverage
```

## Key Technical Decisions

1. **Heap-based Order Book**: O(log n) insertion and O(1) best price lookup
2. **Buffered Channels**: Backpressure handling with non-blocking sends
3. **Atomic Operations for Risk**: Lock-free reads on hot path
4. **Context Cancellation**: Clean shutdown propagation through goroutine tree

## Interview Talking Points

- **Throughput vs Latency**: Buffered channels (size 1000) batch writes, trading ~10ms latency for higher throughput
- **Backpressure**: Non-blocking sends prevent slow consumers from blocking producers
- **Memory Management**: Reusing order objects with sync.Pool in production
- **Testing**: Benchmark tests validate matching engine handles 100K+ ops/sec

## Technologies

- **Go 1.22**: Goroutines, channels, context, sync primitives
- **Gorilla WebSocket**: Production-grade WebSocket implementation
- **Gorilla Mux**: HTTP router with path variables
- **Docker**: Multi-stage build for minimal image size

## License

MIT
