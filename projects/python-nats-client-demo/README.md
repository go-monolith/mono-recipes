# Python NATS Client Demo

A complete example showing how to build polyglot microservices using Python clients with Go-based Mono applications (mono **v0.1.1**) via NATS messaging.

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
├── main.go                    # Go application entry point (embedded NATS, strict boundaries)
├── go.mod                     # Go module definition
├── modules/
│   ├── math/                  # RequestReplyService
│   │   ├── contract/         # Published port: DTOs + Calculate contract
│   │   ├── service.go        # Calculator operations
│   │   ├── module.go         # RegisterServices via the contract
│   │   └── module_test.go    # monotest.CheckModule conformance test
│   ├── notification/          # QueueGroupService
│   │   ├── contract/         # Published port: SendEmail contract
│   │   ├── service.go        # Email processing
│   │   ├── module.go         # RegisterServices via the contract
│   │   └── module_test.go    # monotest.CheckModule conformance test
│   └── payment/               # StreamConsumerService + RequestReplyService
│       ├── contract/         # Published port: ProcessPayment + Status contracts
│       ├── types.go          # Internal payment result type
│       ├── service.go        # Payment processing
│       ├── module.go         # RegisterServices via the contracts
│       └── module_test.go    # Real-app test (JetStream) with boundary report
├── client/                    # Python client library
│   ├── __init__.py
│   ├── math_client.py        # RequestReply client
│   ├── email_client.py       # QueueGroup client
│   └── payment_client.py     # JetStream client
├── tests/                     # Python client unit tests (pytest, mocked NATS)
├── demo.py                    # Interactive demo script
├── requirements.txt           # Python dependencies
└── README.md
```

## Prerequisites

- Go 1.25+
- Python 3.10+

No separate NATS server is needed: the Go application embeds NATS with JetStream (storage in `/tmp/python-nats-client-demo`).

## Quick Start

### 1. Start the Go Server

```bash
cd projects/python-nats-client-demo
go build -o bin/python-nats-client-demo . && ./bin/python-nats-client-demo
# or: go run .
```

NATS listens on `nats://localhost:4222`; set `NATS_PORT` to use another port (and pass `--nats-url nats://localhost:<port>` to `demo.py`).

You should see:
```
[math] Registered services: services.math.calculate
[notification] Registered QueueGroupService: services.notification.email-send (queue: email-workers)
[payment] Registered StreamConsumerService: services.payment.payment-process (stream: payment-requests)
[payment] Registered RequestReplyService: services.payment.status
Application started successfully!
```

### 2. Run the Python Demo

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

1. **Type Safety in Go** - The Go services are registered through typed contracts (JSON on the wire)
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

## Request Deadlines

Since mono v0.0.17, request-reply calls made from Go carry the caller's deadline in two headers, `Mono-Deadline` (absolute, Unix milliseconds) and `Mono-Timeout` (remaining, milliseconds). The Go server drops a request whose deadline already passed while it waited, and the handler context ends at that deadline. The Python clients send plain `nc.request(...)` without these headers, so their requests are never expired server-side; only the client `timeout` applies. A Python client can opt in by sending the headers itself:

```python
import time
timeout = 5.0
headers = {
    "Mono-Deadline": str(int((time.time() + timeout) * 1000)),
    "Mono-Timeout": str(int(timeout * 1000)),
}
response = await nc.request("services.math.calculate", payload, timeout=timeout, headers=headers)
```

Request-reply handlers run concurrently on a worker pool (v0.0.15+), so `math.calculate` and `payment.status` serve parallel Python requests; the payment store is guarded by a mutex.

## Module Boundaries

The Go side follows the module boundaries path of mono v0.1.x:

- **Contract packages**: each module has `modules/<module>/contract` with its DTOs and typed contracts (`helper.RequestReplyService`, `helper.QueueGroupService`, `helper.StreamConsumerService`). The contracts fix the subjects the Python clients use, so they are declared once and never written as strings in module code. The payment stream's subject is taken from `contract.ProcessPayment.Subject()`.
- **No cross-module calls**: the three modules are independent providers called only from Python, so none implements `ConsumeServices()`.
- **Malformed stream messages**: the typed stream consumer fails a whole batch when one message does not decode. The payment module registers `ProcessPayment` with a lenient decoder so a malformed message is terminated instead of holding up valid payments.
- **Strict mode**: `main.go` runs with `mono.WithStrictModuleBoundaries()`.
- **Conformance tests**: `monotest.CheckModule` for `math` and `notification`; `payment` uses JetStream, which `monotest` does not emulate, so `modules/payment/module_test.go` runs it in a real application (temp JetStream dir, default mode so violations are recorded) and asserts that `app.ModuleBoundaries()` reports no violations.

## Testing

### Run Go Tests

```bash
go test ./...
```

### Run Python Tests

```bash
pip install -r requirements.txt
pytest tests/ -v
```

## Learn More

- [NATS Documentation](https://docs.nats.io/)
- [nats-py Client](https://github.com/nats-io/nats.py)
- [Mono Framework](https://github.com/go-monolith/mono)
- [JetStream](https://docs.nats.io/nats-concepts/jetstream)
