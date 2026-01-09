#!/usr/bin/env python3
"""
Python NATS Client Demo

Demonstrates interoperability between Python clients and Go-based Mono applications:
1. RequestReplyService (math.calculate) - Synchronous operations
2. QueueGroupService (email.send) - Fire-and-forget messaging
3. StreamConsumerService (payment.process) - Durable processing

Usage:
    python demo.py                 # Run full demo
    python demo.py --math-only     # Only math demo
    python demo.py --email-only    # Only email demo
    python demo.py --payment-only  # Only payment demo
"""

import argparse
import asyncio
import json
import sys

import nats
from colorama import Fore, Style, init


async def demo_math_service(nc: nats.NATS) -> None:
    """Demonstrate RequestReplyService with math calculations."""
    print(f"\n{Fore.CYAN}{'='*50}")
    print("  RequestReplyService: Math Calculator")
    print(f"{'='*50}{Style.RESET_ALL}\n")

    operations = [
        ("add", 10, 5, "10 + 5"),
        ("subtract", 100, 42, "100 - 42"),
        ("multiply", 7, 8, "7 * 8"),
        ("divide", 100, 4, "100 / 4"),
        ("power", 2, 10, "2 ^ 10"),
        ("sqrt", 144, 0, "sqrt(144)"),
    ]

    print(f"  {Fore.YELLOW}Calling services.math.calculate:{Style.RESET_ALL}\n")

    for op, a, b, desc in operations:
        request = {"operation": op, "a": a, "b": b}
        response = await nc.request(
            "services.math.calculate",
            json.dumps(request).encode(),
            timeout=5.0,
        )
        result = json.loads(response.data)
        value = result.get("result", result.get("error", "unknown"))
        print(f"    {desc} = {Fore.GREEN}{value}{Style.RESET_ALL}")

    # Test error cases
    print(f"\n  {Fore.YELLOW}Testing error handling:{Style.RESET_ALL}\n")

    # Division by zero
    response = await nc.request(
        "services.math.calculate",
        json.dumps({"operation": "divide", "a": 10, "b": 0}).encode(),
        timeout=5.0,
    )
    result = json.loads(response.data)
    print(f"    10 / 0 = {Fore.RED}{result.get('error', 'unknown')}{Style.RESET_ALL}")

    # Square root of negative
    response = await nc.request(
        "services.math.calculate",
        json.dumps({"operation": "sqrt", "a": -16}).encode(),
        timeout=5.0,
    )
    result = json.loads(response.data)
    print(f"    sqrt(-16) = {Fore.RED}{result.get('error', 'unknown')}{Style.RESET_ALL}")


async def demo_email_service(nc: nats.NATS) -> None:
    """Demonstrate QueueGroupService with fire-and-forget emails."""
    print(f"\n{Fore.YELLOW}{'='*50}")
    print("  QueueGroupService: Email Notifications")
    print(f"{'='*50}{Style.RESET_ALL}\n")

    emails = [
        ("user1@example.com", "Welcome to Our Service!", "Thanks for signing up."),
        ("user2@example.com", "Your Order Has Shipped", "Track your package here."),
        ("user3@example.com", "Password Reset Request", "Click to reset your password."),
        ("user4@example.com", "Weekly Newsletter", "Here's what's new this week."),
        ("user5@example.com", "Account Verification", "Please verify your email address."),
    ]

    print(f"  {Fore.YELLOW}Publishing to services.notification.email-send:{Style.RESET_ALL}\n")
    print("    (Fire-and-forget - no response expected)\n")

    for to, subject, body in emails:
        request = {"to": to, "subject": subject, "body": body}
        await nc.publish(
            "services.notification.email-send",
            json.dumps(request).encode(),
        )
        print(f"    {Fore.GREEN}Queued:{Style.RESET_ALL} {to} - \"{subject}\"")

    print(f"\n    {Fore.CYAN}Check Go server logs to see processing by workers{Style.RESET_ALL}")

    # Allow time for processing
    await asyncio.sleep(1.5)


