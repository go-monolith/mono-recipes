"""Tests for the MathClient class."""

import json
from unittest.mock import AsyncMock, MagicMock

import pytest

from client.math_client import MathClient


@pytest.fixture
def mock_nc():
    """Create a mock NATS connection."""
    nc = MagicMock()
    nc.request = AsyncMock()
    return nc


@pytest.fixture
def math_client(mock_nc):
    """Create a MathClient with mocked connection."""
    return MathClient(mock_nc)


def make_response(data: dict) -> MagicMock:
    """Create a mock response with JSON data."""
    response = MagicMock()
    response.data = json.dumps(data).encode()
    return response


class TestMathClientCalculate:
    """Tests for the calculate method."""

    @pytest.mark.asyncio
    async def test_calculate_sends_correct_request(
        self, math_client, mock_nc
    ):
        """Test that calculate sends the correct JSON payload."""
        mock_nc.request.return_value = make_response({"result": 15})

        await math_client.calculate("add", 10, 5)

        mock_nc.request.assert_called_once()
        call_args = mock_nc.request.call_args
        assert call_args[0][0] == "services.math.calculate"

        payload = json.loads(call_args[0][1].decode())
        assert payload == {"operation": "add", "a": 10, "b": 5}

    @pytest.mark.asyncio
    async def test_calculate_returns_result(self, math_client, mock_nc):
        """Test that calculate returns the parsed response."""
        mock_nc.request.return_value = make_response(
            {"result": 25, "operation": "divide"}
        )

        result = await math_client.calculate("divide", 100, 4)

        assert result == {"result": 25, "operation": "divide"}

    @pytest.mark.asyncio
    async def test_calculate_with_custom_timeout(self, math_client, mock_nc):
        """Test that calculate respects custom timeout."""
        mock_nc.request.return_value = make_response({"result": 10})

        await math_client.calculate("add", 5, 5, timeout=10.0)

        mock_nc.request.assert_called_once()
        call_args = mock_nc.request.call_args
        assert call_args[1]["timeout"] == 10.0


class TestMathClientOperations:
    """Tests for convenience operation methods."""

    @pytest.mark.asyncio
    async def test_add(self, math_client, mock_nc):
        """Test add convenience method."""
        mock_nc.request.return_value = make_response({"result": 15})

        result = await math_client.add(10, 5)

        assert result == 15
        payload = json.loads(mock_nc.request.call_args[0][1].decode())
        assert payload["operation"] == "add"

    @pytest.mark.asyncio
    async def test_subtract(self, math_client, mock_nc):
        """Test subtract convenience method."""
        mock_nc.request.return_value = make_response({"result": 58})

        result = await math_client.subtract(100, 42)

        assert result == 58
        payload = json.loads(mock_nc.request.call_args[0][1].decode())
        assert payload["operation"] == "subtract"

    @pytest.mark.asyncio
    async def test_multiply(self, math_client, mock_nc):
        """Test multiply convenience method."""
        mock_nc.request.return_value = make_response({"result": 56})

        result = await math_client.multiply(7, 8)

        assert result == 56
        payload = json.loads(mock_nc.request.call_args[0][1].decode())
        assert payload["operation"] == "multiply"

    @pytest.mark.asyncio
    async def test_divide_success(self, math_client, mock_nc):
        """Test divide returns result on success."""
        mock_nc.request.return_value = make_response({"result": 25})

        result = await math_client.divide(100, 4)

        assert result == 25.0

    @pytest.mark.asyncio
    async def test_divide_by_zero_returns_none(self, math_client, mock_nc):
        """Test divide returns None for division by zero."""
        mock_nc.request.return_value = make_response(
            {"error": "division by zero"}
        )

        result = await math_client.divide(10, 0)

        assert result is None

    @pytest.mark.asyncio
    async def test_power(self, math_client, mock_nc):
        """Test power convenience method."""
        mock_nc.request.return_value = make_response({"result": 1024})

        result = await math_client.power(2, 10)

        assert result == 1024
        payload = json.loads(mock_nc.request.call_args[0][1].decode())
        assert payload["operation"] == "power"

    @pytest.mark.asyncio
    async def test_sqrt_success(self, math_client, mock_nc):
        """Test sqrt returns result on success."""
        mock_nc.request.return_value = make_response({"result": 12})

        result = await math_client.sqrt(144)

        assert result == 12.0

    @pytest.mark.asyncio
    async def test_sqrt_negative_returns_none(self, math_client, mock_nc):
        """Test sqrt returns None for negative numbers."""
        mock_nc.request.return_value = make_response(
            {"error": "cannot calculate square root of negative number"}
        )

        result = await math_client.sqrt(-16)

        assert result is None
