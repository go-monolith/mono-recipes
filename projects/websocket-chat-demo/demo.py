#!/usr/bin/env python3
"""
WebSocket Chat Demo Script

Demonstrates the WebSocket chat functionality with multiple concurrent clients.
Simulates a chat conversation between multiple users using asyncio and websockets.

Usage:
    python demo.py                    # Run with default settings (3 users, 5 messages each)
    python demo.py --users 5          # Run with 5 users
    python demo.py --messages 10      # Each user sends 10 messages
    python demo.py --room "general"   # Use custom room name
"""

import argparse
import asyncio
import json
import random
import sys
from datetime import datetime

try:
    import websockets
except ImportError:
    print("Error: websockets library required. Install with: pip install websockets")
    sys.exit(1)

try:
    from colorama import Fore, Style, init
    init()
    COLORS_AVAILABLE = True
except ImportError:
    COLORS_AVAILABLE = False
    # Fallback: no colors
    class Fore:
        RED = GREEN = YELLOW = BLUE = MAGENTA = CYAN = WHITE = RESET = ""
    class Style:
        BRIGHT = RESET_ALL = ""

# User colors for distinguishing messages
USER_COLORS = [Fore.GREEN, Fore.YELLOW, Fore.BLUE, Fore.MAGENTA, Fore.CYAN]

# Sample messages for the chat simulation
SAMPLE_MESSAGES = [
    "Hello everyone! 👋",
    "How's it going?",
    "Just joined the chat!",
    "Anyone here?",
    "Great to be here!",
    "What are we discussing today?",
    "Nice to meet you all!",
    "This chat is working great!",
    "WebSocket magic! ✨",
    "Real-time messaging is awesome!",
    "Testing the event-driven architecture",
    "Messages via EventBus pubsub",
    "Loving the Fiber + NATS combo",
    "Mono framework rocks! 🚀",
    "Chat demo working perfectly",
]


def get_color(index: int) -> str:
    """Get a color for a user based on their index."""
    return USER_COLORS[index % len(USER_COLORS)]


def log(message: str, prefix: str = "DEMO", color: str = Fore.WHITE):
    """Print a log message with timestamp."""
    timestamp = datetime.now().strftime("%H:%M:%S")
    print(f"{color}[{timestamp}] [{prefix}] {message}{Style.RESET_ALL}")


async def create_room(base_url: str, room_name: str) -> str:
    """Create a chat room via REST API."""
    import aiohttp

    async with aiohttp.ClientSession() as session:
        async with session.post(
            f"{base_url}/api/v1/rooms",
            json={"name": room_name}
        ) as response:
            if response.status == 201:
                data = await response.json()
                return data["id"]
            elif response.status == 200:
                # Room might already exist, list and find it
                async with session.get(f"{base_url}/api/v1/rooms") as list_resp:
                    if list_resp.status == 200:
                        rooms = await list_resp.json()
                        for room in rooms.get("rooms", []):
                            if room["name"] == room_name:
                                return room["id"]
            raise Exception(f"Failed to create room: {response.status}")


