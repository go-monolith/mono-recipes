#!/usr/bin/env python3
"""
Background Jobs Demo Script

Demonstrates:
- Enqueueing multiple jobs of different types
- Watching job progress in real-time
- Simulating job failures and retries
- Viewing dead-letter queue jobs
"""

import requests
import time
import sys
from typing import Dict, List, Any
from datetime import datetime

# API Configuration
API_BASE_URL = "http://localhost:8080/api/v1"
POLL_INTERVAL = 0.5  # seconds


class JobClient:
    """Client for interacting with the background jobs API."""

    def __init__(self, base_url: str):
        self.base_url = base_url

    def create_job(self, job_type: str, payload: Dict[str, Any], priority: int = 0) -> Dict[str, Any]:
        """Create a new job."""
        response = requests.post(
            f"{self.base_url}/jobs",
            json={
                "type": job_type,
                "payload": payload,
                "priority": priority
            }
        )
        response.raise_for_status()
        return response.json()

    def get_job(self, job_id: str) -> Dict[str, Any]:
        """Get job status by ID."""
        response = requests.get(f"{self.base_url}/jobs/{job_id}")
        response.raise_for_status()
        return response.json()

    def list_jobs(self, status: str = None, job_type: str = None, limit: int = 50) -> List[Dict[str, Any]]:
        """List jobs with optional filtering."""
        params = {"limit": limit}
        if status:
            params["status"] = status
        if job_type:
            params["type"] = job_type

        response = requests.get(f"{self.base_url}/jobs", params=params)
        response.raise_for_status()
        return response.json()["jobs"]


def print_header(text: str):
    """Print a formatted header."""
    print(f"\n{'=' * 80}")
    print(f"  {text}")
    print(f"{'=' * 80}\n")


def print_job_status(job: Dict[str, Any]):
    """Print formatted job status."""
    status_emoji = {
        "pending": "⏳",
        "processing": "🔄",
        "completed": "✅",
        "failed": "❌",
        "dead_letter": "💀"
    }

    emoji = status_emoji.get(job["status"], "❓")
    print(f"{emoji} Job {job['id'][:8]}... | Type: {job['type']:20s} | Status: {job['status']:12s}", end="")

    if job["status"] == "processing":
        print(f" | Progress: {job['progress']:3d}% | {job.get('progress_message', '')}", end="")

    if job["status"] == "completed":
        duration = "N/A"
        if job.get("started_at") and job.get("completed_at"):
            start = datetime.fromisoformat(job["started_at"].replace("Z", "+00:00"))
            end = datetime.fromisoformat(job["completed_at"].replace("Z", "+00:00"))
            duration = f"{(end - start).total_seconds():.2f}s"
        print(f" | Duration: {duration}", end="")

    if job["status"] == "failed" or job["status"] == "dead_letter":
        print(f" | Error: {job.get('error', 'Unknown')}", end="")
        print(f" | Retries: {job['retry_count']}/{job['max_retries']}", end="")

    print()


def watch_jobs(client: JobClient, job_ids: List[str], max_wait: int = 60):
    """Watch jobs until they complete or timeout."""
    print_header("Watching Job Progress")

    start_time = time.time()
    completed_jobs = set()

    while len(completed_jobs) < len(job_ids):
        if time.time() - start_time > max_wait:
            print(f"\n⏱️  Timeout reached ({max_wait}s), stopping watch...")
            break

        # Clear screen (works on Unix-like systems)
        print("\033[H\033[J", end="")
        print(f"Time elapsed: {time.time() - start_time:.1f}s | Watching {len(job_ids)} jobs\n")

        for job_id in job_ids:
            if job_id in completed_jobs:
                continue

            try:
                job = client.get_job(job_id)
                print_job_status(job)

                if job["status"] in ["completed", "failed", "dead_letter"]:
                    completed_jobs.add(job_id)
            except requests.RequestException as e:
                print(f"❌ Error fetching job {job_id[:8]}...: {e}")

        if len(completed_jobs) < len(job_ids):
            time.sleep(POLL_INTERVAL)

    print(f"\n✅ {len(completed_jobs)}/{len(job_ids)} jobs finished")


