"""Math client for RequestReplyService interactions."""

import json
from typing import Optional

import nats


class MathClient:
    """Client for math.calculate RequestReplyService.

    Demonstrates synchronous request-reply pattern where the client
    waits for a response from the Go service.
    """

    def __init__(self, nc: nats.NATS):
        """Initialize the math client.

        Args:
            nc: An active NATS connection.
        """
        self._nc = nc

    async def calculate(
        self, operation: str, a: float, b: float = 0.0, timeout: float = 5.0
    ) -> dict:
        """Perform a math calculation via RequestReplyService.

        Args:
            operation: One of "add", "subtract", "multiply", "divide", "power", "sqrt".
            a: First operand.
            b: Second operand (optional for sqrt).
            timeout: Request timeout in seconds.

        Returns:
            dict with "result" and "operation", or "error" if operation failed.
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

    async def subtract(self, a: float, b: float) -> float:
        """Subtract b from a."""
        result = await self.calculate("subtract", a, b)
        return result.get("result", 0)

    async def multiply(self, a: float, b: float) -> float:
        """Multiply two numbers."""
        result = await self.calculate("multiply", a, b)
        return result.get("result", 0)

    async def divide(self, a: float, b: float) -> Optional[float]:
        """Divide a by b. Returns None if division by zero."""
        result = await self.calculate("divide", a, b)
        if "error" in result:
            return None
        return result.get("result")

    async def power(self, base: float, exponent: float) -> float:
        """Calculate base raised to exponent."""
        result = await self.calculate("power", base, exponent)
        return result.get("result", 0)

    async def sqrt(self, value: float) -> Optional[float]:
        """Calculate square root. Returns None for negative numbers."""
        result = await self.calculate("sqrt", value)
        if "error" in result:
            return None
        return result.get("result")