async def chat_client(
    ws_url: str,
    username: str,
    room_id: str,
    num_messages: int,
    user_index: int,
    message_delay: float
):
    """Simulate a chat client."""
    color = get_color(user_index)

    try:
        async with websockets.connect(f"{ws_url}?username={username}") as ws:
            log(f"{username} connected to WebSocket", "CLIENT", color)

            # Wait for welcome message
            welcome = await ws.recv()
            welcome_data = json.loads(welcome)
            if welcome_data.get("type") == "connected":
                log(f"{username} received welcome (id: {welcome_data.get('user_id', 'N/A')[:8]}...)", "CLIENT", color)

            # Join the room
            await ws.send(json.dumps({
                "type": "join",
                "room_id": room_id
            }))

            # Wait for join confirmation
            join_response = await ws.recv()
            join_data = json.loads(join_response)
            if join_data.get("type") == "joined":
                log(f"{username} joined room", "CLIENT", color)

            # Small delay to let other users join
            await asyncio.sleep(random.uniform(0.5, 1.5))

            # Send messages
            for i in range(num_messages):
                message = random.choice(SAMPLE_MESSAGES)

                await ws.send(json.dumps({
                    "type": "message",
                    "content": message
                }))

                log(f"{username}: {message}", "CHAT", color)

                # Wait for confirmation and broadcast
                try:
                    # Read messages until we get our confirmation or timeout
                    async def read_with_timeout():
                        while True:
                            try:
                                msg = await asyncio.wait_for(ws.recv(), timeout=2.0)
                                data = json.loads(msg)

                                # Handle different message types
                                if data.get("type") == "message" and data.get("message_id"):
                                    # This is our confirmation
                                    return
                                elif data.get("type") == "message" and data.get("username"):
                                    # This is a broadcast from another user
                                    if data.get("username") != username:
                                        other_color = Fore.WHITE
                                        log(f"  ← {data['username']}: {data.get('content', '')}", "RECV", other_color)
                                elif data.get("type") == "user_joined":
                                    log(f"  ← {data.get('username', 'someone')} joined", "EVENT", Fore.CYAN)
                                elif data.get("type") == "user_left":
                                    log(f"  ← {data.get('username', 'someone')} left", "EVENT", Fore.CYAN)
                            except asyncio.TimeoutError:
                                return

                    await read_with_timeout()
                except Exception:
                    pass

                # Random delay between messages
                await asyncio.sleep(message_delay + random.uniform(0, 0.5))

            # Leave room
            await ws.send(json.dumps({"type": "leave"}))
            log(f"{username} left the room", "CLIENT", color)

    except Exception as e:
        log(f"{username} error: {e}", "ERROR", Fore.RED)


async def message_listener(ws_url: str, room_id: str, duration: float):
    """Listen for all messages in a room (observer)."""
    try:
        async with websockets.connect(f"{ws_url}?username=observer") as ws:
            # Wait for welcome
            await ws.recv()

            # Join room
            await ws.send(json.dumps({
                "type": "join",
                "room_id": room_id
            }))
            await ws.recv()  # join confirmation

            log("Observer connected and listening...", "OBSERVER", Fore.WHITE)

            end_time = asyncio.get_event_loop().time() + duration
            while asyncio.get_event_loop().time() < end_time:
                try:
                    msg = await asyncio.wait_for(ws.recv(), timeout=1.0)
                    data = json.loads(msg)

                    if data.get("type") == "message" and data.get("username"):
                        log(f"[{data['username']}] {data.get('content', '')}", "BROADCAST", Fore.WHITE)
                    elif data.get("type") == "user_joined":
                        log(f"+ {data.get('username', 'someone')} joined the room", "BROADCAST", Fore.GREEN)
                    elif data.get("type") == "user_left":
                        log(f"- {data.get('username', 'someone')} left the room", "BROADCAST", Fore.YELLOW)
                except asyncio.TimeoutError:
                    continue

    except Exception as e:
        log(f"Observer error: {e}", "ERROR", Fore.RED)


