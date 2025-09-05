#!/usr/bin/env python
"""
Worker script that processes tasks from Huey's queue.
Run multiple instances for parallel processing.
"""

import os
import sys
import signal
import time
from huey.consumer import Consumer
from tasks import huey

def signal_handler(sig, frame):
    """Handle graceful shutdown."""
    print(f"\n[Worker {os.getpid()}] Shutting down gracefully...")
    sys.exit(0)

def main():
    # Register signal handler for graceful shutdown
    signal.signal(signal.SIGINT, signal_handler)
    
    # Worker configuration
    worker_config = {
        'workers': 1,  # Number of worker threads per process
        'periodic': True,  # Enable periodic tasks
        'initial_delay': 0.1,  # Small delay before starting
        'backoff': 1.15,  # Exponential backoff multiplier
        'max_delay': 10.0,  # Maximum delay between task checks
        'scheduler_interval': 1,  # Check for scheduled tasks every second
        'check_worker_health': True,
        'health_check_interval': 10,
    }
    
    print(f"[Worker {os.getpid()}] Starting Huey consumer...")
    print(f"Configuration: {worker_config}")
    
    # Create and run consumer
    consumer = Consumer(huey, **worker_config)
    
    print(f"[Worker {os.getpid()}] Ready to process tasks!")
    print("Press Ctrl+C to stop\n")
    
    try:
        consumer.run()
    except KeyboardInterrupt:
        print(f"\n[Worker {os.getpid()}] Received interrupt, shutting down...")
        consumer.stop()

if __name__ == "__main__":
    main()