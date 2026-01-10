# Polyglot Client Patterns

Mono applications can be consumed by clients written in any language via NATS messaging. This reference covers Python (nats.py) and Node.js (nats.js) client patterns.

## NATS Subject Conventions

Mono framework uses these subject patterns:

| Service Type | Subject Pattern | Example |
|--------------|-----------------|---------|
| RequestReplyService | `services.<module>.<service>` | `services.math.calculate` |
| QueueGroupService | `services.<module>.<service>` | `services.notification.email-send` |
| StreamConsumerService | `services.<module>.<service>` | `services.payment.payment-process` |

## Python Client (nats.py)

### Installation

```bash
pip install nats-py
# or
uv add nats-py
```

### Basic Client Structure

```python
"""NATS client for Mono services."""

import json
from typing import Any

import nats


class MonoClient:
    """Base client for Mono service communication."""

    def __init__(self, url: str = "nats://localhost:4222"):
        """Initialize the client.

        Args:
            url: NATS server URL.
        """
        self._url = url
        self._nc: nats.NATS = None

    async def connect(self) -> None:
        """Connect to NATS server."""
        self._nc = await nats.connect(self._url)

    async def close(self) -> None:
        """Close the NATS connection."""
        if self._nc:
            await self._nc.drain()

    async def request_reply(
        self, module: str, service: str, request: dict, timeout: float = 5.0
    ) -> dict:
        """Call a RequestReplyService.

        Args:
            module: Module name (e.g., "math").
            service: Service name (e.g., "calculate").
            request: Request payload.
            timeout: Request timeout in seconds.

        Returns:
            Response as dictionary.
        """
        subject = f"services.{module}.{service}"
        response = await self._nc.request(
            subject,
            json.dumps(request).encode(),
            timeout=timeout,
        )
        return json.loads(response.data)

    async def publish_queue(
        self, module: str, service: str, request: dict
    ) -> None:
        """Publish to a QueueGroupService (fire-and-forget).

        Args:
            module: Module name.
            service: Service name.
            request: Request payload.
        """
        subject = f"services.{module}.{service}"
        await self._nc.publish(subject, json.dumps(request).encode())
```

### Service-Specific Client

```python
"""Math client for RequestReplyService interactions."""

import json
from typing import Optional

import nats


class MathClient:
    """Client for math.calculate RequestReplyService."""

    def __init__(self, nc: nats.NATS):
        """Initialize the math client.

        Args:
            nc: An active NATS connection.
        """
        self._nc = nc

    async def calculate(
        self, operation: str, a: float, b: float = 0.0, timeout: float = 5.0
    ) -> dict:
        """Perform a math calculation.

        Args:
            operation: One of "add", "subtract", "multiply", "divide", "power", "sqrt".
            a: First operand.
            b: Second operand.
            timeout: Request timeout in seconds.

        Returns:
            dict with "result" and "operation", or "error" if failed.
        """
        request = {"operation": operation, "a": a, "b": b}
        response = await self._nc.request(
            "services.math.calculate",
            json.dumps(request).encode(),
            timeout=timeout,
        )
        return json.loads(response.data)

    async def add(self, a: float, b: float) -> float:
        """Add two numbers."""
        result = await self.calculate("add", a, b)
        return result.get("result", 0)

    async def divide(self, a: float, b: float) -> Optional[float]:
        """Divide a by b. Returns None on error."""
        result = await self.calculate("divide", a, b)
        if "error" in result:
            return None
        return result.get("result")
```

### JetStream Client for Stream Operations

```python
"""Payment client for StreamConsumerService interactions."""

import json

import nats
from nats.js import JetStreamContext


class PaymentClient:
    """Client for payment.process StreamConsumerService."""

    def __init__(self, nc: nats.NATS):
        self._nc = nc
        self._js: JetStreamContext = None

    async def init(self) -> None:
        """Initialize JetStream context."""
        self._js = self._nc.jetstream()

    async def submit_payment(
        self, payment_id: str, user_id: str, amount: float
    ) -> None:
        """Submit a payment for processing.

        Args:
            payment_id: Unique payment identifier.
            user_id: User identifier.
            amount: Payment amount.
        """
        request = {
            "payment_id": payment_id,
            "user_id": user_id,
            "amount": amount,
        }
        await self._js.publish(
            "services.payment.payment-process",
            json.dumps(request).encode(),
        )

    async def get_status(self, payment_id: str) -> dict:
        """Query payment status via RequestReplyService."""
        request = {"payment_id": payment_id}
        response = await self._nc.request(
            "services.payment.status",
            json.dumps(request).encode(),
            timeout=5.0,
        )
        return json.loads(response.data)
```

