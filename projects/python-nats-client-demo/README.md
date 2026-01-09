# Python NATS Client Demo

A complete example showing how to build polyglot microservices using Python clients with Go-based Mono applications via NATS messaging.

## Why Python NATS Client for Polyglot Microservices?

### The Challenge

Modern systems often need to combine:
- **Go services** for high-performance, concurrent workloads
- **Python scripts** for data processing, ML inference, or automation
- **Other languages** for specialized tasks

Traditional REST APIs introduce latency, require HTTP infrastructure, and create tight coupling between services.

### The Solution: NATS as a Universal Message Bus

NATS provides a language-agnostic messaging layer that enables:

1. **Zero HTTP overhead** - Direct binary messaging over TCP
2. **Natural async patterns** - Fire-and-forget, request-reply, streaming
3. **Decoupled architecture** - Services discover each other via subjects
4. **Polyglot by design** - Clients exist for 40+ languages

## Service Patterns Demonstrated

This demo showcases three distinct messaging patterns, each suited for different use cases:

### 1. RequestReplyService (math.calculate)

**Pattern**: Synchronous request-response

**Use when**:
- You need an immediate response
- The operation is stateless and fast
- Similar to traditional RPC/REST calls

**Example**: Calculator operations where the client waits for the result.

```python
# Python client
response = await nc.request("services.math.calculate",
    json.dumps({"operation": "add", "a": 10, "b": 5}).encode())
result = json.loads(response.data)  # {"result": 15}
```

### 2. QueueGroupService (notification.email-send)

**Pattern**: Fire-and-forget with load balancing

**Use when**:
- You don't need a response
- Work should be distributed across multiple workers
- Delivery to at least one worker is sufficient

**Example**: Sending email notifications where you queue the request and continue.

```python
# Python client - no response expected
await nc.publish("services.notification.email-send",
    json.dumps({"to": "user@example.com", "subject": "Hello"}).encode())
```

### 3. StreamConsumerService (payment.payment-process)

**Pattern**: Durable streaming with acknowledgments

**Use when**:
- Messages must never be lost
- Processing can be delayed or retry on failure
- You need exactly-once or at-least-once delivery
- Audit trails are required

**Example**: Payment processing where each transaction must be recorded and processed reliably.

```python
# Python client - publish to JetStream
js = nc.jetstream()
await js.publish("services.payment.payment-process",
    json.dumps({"payment_id": "pay-001", "amount": 99.99}).encode())
```

## When to Use Each Pattern

| Pattern | Durability | Response | Use Case |
|---------|------------|----------|----------|
| RequestReply | None | Immediate | Queries, calculations, validation |
| QueueGroup | None | None | Notifications, logging, metrics |
| StreamConsumer | Persistent | Async | Transactions, orders, audit events |

## Project Structure

```
python-nats-client-demo/
├── main.go                    # Go application entry point
├── go.mod                     # Go module definition
├── modules/
│   ├── math/                  # RequestReplyService
│   │   ├── types.go          # Request/response types
│   │   ├── service.go        # Calculator operations
│   │   └── module.go         # Service registration
│   ├── notification/          # QueueGroupService
│   │   ├── types.go          # Email request type
│   │   ├── service.go        # Email processing
│   │   └── module.go         # Service registration
│   └── payment/               # StreamConsumerService
│       ├── types.go          # Payment types
│       ├── service.go        # Payment processing
│       └── module.go         # Service registration
├── client/                    # Python client library
│   ├── __init__.py
│   ├── math_client.py        # RequestReply client
│   ├── email_client.py       # QueueGroup client
│   └── payment_client.py     # JetStream client
├── demo.py                    # Interactive demo script
├── requirements.txt           # Python dependencies
└── README.md
```

## Prerequisites

- Go 1.21+
- Python 3.10+
- Docker and Docker Compose (for NATS)

## Quick Start

### 1. Start NATS with JetStream

```bash
docker run -d --name nats -p 4222:4222 nats:latest -js
```

### 2. Start the Go Server

```bash
cd projects/python-nats-client-demo
go run .
```

You should see:
```
[math] Registered services: services.math.calculate
[notification] Registered services: services.notification.email-send
[payment] Registered services: services.payment.payment-process, services.payment.status
Starting Mono application...
```

### 3. Run the Python Demo

In a new terminal:

```bash
cd projects/python-nats-client-demo

# Install dependencies
pip install -r requirements.txt

# Run full demo
python demo.py

# Or run specific demos
python demo.py --math-only      # RequestReplyService only
python demo.py --email-only     # QueueGroupService only
python demo.py --payment-only   # StreamConsumerService only
```

## Using the Python Client Library

The `client/` directory provides reusable Python classes:

```python
import asyncio
import nats
from client import MathClient, EmailClient, PaymentClient

async def main():
    nc = await nats.connect("nats://localhost:4222")

    # RequestReplyService - synchronous math
    math = MathClient(nc)
    result = await math.add(10, 5)  # Returns 15

    # QueueGroupService - fire-and-forget email
    email = EmailClient(nc)
    await email.send_email("user@example.com", "Subject", "Body")

    # StreamConsumerService - durable payment
    payment = PaymentClient(nc)
    await payment.submit_payment("pay-001", "user-123", "sub-monthly", 9.99)
    status = await payment.get_status("pay-001")  # {"status": "completed"}

    await nc.drain()

asyncio.run(main())
```

## Architecture Benefits

### Why Mono + Python NATS?

1. **Type Safety in Go** - The Go services use typed handlers with JSON marshaling
2. **Flexibility in Python** - Python scripts can interact without code generation
3. **Shared Nothing** - Services communicate only via messages
4. **Independent Scaling** - Add more Go workers or Python clients as needed
5. **Operational Simplicity** - NATS is a single binary with minimal config

### Comparison to Alternatives

| Approach | Pros | Cons |
|----------|------|------|
| REST APIs | Familiar, tooling | HTTP overhead, tight coupling |
| gRPC | Type-safe, efficient | Code generation, complexity |
| NATS | Simple, polyglot, patterns | Less tooling, different paradigm |

## Testing

### Run Go Tests

```bash
go test ./...
```

### Run Python Tests

```bash
pytest tests/ -v
```

## Learn More

- [NATS Documentation](https://docs.nats.io/)
- [nats-py Client](https://github.com/nats-io/nats.py)
- [Mono Framework](https://github.com/go-monolith/mono)
- [JetStream](https://docs.nats.io/nats-concepts/jetstream)