def demo_email_jobs(client: JobClient, count: int = 3) -> List[str]:
    """Create email sending jobs."""
    print_header("Creating Email Jobs")

    job_ids = []
    for i in range(count):
        job = client.create_job(
            job_type="email",
            payload={
                "to": f"user{i+1}@example.com",
                "subject": f"Test Email #{i+1}",
                "body": "This is a test email from the background jobs demo"
            },
            priority=1
        )
        job_ids.append(job["id"])
        print(f"📧 Created email job: {job['id']}")

    return job_ids


def demo_image_processing_jobs(client: JobClient, count: int = 2) -> List[str]:
    """Create image processing jobs."""
    print_header("Creating Image Processing Jobs")

    job_ids = []
    for i in range(count):
        job = client.create_job(
            job_type="image_processing",
            payload={
                "image_url": f"https://example.com/images/photo{i+1}.jpg",
                "operations": ["resize", "watermark", "optimize"],
                "output_path": f"/output/processed_photo{i+1}.jpg"
            },
            priority=2
        )
        job_ids.append(job["id"])
        print(f"🖼️  Created image processing job: {job['id']}")

    return job_ids


def demo_report_generation_jobs(client: JobClient, count: int = 2) -> List[str]:
    """Create report generation jobs."""
    print_header("Creating Report Generation Jobs")

    job_ids = []
    reports = ["daily_sales", "monthly_revenue", "user_activity", "inventory_status"]

    for i in range(count):
        job = client.create_job(
            job_type="report_generation",
            payload={
                "report_type": reports[i % len(reports)],
                "format": "pdf",
                "date_range": {
                    "start": "2024-01-01",
                    "end": "2024-01-31"
                }
            },
            priority=0
        )
        job_ids.append(job["id"])
        print(f"📊 Created report generation job: {job['id']}")

    return job_ids


def show_summary(client: JobClient):
    """Show summary of all jobs."""
    print_header("Job Summary")

    try:
        all_jobs = client.list_jobs(limit=100)

        status_counts = {}
        type_counts = {}

        for job in all_jobs:
            status = job["status"]
            job_type = job["type"]

            status_counts[status] = status_counts.get(status, 0) + 1
            type_counts[job_type] = type_counts.get(job_type, 0) + 1

        print("📈 Status Distribution:")
        for status, count in sorted(status_counts.items()):
            print(f"   {status:15s}: {count:3d}")

        print("\n📋 Type Distribution:")
        for job_type, count in sorted(type_counts.items()):
            print(f"   {job_type:25s}: {count:3d}")

        # Show dead-letter queue jobs if any
        dlq_jobs = [j for j in all_jobs if j["status"] == "dead_letter"]
        if dlq_jobs:
            print_header("Dead-Letter Queue Jobs")
            for job in dlq_jobs:
                print_job_status(job)

    except requests.RequestException as e:
        print(f"❌ Error fetching job summary: {e}")


def main():
    """Main demo function."""
    print_header("Background Jobs Demo - Starting")

    # Check if API is available
    client = JobClient(API_BASE_URL)

    try:
        requests.get(f"http://localhost:8080/health", timeout=2)
    except requests.RequestException:
        print("❌ Error: Cannot connect to API at http://localhost:8080")
        print("   Please ensure the application is running with: go run main.go")
        sys.exit(1)

    print("✅ Connected to API\n")

    # Create jobs
    job_ids = []

    # Create email jobs
    job_ids.extend(demo_email_jobs(client, count=3))
    time.sleep(0.5)

    # Create image processing jobs
    job_ids.extend(demo_image_processing_jobs(client, count=2))
    time.sleep(0.5)

    # Create report generation jobs
    job_ids.extend(demo_report_generation_jobs(client, count=2))

    print(f"\n📝 Created {len(job_ids)} jobs total")
    time.sleep(2)

    # Watch jobs progress
    watch_jobs(client, job_ids, max_wait=120)

    # Show final summary
    show_summary(client)

    print_header("Demo Complete")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n⚠️  Demo interrupted by user")
        sys.exit(0)
    except Exception as e:
        print(f"\n\n❌ Unexpected error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