async def run_demo(
    host: str,
    port: int,
    num_users: int,
    num_messages: int,
    room_name: str,
    message_delay: float
):
    """Run the chat demo."""
    base_url = f"http://{host}:{port}"
    ws_url = f"ws://{host}:{port}/ws"

    print()
    print(f"{Fore.CYAN}{'='*60}{Style.RESET_ALL}")
    print(f"{Fore.CYAN}  WebSocket Chat Demo{Style.RESET_ALL}")
    print(f"{Fore.CYAN}{'='*60}{Style.RESET_ALL}")
    print()
    print(f"  Server:      {base_url}")
    print(f"  WebSocket:   {ws_url}")
    print(f"  Users:       {num_users}")
    print(f"  Messages:    {num_messages} per user")
    print(f"  Room:        {room_name}")
    print()
    print(f"{Fore.CYAN}{'='*60}{Style.RESET_ALL}")
    print()

    # Check server health
    log("Checking server health...", "DEMO")
    try:
        import aiohttp
        async with aiohttp.ClientSession() as session:
            async with session.get(f"{base_url}/health", timeout=aiohttp.ClientTimeout(total=5)) as response:
                if response.status == 200:
                    log("Server is healthy!", "DEMO", Fore.GREEN)
                else:
                    log(f"Server returned status {response.status}", "DEMO", Fore.YELLOW)
    except Exception as e:
        log(f"Cannot connect to server: {e}", "ERROR", Fore.RED)
        log("Make sure the server is running: go run main.go", "DEMO", Fore.YELLOW)
        return

    # Create room
    log(f"Creating room '{room_name}'...", "DEMO")
    try:
        room_id = await create_room(base_url, room_name)
        log(f"Room created with ID: {room_id[:8]}...", "DEMO", Fore.GREEN)
    except Exception as e:
        log(f"Failed to create room: {e}", "ERROR", Fore.RED)
        return

    print()
    log("Starting chat simulation...", "DEMO", Fore.CYAN)
    print()

    # Create user names
    usernames = [f"User{i+1}" for i in range(num_users)]

    # Calculate expected duration
    expected_duration = (num_messages * (message_delay + 0.5)) + 5

    # Create tasks for all users
    tasks = []
    for i, username in enumerate(usernames):
        task = asyncio.create_task(
            chat_client(
                ws_url=ws_url,
                username=username,
                room_id=room_id,
                num_messages=num_messages,
                user_index=i,
                message_delay=message_delay
            )
        )
        tasks.append(task)
        # Stagger user connections
        await asyncio.sleep(0.3)

    # Wait for all users to finish
    await asyncio.gather(*tasks, return_exceptions=True)

    print()
    log("Chat simulation completed!", "DEMO", Fore.GREEN)

    # Show final stats
    print()
    print(f"{Fore.CYAN}{'='*60}{Style.RESET_ALL}")
    print(f"  Summary:")
    print(f"    - Users: {num_users}")
    print(f"    - Messages sent: {num_users * num_messages}")
    print(f"    - Room: {room_name}")
    print(f"{Fore.CYAN}{'='*60}{Style.RESET_ALL}")
    print()

    # Check message history
    log("Fetching message history...", "DEMO")
    try:
        import aiohttp
        async with aiohttp.ClientSession() as session:
            async with session.get(f"{base_url}/api/v1/rooms/{room_id}/history?limit=100") as response:
                if response.status == 200:
                    data = await response.json()
                    messages = data.get("messages", [])
                    log(f"Total messages in history: {len(messages)}", "DEMO", Fore.GREEN)
    except Exception as e:
        log(f"Failed to fetch history: {e}", "ERROR", Fore.RED)


def main():
    parser = argparse.ArgumentParser(
        description="WebSocket Chat Demo - Simulate multi-user chat",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
    python demo.py                         # Default: 3 users, 5 messages each
    python demo.py --users 5 --messages 10 # 5 users, 10 messages each
    python demo.py --room "developers"     # Use custom room name
    python demo.py --host 192.168.1.100    # Connect to remote server
        """
    )

    parser.add_argument(
        "--host",
        default="localhost",
        help="Server host (default: localhost)"
    )
    parser.add_argument(
        "--port",
        type=int,
        default=3000,
        help="Server port (default: 3000)"
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
        help="Messages per user (default: 5)"
    )
    parser.add_argument(
        "--room",
        default="demo-room",
        help="Room name (default: demo-room)"
    )
    parser.add_argument(
        "--delay",
        type=float,
        default=1.0,
        help="Delay between messages in seconds (default: 1.0)"
    )

    args = parser.parse_args()

    # Validate arguments
    if args.users < 1:
        print("Error: --users must be at least 1")
        sys.exit(1)
    if args.users > 20:
        print("Warning: Using more than 20 users may cause performance issues")
    if args.messages < 1:
        print("Error: --messages must be at least 1")
        sys.exit(1)

    # Check for required dependencies
    try:
        import aiohttp
    except ImportError:
        print("Error: aiohttp library required. Install with: pip install aiohttp")
        sys.exit(1)

    # Run the demo
    try:
        asyncio.run(run_demo(
            host=args.host,
            port=args.port,
            num_users=args.users,
            num_messages=args.messages,
            room_name=args.room,
            message_delay=args.delay
        ))
    except KeyboardInterrupt:
        print("\n\nDemo interrupted by user.")
    except Exception as e:
        print(f"\nError: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