### Demo Application

```python
#!/usr/bin/env python3
"""Python NATS Client Demo."""

import asyncio
import nats


async def main():
    # Connect to NATS
    nc = await nats.connect("nats://localhost:4222")

    try:
        # RequestReplyService demo
        response = await nc.request(
            "services.math.calculate",
            b'{"operation": "add", "a": 10, "b": 5}',
            timeout=5.0,
        )
        print(f"10 + 5 = {response.data.decode()}")

        # QueueGroupService demo (fire-and-forget)
        await nc.publish(
            "services.notification.email-send",
            b'{"to": "user@example.com", "subject": "Hello"}',
        )
        print("Email queued")

    finally:
        await nc.drain()


if __name__ == "__main__":
    asyncio.run(main())
```

## Node.js Client (nats.js)

### Installation

```bash
npm install nats
# or
yarn add nats
```

### Basic Client Structure

```javascript
// client/nats-client.js
import { connect } from 'nats';

/**
 * NATSClient manages connection to NATS server
 */
export class NATSClient {
  /** @type {string} */
  url;
  /** @type {import('nats').NatsConnection | null} */
  nc = null;

  /**
   * @param {string} url - NATS server URL
   */
  constructor(url = 'nats://localhost:4222') {
    this.url = url;
  }

  /**
   * Connect to NATS server
   * @returns {Promise<import('nats').NatsConnection>}
   */
  async connect() {
    this.nc = await connect({ servers: this.url });
    console.log(`Connected to NATS at ${this.url}`);
    return this.nc;
  }

  /**
   * Disconnect from NATS server
   * @returns {Promise<void>}
   */
  async disconnect() {
    if (!this.nc) return;
    await this.nc.drain();
    console.log('Disconnected from NATS');
  }

  /**
   * Get the active NATS connection
   * @returns {import('nats').NatsConnection}
   */
  getConnection() {
    if (!this.nc) {
      throw new Error('Not connected. Call connect() first.');
    }
    return this.nc;
  }
}
```

### Service-Specific Client

```javascript
// client/file-service.js
const SERVICE_UNAVAILABLE_CODE = '503';

/**
 * @typedef {Object} SaveFileResult
 * @property {string} file_id - Unique file identifier
 * @property {string} filename - Name of the saved file
 * @property {number} size - File size in bytes
 * @property {string} digest - File content digest
 */

/**
 * FileService wraps RequestReplyService for file.save operations
 */
export class FileService {
  /** @type {import('nats').NatsConnection} */
  nc;
  /** @type {string} */
  subject = 'services.fileops.save';

  /**
   * @param {import('nats').NatsConnection} nc - NATS connection
   */
  constructor(nc) {
    this.nc = nc;
  }

  /**
   * Save a JSON file to the bucket
   * @param {string} filename - Name of the file
   * @param {object} content - JSON content to save
   * @param {number} timeout - Request timeout in milliseconds
   * @returns {Promise<SaveFileResult>}
   */
  async saveFile(filename, content, timeout = 5000) {
    const request = { filename, content };
    return this.request(request, timeout);
  }

  /**
   * @param {object} request - Request payload
   * @param {number} timeout - Request timeout in milliseconds
   * @returns {Promise<SaveFileResult>}
   * @private
   */
  async request(request, timeout) {
    let response;
    try {
      response = await this.nc.request(
        this.subject,
        JSON.stringify(request),
        { timeout }
      );
    } catch (error) {
      if (error.code === SERVICE_UNAVAILABLE_CODE) {
        throw new Error('Service unavailable. Make sure Go server is running.');
      }
      throw error;
    }

    const result = JSON.parse(new TextDecoder().decode(response.data));

    if (result.error) {
      throw new Error(result.error);
    }

    return result;
  }
}
```

### QueueGroupService Client (Fire-and-Forget)

```javascript
// client/archive-service.js
/**
 * ArchiveService wraps QueueGroupService for file.archive operations
 */
export class ArchiveService {
  /** @type {import('nats').NatsConnection} */
  nc;
  /** @type {string} */
  subject = 'services.fileops.archive';

  /**
   * @param {import('nats').NatsConnection} nc - NATS connection
   */
  constructor(nc) {
    this.nc = nc;
  }

  /**
   * Queue a file for archiving (fire-and-forget)
   * @param {string} fileId - File ID to archive
   */
  archiveFile(fileId) {
    const request = { file_id: fileId };
    this.nc.publish(this.subject, JSON.stringify(request));
  }
}
```

### JetStream Object Store Watcher

