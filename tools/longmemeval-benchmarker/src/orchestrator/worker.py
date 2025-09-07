#!/usr/bin/env python3
"""
Huey worker process for LongMemEval benchmarker.
Processes tasks from the queue.
"""

import sys
import logging
import signal
import argparse
import os

from src.orchestrator.orchestrator_config import huey, LOGGING_CONFIG

# Import tasks to ensure they are registered with Huey
from src.orchestrator.tasks import process_question, run_qa, check_run_health  # noqa: F401

# Configure logging
import logging.config

logging.config.dictConfig(LOGGING_CONFIG)

logger = logging.getLogger('orchestrator.worker')


def signal_handler(sig, frame):
    """Handle shutdown signals gracefully."""
    logger.info("Worker shutting down...")
    sys.exit(0)


def run_worker(workers: int = 1):
    """Run the Huey consumer to process tasks."""
    # Register signal handlers
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    logger.info("Starting Huey worker with %s consumer process(es)...", workers)

    from huey.consumer import Consumer

    consumer = Consumer(huey, workers=workers)
    consumer.run()


def main():
    """CLI entry point for worker launcher."""
    parser = argparse.ArgumentParser(description="Huey worker for LongMemEval orchestrator")
    parser.add_argument('--workers', '-w', type=int, default=int(os.environ.get('HUEY_WORKERS', '1')),
                        help='Number of consumer processes (default: 1 or $HUEY_WORKERS)')

    args = parser.parse_args()

    run_worker(workers=max(1, args.workers))


if __name__ == '__main__':
    main()