async def demo_payment_service(nc: nats.NATS) -> None:
    """Demonstrate StreamConsumerService with payment processing."""
    print(f"\n{Fore.MAGENTA}{'='*50}")
    print("  StreamConsumerService: Payment Processing")
    print(f"{'='*50}{Style.RESET_ALL}\n")

    js = nc.jetstream()

    payments = [
        ("pay-001", "user-123", "sub-monthly", 9.99),
        ("pay-002", "user-456", "sub-annual", 99.99),
        ("pay-003", "user-789", "sub-premium", 149.99),
    ]

    print(f"  {Fore.YELLOW}Publishing to JetStream (services.payment.payment-process):{Style.RESET_ALL}\n")

    for payment_id, user_id, subscription_id, amount in payments:
        request = {
            "payment_id": payment_id,
            "user_id": user_id,
            "subscription_id": subscription_id,
            "amount": amount,
        }
        try:
            await js.publish(
                "services.payment.payment-process",
                json.dumps(request).encode(),
            )
            print(
                f"    {Fore.GREEN}Submitted:{Style.RESET_ALL} "
                f"{payment_id} - ${amount:.2f} ({subscription_id})"
            )
        except Exception as e:
            print(f"    {Fore.RED}Error:{Style.RESET_ALL} {e}")

    # Wait for processing
    print(f"\n    {Fore.CYAN}Waiting for payment processing...{Style.RESET_ALL}")
    await asyncio.sleep(2)

    # Query statuses
    print(f"\n  {Fore.YELLOW}Querying payment statuses (services.payment.status):{Style.RESET_ALL}\n")

    status_indicators = {
        "completed": f"{Fore.GREEN}[OK]{Style.RESET_ALL}",
        "processing": f"{Fore.YELLOW}[..]{Style.RESET_ALL}",
        "failed": f"{Fore.RED}[X]{Style.RESET_ALL}",
    }
    default_indicator = f"{Fore.CYAN}[?]{Style.RESET_ALL}"

    for payment_id, _, _, _ in payments:
        try:
            response = await nc.request(
                "services.payment.status",
                json.dumps({"payment_id": payment_id}).encode(),
                timeout=5.0,
            )
            result = json.loads(response.data)
            status = result.get("status", "unknown")
            indicator = status_indicators.get(status, default_indicator)
            print(f"    {indicator} {payment_id}: {status}")
        except Exception as e:
            print(f"    {Fore.RED}Error querying {payment_id}:{Style.RESET_ALL} {e}")


async def main() -> int:
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description="Python NATS Client Demo for Mono Services",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python demo.py                 # Run all demos
  python demo.py --math-only     # Only math calculations
  python demo.py --email-only    # Only email queue
  python demo.py --payment-only  # Only payment stream
  python demo.py --nats-url nats://host:4222
        """,
    )
    parser.add_argument(
        "--math-only",
        action="store_true",
        help="Only run math service demo",
    )
    parser.add_argument(
        "--email-only",
        action="store_true",
        help="Only run email service demo",
    )
    parser.add_argument(
        "--payment-only",
        action="store_true",
        help="Only run payment service demo",
    )
    parser.add_argument(
        "--nats-url",
        default="nats://localhost:4222",
        help="NATS server URL (default: nats://localhost:4222)",
    )
    args = parser.parse_args()

    # Initialize colorama for cross-platform colored output
    init()

    print(f"\n{Fore.CYAN}Python NATS Client Demo{Style.RESET_ALL}")
    print("=" * 50)
    print(f"Connecting to: {args.nats_url}\n")

    try:
        nc = await nats.connect(args.nats_url)
    except Exception as e:
        print(f"{Fore.RED}Failed to connect to NATS: {e}{Style.RESET_ALL}")
        print("\nMake sure the Go server is running:")
        print("  cd projects/python-nats-client-demo")
        print("  go run .")
        return 1

    try:
        run_all = not (args.math_only or args.email_only or args.payment_only)

        if run_all or args.math_only:
            await demo_math_service(nc)

        if run_all or args.email_only:
            await demo_email_service(nc)

        if run_all or args.payment_only:
            await demo_payment_service(nc)

        print(f"\n{Fore.GREEN}{'='*50}")
        print("  Demo completed successfully!")
        print(f"{'='*50}{Style.RESET_ALL}\n")

        print("What was demonstrated:")
        print("  1. RequestReplyService - Synchronous math calculations")
        print("  2. QueueGroupService - Fire-and-forget email queue")
        print("  3. StreamConsumerService - Durable payment processing")
        print("")

        return 0

    finally:
        await nc.drain()


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