```javascript
// client/watcher.js
/**
 * @typedef {Object} WatchEvent
 * @property {string} bucket - Bucket name
 * @property {string} name - File name
 * @property {boolean} deleted - Whether the file was deleted
 * @property {number} size - File size in bytes
 * @property {Date} timestamp - Event timestamp
 */

/**
 * BucketWatcher monitors JetStream object store for file changes
 */
export class BucketWatcher {
  /** @type {import('nats').NatsConnection} */
  nc;
  /** @type {string} */
  bucketName;
  /** @type {import('nats').ObjectStore | null} */
  os = null;
  /** @type {import('nats').ObjectStoreStatus | null} */
  watchSub = null;

  /**
   * @param {import('nats').NatsConnection} nc - NATS connection
   * @param {string} bucketName - Name of the object store bucket
   */
  constructor(nc, bucketName = 'user-settings') {
    this.nc = nc;
    this.bucketName = bucketName;
  }

  /**
   * Initialize the object store connection
   */
  async initialize() {
    const js = this.nc.jetstream();
    this.os = await js.views.os(this.bucketName);
    console.log(`Initialized object store: ${this.bucketName}`);
  }

  /**
   * Watch the bucket for changes
   * @param {function} callback - Function to call for each change event
   */
  async watch(callback) {
    if (!this.os) {
      throw new Error('Watcher not initialized. Call initialize() first.');
    }

    this.watchSub = await this.os.watch({
      includeHistory: false,
      ignoreDeletes: false,
    });

    console.log(`Watching bucket: ${this.bucketName}`);
    this.processEvents(callback);
  }

  /**
   * @private
   */
  async processEvents(callback) {
    for await (const entry of this.watchSub) {
      if (!entry) continue;

      const event = {
        bucket: entry.bucket,
        name: entry.name,
        deleted: entry.deleted ?? false,
        size: entry.size,
        timestamp: new Date(),
      };
      await callback(event);
    }
  }

  /**
   * Stop watching
   */
  async stop() {
    if (this.watchSub) {
      await this.watchSub.stop();
      console.log('Watcher stopped');
    }
  }
}
```

### Demo Application

```javascript
// demo.js
import { NATSClient, FileService, ArchiveService, BucketWatcher } from './client/index.js';

async function runDemo(natsUrl) {
  const client = new NATSClient(natsUrl);
  let watcher;

  try {
    await client.connect();
    const nc = client.getConnection();

    const fileService = new FileService(nc);
    const archiveService = new ArchiveService(nc);

    // Start watching the bucket
    watcher = new BucketWatcher(nc, 'user-settings');
    await watcher.initialize();
    await watcher.watch((event) => {
      console.log(`File ${event.deleted ? 'deleted' : 'created'}: ${event.name}`);
    });

    // Save files via RequestReplyService
    const result = await fileService.saveFile('settings.json', { theme: 'dark' });
    console.log(`Saved: ${result.filename} (${result.size} bytes)`);

    // Archive via QueueGroupService (fire-and-forget)
    archiveService.archiveFile(result.file_id);
    console.log('Archive queued');

  } finally {
    if (watcher) await watcher.stop();
    await client.disconnect();
  }
}

runDemo(process.argv[2] || 'nats://localhost:4222');
```

## Error Handling

### Python

```python
import nats
from nats.errors import TimeoutError, NoRespondersError

try:
    response = await nc.request(subject, data, timeout=5.0)
except TimeoutError:
    print("Request timed out")
except NoRespondersError:
    print("No service available")
except Exception as e:
    print(f"Error: {e}")
```

### Node.js

```javascript
const SERVICE_UNAVAILABLE_CODE = '503';

try {
  const response = await nc.request(subject, data, { timeout: 5000 });
} catch (error) {
  if (error.code === SERVICE_UNAVAILABLE_CODE) {
    console.log('No service available');
  } else if (error.code === 'TIMEOUT') {
    console.log('Request timed out');
  } else {
    throw error;
  }
}
```

## Example Projects

| Project | Language | Features |
|---------|----------|----------|
| `python-nats-client-demo` | Python | RequestReply, QueueGroup, StreamConsumer |
| `node-nats-client-demo` | Node.js | RequestReply, QueueGroup, ObjectStore watcher |

## Best Practices

1. **Reuse connections** - Create one NATS connection and share it across clients
2. **Use typed clients** - Create service-specific wrapper classes
3. **Handle errors gracefully** - Catch timeout and no-responders errors
4. **Set appropriate timeouts** - Match server-side processing expectations
5. **Use JetStream for durability** - StreamConsumerService for at-least-once delivery
6. **Drain connections on shutdown** - Use `drain()` instead of `close()`
7. **Log connection events** - Track connect/disconnect for debugging
