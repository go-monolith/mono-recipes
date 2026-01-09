"""Tests for the PaymentClient class."""

import json
from unittest.mock import AsyncMock, MagicMock

import pytest

from client.payment_client import PaymentClient


@pytest.fixture
def mock_nc():
    """Create a mock NATS connection."""
    nc = MagicMock()
    nc.request = AsyncMock()

    # Mock JetStream context
    mock_js = MagicMock()
    mock_js.publish = AsyncMock()
    nc.jetstream.return_value = mock_js

    return nc


@pytest.fixture
def payment_client(mock_nc):
    """Create a PaymentClient with mocked connection."""
    return PaymentClient(mock_nc)


def make_response(data: dict) -> MagicMock:
    """Create a mock response with JSON data."""
    response = MagicMock()
    response.data = json.dumps(data).encode()
    return response


class TestPaymentClientSubmitPayment:
    """Tests for the submit_payment method."""

    @pytest.mark.asyncio
    async def test_submit_payment_publishes_to_jetstream(
        self, payment_client, mock_nc
    ):
        """Test that submit_payment publishes to JetStream."""
        await payment_client.submit_payment(
            "pay-001", "user-123", "sub-monthly", 9.99
        )

        mock_js = mock_nc.jetstream.return_value
        mock_js.publish.assert_called_once()

    @pytest.mark.asyncio
    async def test_submit_payment_correct_subject(
        self, payment_client, mock_nc
    ):
        """Test that submit_payment uses the correct subject."""
        await payment_client.submit_payment(
            "pay-001", "user-123", "sub-monthly", 9.99
        )

        mock_js = mock_nc.jetstream.return_value
        call_args = mock_js.publish.call_args
        assert call_args[0][0] == "services.payment.payment-process"

    @pytest.mark.asyncio
    async def test_submit_payment_correct_payload(
        self, payment_client, mock_nc
    ):
        """Test that submit_payment sends the correct JSON payload."""
        await payment_client.submit_payment(
            "pay-001", "user-123", "sub-monthly", 9.99
        )

        mock_js = mock_nc.jetstream.return_value
        call_args = mock_js.publish.call_args
        payload = json.loads(call_args[0][1].decode())

        assert payload == {
            "payment_id": "pay-001",
            "user_id": "user-123",
            "subscription_id": "sub-monthly",
            "amount": 9.99,
        }

    @pytest.mark.asyncio
    async def test_submit_payment_creates_jetstream_once(
        self, payment_client, mock_nc
    ):
        """Test that JetStream context is created only once."""
        await payment_client.submit_payment(
            "pay-001", "user-123", "sub-monthly", 9.99
        )
        await payment_client.submit_payment(
            "pay-002", "user-456", "sub-annual", 99.99
        )

        # jetstream() should be called only once (lazy initialization)
        assert mock_nc.jetstream.call_count == 1


class TestPaymentClientGetStatus:
    """Tests for the get_status method."""

    @pytest.mark.asyncio
    async def test_get_status_sends_correct_request(
        self, payment_client, mock_nc
    ):
        """Test that get_status sends the correct request."""
        mock_nc.request.return_value = make_response(
            {"payment_id": "pay-001", "status": "completed"}
        )

        await payment_client.get_status("pay-001")

        mock_nc.request.assert_called_once()
        call_args = mock_nc.request.call_args
        assert call_args[0][0] == "services.payment.status"

        payload = json.loads(call_args[0][1].decode())
        assert payload == {"payment_id": "pay-001"}

    @pytest.mark.asyncio
    async def test_get_status_returns_result(self, payment_client, mock_nc):
        """Test that get_status returns the parsed response."""
        expected = {
            "payment_id": "pay-001",
            "status": "completed",
            "message": "Payment processed successfully",
        }
        mock_nc.request.return_value = make_response(expected)

        result = await payment_client.get_status("pay-001")

        assert result == expected

    @pytest.mark.asyncio
    async def test_get_status_with_custom_timeout(
        self, payment_client, mock_nc
    ):
        """Test that get_status respects custom timeout."""
        mock_nc.request.return_value = make_response({"status": "pending"})

        await payment_client.get_status("pay-001", timeout=10.0)

        call_args = mock_nc.request.call_args
        assert call_args[1]["timeout"] == 10.0

    @pytest.mark.asyncio
    async def test_get_status_pending(self, payment_client, mock_nc):
        """Test get_status for pending payment."""
        mock_nc.request.return_value = make_response(
            {
                "payment_id": "unknown",
                "status": "pending",
                "message": "Payment not found or not yet processed",
            }
        )

        result = await payment_client.get_status("unknown")

        assert result["status"] == "pending"

    @pytest.mark.asyncio
    async def test_get_status_processing(self, payment_client, mock_nc):
        """Test get_status for processing payment."""
        mock_nc.request.return_value = make_response(
            {"payment_id": "pay-002", "status": "processing"}
        )

        result = await payment_client.get_status("pay-002")

        assert result["status"] == "processing"

    @pytest.mark.asyncio
    async def test_get_status_failed(self, payment_client, mock_nc):
        """Test get_status for failed payment."""
        mock_nc.request.return_value = make_response(
            {
                "payment_id": "pay-003",
                "status": "failed",
                "message": "Insufficient funds",
            }
        )

        result = await payment_client.get_status("pay-003")

        assert result["status"] == "failed"
        assert result["message"] == "Insufficient funds"
