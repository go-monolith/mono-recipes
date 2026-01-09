"""Tests for the EmailClient class."""

import json
from unittest.mock import AsyncMock, MagicMock

import pytest

from client.email_client import EmailClient


@pytest.fixture
def mock_nc():
    """Create a mock NATS connection."""
    nc = MagicMock()
    nc.publish = AsyncMock()
    return nc


@pytest.fixture
def email_client(mock_nc):
    """Create an EmailClient with mocked connection."""
    return EmailClient(mock_nc)


class TestEmailClientSendEmail:
    """Tests for the send_email method."""

    @pytest.mark.asyncio
    async def test_send_email_publishes_correct_subject(
        self, email_client, mock_nc
    ):
        """Test that send_email publishes to the correct subject."""
        await email_client.send_email(
            "user@example.com", "Test Subject", "Test Body"
        )

        mock_nc.publish.assert_called_once()
        call_args = mock_nc.publish.call_args
        assert call_args[0][0] == "services.notification.email-send"

    @pytest.mark.asyncio
    async def test_send_email_publishes_correct_payload(
        self, email_client, mock_nc
    ):
        """Test that send_email sends the correct JSON payload."""
        await email_client.send_email(
            "user@example.com", "Test Subject", "Test Body"
        )

        call_args = mock_nc.publish.call_args
        payload = json.loads(call_args[0][1].decode())

        assert payload == {
            "to": "user@example.com",
            "subject": "Test Subject",
            "body": "Test Body",
        }

    @pytest.mark.asyncio
    async def test_send_email_fire_and_forget(self, email_client, mock_nc):
        """Test that send_email does not return any value."""
        result = await email_client.send_email(
            "user@example.com", "Subject", "Body"
        )

        assert result is None


class TestEmailClientSendBulkEmails:
    """Tests for the send_bulk_emails method."""

    @pytest.mark.asyncio
    async def test_send_bulk_emails_sends_all(self, email_client, mock_nc):
        """Test that send_bulk_emails sends all emails."""
        emails = [
            ("user1@example.com", "Subject 1", "Body 1"),
            ("user2@example.com", "Subject 2", "Body 2"),
            ("user3@example.com", "Subject 3", "Body 3"),
        ]

        await email_client.send_bulk_emails(emails)

        assert mock_nc.publish.call_count == 3

    @pytest.mark.asyncio
    async def test_send_bulk_emails_returns_count(self, email_client, mock_nc):
        """Test that send_bulk_emails returns the count of emails sent."""
        emails = [
            ("user1@example.com", "Subject 1", "Body 1"),
            ("user2@example.com", "Subject 2", "Body 2"),
        ]

        count = await email_client.send_bulk_emails(emails)

        assert count == 2

    @pytest.mark.asyncio
    async def test_send_bulk_emails_empty_list(self, email_client, mock_nc):
        """Test that send_bulk_emails handles empty list."""
        count = await email_client.send_bulk_emails([])

        assert count == 0
        mock_nc.publish.assert_not_called()

    @pytest.mark.asyncio
    async def test_send_bulk_emails_correct_payloads(
        self, email_client, mock_nc
    ):
        """Test that send_bulk_emails sends correct payloads for each email."""
        emails = [
            ("user1@example.com", "Subject 1", "Body 1"),
            ("user2@example.com", "Subject 2", "Body 2"),
        ]

        await email_client.send_bulk_emails(emails)

        calls = mock_nc.publish.call_args_list
        for i, call in enumerate(calls):
            payload = json.loads(call[0][1].decode())
            assert payload["to"] == emails[i][0]
            assert payload["subject"] == emails[i][1]
            assert payload["body"] == emails[i][2]
