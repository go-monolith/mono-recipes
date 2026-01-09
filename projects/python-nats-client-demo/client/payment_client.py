"""Payment client for StreamConsumerService interactions."""

import json
from typing import Optional

import nats
from nats.js import JetStreamContext


class PaymentClient:
    """Client for payment.payment-process StreamConsumerService.

    Demonstrates durable message processing via JetStream where messages
    are persisted and processed with at-least-once delivery semantics.
    """

    def __init__(self, nc: nats.NATS):
        """Initialize the payment client.

        Args:
            nc: An active NATS connection.
        """
        self._nc = nc
        self._js: Optional[JetStreamContext] = None

    async def _get_jetstream(self) -> JetStreamContext:
        """Get or create JetStream context."""
        if self._js is None:
            self._js = self._nc.jetstream()
        return self._js

    async def submit_payment(
        self,
        payment_id: str,
        user_id: str,
        subscription_id: str,
        amount: float,
    ) -> None:
        """Submit a payment for processing via JetStream.

        The payment will be durably stored and processed by the
        StreamConsumerService with acknowledgment handling.

        Args:
            payment_id: Unique payment identifier.
            user_id: User making the payment.
            subscription_id: Subscription being paid for.
            amount: Payment amount.
        """
        js = await self._get_jetstream()
        request = {
            "payment_id": payment_id,
            "user_id": user_id,
            "subscription_id": subscription_id,
            "amount": amount,
        }
        await js.publish(
            "services.payment.payment-process",
            json.dumps(request).encode(),
        )

    async def get_status(
        self, payment_id: str, timeout: float = 5.0
    ) -> dict:
        """Query payment status via RequestReplyService.

        Args:
            payment_id: Payment identifier to query.
            timeout: Request timeout in seconds.

        Returns:
            dict with "payment_id", "status", and optional "message".
        """
        request = {"payment_id": payment_id}
        response = await self._nc.request(
            "services.payment.status",
            json.dumps(request).encode(),
            timeout=timeout,
        )
        return json.loads(response.data)
