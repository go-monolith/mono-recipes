#!/usr/bin/env python3
"""
WebSocket Chat Demo Client

Demonstrates the WebSocket chat server with multiple concurrent clients
simulating a real-time chat conversation.

Usage:
    python demo.py                    # Default: 3 users, 5 messages each
    python demo.py --users 5          # 5 users, 5 messages each
    python demo.py --messages 10      # 3 users, 10 messages each
    python demo.py --room general     # Use 'general' room instead of 'lobby'
"""

import argparse
import asyncio
import json
import random
import sys
import urllib.error
import urllib.request
from datetime import datetime
from typing import Union, List, Dict, Any
from urllib.parse import urlparse

try:
    import websockets
except ImportError:
    try:
        from colorama import Fore, Style
        print(f"{Fore.RED}Error: websockets library required. Install with: pip install websockets{Style.RESET_ALL}")
    except ImportError:
        print("Error: websockets library required. Install with: pip install websockets")
    sys.exit(1)

try:
    from colorama import Fore, Style, init
    init()
    # Exclude RED from user colors - reserved for errors only
    COLORS = [Fore.GREEN, Fore.YELLOW, Fore.BLUE, Fore.MAGENTA, Fore.CYAN]
except ImportError:
    print("Warning: colorama not installed. Output will not be colored.")
    COLORS = [""] * 5
    class Style:
        RESET_ALL = ""
    class Fore:
        WHITE = ""
        RED = ""
        GREEN = ""
        YELLOW = ""
        BLUE = ""
        MAGENTA = ""
        CYAN = ""

# Sample usernames
USERNAMES = [
    "Alice", "Bob", "Charlie", "Diana", "Eve", "Frank",
    "Grace", "Henry", "Ivy", "Jack", "Kate", "Leo"
]

# Sample chat messages for simulation
SAMPLE_MESSAGES = [
    "Hello everyone! 👋",
    "Hey there!",
    "How's it going?",
    "Great to be here!",
    "Anyone working on something interesting?",
    "Just joined, what did I miss?",
    "This chat app is pretty cool!",
    "Love the real-time updates!",
    "WebSockets are awesome 🚀",
    "The EventBus pattern is elegant",
    "Mono framework FTW!",
    "Has anyone tried the file upload demo?",
    "I'm learning Go, any tips?",
    "Check out the URL shortener recipe too",
    "Clean architecture is the way to go",
    "Microservices? More like modular monolith!",
    "NATS messaging is fast ⚡",
    "Fiber is a great web framework",
    "Anyone here from the Go community?",
    "Happy coding everyone! 💻",
]


