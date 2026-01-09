"""Email client for QueueGroupService interactions."""

import json

import nats


class EmailClient:
    """Client for notification.email-send QueueGroupService.

    Demonstrates fire-and-forget pattern where the client publishes
    messages to a queue without waiting for a response.
    """

    def __init__(self, nc: nats.NATS):
        """Initialize the email client.

        Args:
            nc: An active NATS connection.
        """
        self._nc = nc

    async def send_email(self, to: str, subject: str, body: str) -> None:
        """Send an email via QueueGroupService (fire-and-forget).

        Messages are load-balanced across all workers in the queue group.
        No response is expected or returned.

        Args:
            to: Recipient email address.
            subject: Email subject line.
            body: Email body content.
        """
        request = {"to": to, "subject": subject, "body": body}
        await self._nc.publish(
            "services.notification.email-send",
            json.dumps(request).encode(),
        )

    async def send_bulk_emails(
        self, emails: list[tuple[str, str, str]]
    ) -> int:
        """Send multiple emails in bulk.

        Args:
            emails: List of (to, subject, body) tuples.

        Returns:
            Number of emails queued.
        """
        for to, subject, body in emails:
            await self.send_email(to, subject, body)
        return len(emails)
