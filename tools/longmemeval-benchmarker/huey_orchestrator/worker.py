#!/usr/bin/env python3
"""
Huey worker process for LongMemEval benchmarker.
Processes tasks from the queue.
"""

import sys
import logging
import signal
from huey_config import huey, LOGGING_CONFIG
from tasks import process_question, run_qa, check_run_health

# Configure logging
import logging.config
logging.config.dictConfig(LOGGING_CONFIG)

logger = logging.getLogger('orchestrator.worker')


def signal_handler(sig, frame):
    """Handle shutdown signals gracefully."""
    logger.info("Worker shutting down...")
    sys.exit(0)


def run_worker():
    """Run the Huey consumer to process tasks."""
    # Register signal handlers
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    logger.info("Starting Huey worker...")
    
    # Import huey consumer
    from huey.consumer import Consumer
    
    # Create and run consumer
    consumer = Consumer(huey, workers=1)
    consumer.run()


if __name__ == '__main__':
    run_worker()