class ChatClient:
    """WebSocket chat client for demo purposes."""

    def __init__(self, username: str, color: str, server_url: str):
        self.username = username
        self.color = color
        self.server_url = server_url
        self.ws = None
        self.room_id = None
        self.running = True

    def log(self, message: str):
        """Print a colored log message."""
        timestamp = datetime.now().strftime("%H:%M:%S")
        print(f"{self.color}[{timestamp}] {self.username}: {message}{Style.RESET_ALL}")

    def log_error(self, message: str):
        """Print a colored error message."""
        timestamp = datetime.now().strftime("%H:%M:%S")
        print(f"{Fore.RED}[{timestamp}] {self.username}: {message}{Style.RESET_ALL}")

    def _extract_usernames(self, payload: Union[List[str], Dict[str, Any]]) -> List[str]:
        """Extract usernames from users payload (handles multiple formats).

        Supports:
        - List format: ["Alice", "Bob", "Charlie"]
        - Dict format with strings: {"users": ["Alice", "Bob"]}
        - Dict format with objects: {"users": [{"username": "Alice"}, {"username": "Bob"}]}

        Args:
            payload: Server response containing user information

        Returns:
            List of username strings, empty list if format is unexpected
        """
        if isinstance(payload, list):
            # Validate all items are strings
            return [u for u in payload if isinstance(u, str)]

        if not isinstance(payload, dict):
            return []  # Handle unexpected types gracefully

        users = payload.get("users", [])
        if not isinstance(users, list):
            return []

        result = []
        for u in users:
            if isinstance(u, dict):
                username = u.get("username")
                if isinstance(username, str):
                    result.append(username)
            elif isinstance(u, str):
                result.append(u)
        return result

    async def connect(self):
        """Establish WebSocket connection."""
        try:
            self.ws = await websockets.connect(self.server_url)
            self.log("Connected to server")
            return True
        except Exception as e:
            self.log_error(f"Failed to connect: {e}")
            return False

    async def join_room(self, room_id: str):
        """Join a chat room."""
        self.room_id = room_id
        message = {
            "type": "join",
            "payload": {
                "room_id": room_id,
                "username": self.username
            }
        }
        await self.ws.send(json.dumps(message))
        self.log(f"Joining room '{room_id}'...")

    async def send_message(self, content: str):
        """Send a chat message."""
        message = {
            "type": "message",
            "payload": {
                "content": content
            }
        }
        await self.ws.send(json.dumps(message))

    async def leave_room(self):
        """Leave the current room."""
        message = {"type": "leave"}
        await self.ws.send(json.dumps(message))
        self.log("Left the room")

    async def get_history(self):
        """Request message history."""
        message = {"type": "history"}
        await self.ws.send(json.dumps(message))

    async def get_users(self):
        """Request list of users in room."""
        message = {"type": "users"}
        await self.ws.send(json.dumps(message))

    async def listen(self):
        """Listen for incoming messages."""
        try:
            async for raw_message in self.ws:
                if not self.running:
                    break
                try:
                    data = json.loads(raw_message)
                    msg_type = data.get("type", "")
                    payload = data.get("payload", {})

                    if msg_type == "joined":
                        if isinstance(payload, dict):
                            self.log(f"Successfully joined room '{payload.get('room_id')}'")
                    elif msg_type == "user_joined":
                        if isinstance(payload, dict) and payload.get("username") != self.username:
                            self.log(f"📥 {payload.get('username')} joined the room")
                    elif msg_type == "user_left":
                        if isinstance(payload, dict):
                            self.log(f"📤 {payload.get('username')} left the room")
                    elif msg_type == "chat_message":
                        if isinstance(payload, dict):
                            sender = payload.get("username", "Unknown")
                            content = payload.get("content", "")
                            if sender != self.username:
                                self.log(f"💬 {sender}: {content}")
                    elif msg_type == "history":
                        # Handle both formats: payload as list or payload as dict with "messages" key
                        if isinstance(payload, list):
                            messages = payload
                        elif isinstance(payload, dict):
                            messages = payload.get("messages", [])
                        else:
                            messages = []

                        # Validate messages is a list before using it
                        if isinstance(messages, list) and messages:
                            self.log(f"📜 History: {len(messages)} messages")
                    elif msg_type == "users":
                        usernames = self._extract_usernames(payload)
                        self.log(f"👥 Users in room: {', '.join(usernames)}")
                    elif msg_type == "error":
                        if isinstance(payload, dict):
                            self.log_error(f"❌ Error: {payload.get('message', 'Unknown error')}")

                except json.JSONDecodeError:
                    self.log_error(f"Invalid JSON received: {raw_message}")
        except websockets.exceptions.ConnectionClosed:
            self.log("Connection closed")
        except Exception as e:
            self.log_error(f"Listen error: {e}")

    async def close(self):
        """Close the WebSocket connection."""
        self.running = False
        if self.ws:
            await self.ws.close()


async def ensure_room_exists(server_url: str, room_name: str):
    """Create the room via REST API if it doesn't exist."""
    # Validate URL scheme
    parsed = urlparse(server_url)
    if parsed.scheme not in ('ws', 'wss', 'http', 'https'):
        print(f"{Fore.RED}⚠ Invalid server URL scheme: {parsed.scheme}{Style.RESET_ALL}")
        return

    if not parsed.netloc:
        print(f"{Fore.RED}⚠ Invalid server URL: missing host{Style.RESET_ALL}")
        return

    # Convert ws:// to http://
    http_scheme = 'https' if parsed.scheme == 'wss' else 'http'
    http_url = f"{http_scheme}://{parsed.netloc}{parsed.path.replace('/ws', '')}"
    api_url = f"{http_url}/api/v1/rooms"

    try:
        # Try to create the room
        data = json.dumps({"name": room_name}).encode("utf-8")
        req = urllib.request.Request(
            api_url,
            data=data,
            headers={"Content-Type": "application/json"},
            method="POST"
        )
        urllib.request.urlopen(req, timeout=2)
        print(f"{Fore.WHITE}✓ Created room '{room_name}'{Style.RESET_ALL}")
    except urllib.error.HTTPError as e:
        if e.code == 409:
            print(f"{Fore.WHITE}✓ Room '{room_name}' already exists{Style.RESET_ALL}")
        else:
            print(f"{Fore.RED}⚠ Could not create room: {e}{Style.RESET_ALL}")
    except Exception as e:
        print(f"{Fore.RED}⚠ Room creation skipped: {e}{Style.RESET_ALL}")


async def simulate_chat(client: ChatClient, room_id: str, num_messages: int):
    """Simulate a chat session with random messages."""
    # Connect
    if not await client.connect():
        return

    # Start listener task
    listener_task = asyncio.create_task(client.listen())

    # Join room
    await client.join_room(room_id)
    await asyncio.sleep(0.5)  # Wait for join confirmation

    # Request history and users
    await client.get_history()
    await asyncio.sleep(0.2)
    await client.get_users()
    await asyncio.sleep(0.3)

    # Send messages with random delays
    for i in range(num_messages):
        await asyncio.sleep(random.uniform(0.5, 2.0))
        message = random.choice(SAMPLE_MESSAGES)
        client.log(f"💬 Sending: {message}")
        await client.send_message(message)

    # Wait a bit to receive any final messages
    await asyncio.sleep(1.0)

    # Leave room
    await client.leave_room()
    await asyncio.sleep(0.3)

    # Close connection
    client.running = False
    listener_task.cancel()
    try:
        await listener_task
    except asyncio.CancelledError:
        pass
    await client.close()


async def run_demo(server_url: str, room_id: str, num_users: int, num_messages: int):
    """Run the chat demo with multiple concurrent users."""
    print(f"\n{'='*60}")
    print(f"  WebSocket Chat Demo")
    print(f"  Server: {server_url}")
    print(f"  Room: {room_id}")
    print(f"  Users: {num_users}")
    print(f"  Messages per user: {num_messages}")
    print(f"{'='*60}\n")

    # Ensure room exists
    await ensure_room_exists(server_url, room_id.capitalize())

    # Select random usernames
    selected_users = random.sample(USERNAMES, min(num_users, len(USERNAMES)))

    # Create clients
    clients = []
    for i, username in enumerate(selected_users):
        color = COLORS[i % len(COLORS)]
        client = ChatClient(username, color, server_url)
        clients.append(client)

    print(f"\n{Fore.WHITE}Starting chat simulation with {len(clients)} users...{Style.RESET_ALL}\n")

    # Run all clients concurrently
    tasks = [
        simulate_chat(client, room_id, num_messages)
        for client in clients
    ]

    await asyncio.gather(*tasks)

    print(f"\n{Fore.WHITE}{'='*60}{Style.RESET_ALL}")
    print(f"{Fore.WHITE}  Demo completed!{Style.RESET_ALL}")
    print(f"{Fore.WHITE}{'='*60}{Style.RESET_ALL}\n")


def main():
    parser = argparse.ArgumentParser(
        description="WebSocket Chat Demo - Simulates multiple chat users"
    )
    parser.add_argument(
        "--server",
        default="ws://localhost:8080/ws",
        help="WebSocket server URL (default: ws://localhost:8080/ws)"
    )
    parser.add_argument(
        "--room",
        default="lobby",
        help="Chat room to join (default: lobby)"
    )
    parser.add_argument(
        "--users",
        type=int,
        default=3,
        help="Number of simulated users (default: 3)"
    )
    parser.add_argument(
        "--messages",
        type=int,
        default=5,
        help="Number of messages per user (default: 5)"
    )

    args = parser.parse_args()

    if args.users < 1:
        print(f"{Fore.RED}Error: Must have at least 1 user{Style.RESET_ALL}")
        sys.exit(1)

    if args.users > len(USERNAMES):
        print(f"{Fore.RED}Error: Maximum {len(USERNAMES)} users supported{Style.RESET_ALL}")
        sys.exit(1)

    if args.messages < 1:
        print(f"{Fore.RED}Error: Must send at least 1 message{Style.RESET_ALL}")
        sys.exit(1)

    try:
        asyncio.run(run_demo(
            server_url=args.server,
            room_id=args.room,
            num_users=args.users,
            num_messages=args.messages
        ))
    except KeyboardInterrupt:
        print(f"\n{Fore.WHITE}Demo interrupted by user{Style.RESET_ALL}")
    except Exception as e:
        print(f"\n{Fore.RED}Error: {e}{Style.RESET_ALL}")
        sys.exit(1)


if __name__ == "__main__":
    main()
