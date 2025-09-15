#!/usr/bin/env python3
"""
Huey configuration for LongMemEval benchmarker orchestration.
Uses SqliteHuey for persistent task queue with proper retry logic.

Strong isolation: the queue name is derived from the environment variable
HUEY_QUEUE_NAME. Orchestrator sets this per-run so workers only consume tasks
for the intended run.
"""

import os
from huey import SqliteHuey
from pathlib import Path

# Fixed configuration (no env- or CLI-driven toggles)
_BASE_DIR = Path(__file__).resolve().parents[2]  # Go up 2 levels: orchestrator -> src -> benchmarker root
_DATA_DIR = _BASE_DIR / 'data'
_DATA_DIR.mkdir(parents=True, exist_ok=True)

# Task queue (Huey) SQLite DB under data/
HUEY_DB_PATH = str(_DATA_DIR / 'orchestrator.db')
# Timeouts are configured in the benchmarker config file; provide sane defaults here.
INGEST_TIMEOUT_SEC = 7200  # 2 hours per question
QA_TIMEOUT_SEC = 900       # 15 minutes per QA

# Logs directory (fixed default)
LOGS_DIR = 'logs'

# Per-run queue isolation: pick up queue name from environment
HUEY_QUEUE_NAME = os.environ.get('HUEY_QUEUE_NAME', 'default')

# Per-run log file: pick up run_id from environment for log file path
HUEY_RUN_ID = os.environ.get('HUEY_RUN_ID', 'default')
ORCHESTRATOR_LOG_DIR = _BASE_DIR / 'logs' / HUEY_RUN_ID
ORCHESTRATOR_LOG_DIR.mkdir(parents=True, exist_ok=True)
ORCHESTRATOR_LOG_FILE = ORCHESTRATOR_LOG_DIR / 'orchestrator.log'

# Create Huey instance for asynchronous execution only
huey = SqliteHuey(filename=HUEY_DB_PATH, name=HUEY_QUEUE_NAME, immediate=False)

# Configuration constants
DEFAULT_TASK_RETRIES = 3
DEFAULT_RETRY_DELAY = 60  # seconds
SESSION_PROCESSING_TIMEOUT = 300  # 5 minutes per session
# Backward-compat constant for tasks module
QUESTION_PROCESSING_TIMEOUT = INGEST_TIMEOUT_SEC

# Logging configuration
LOGGING_CONFIG = {
    'version': 1,
    'disable_existing_loggers': False,
    'formatters': {
        'standard': {
            'format': '%(asctime)s [%(levelname)s] %(name)s: %(message)s'
        },
    },
    'handlers': {
        'console': {
            'class': 'logging.StreamHandler',
            'formatter': 'standard'
        },
        'file': {
            'class': 'logging.handlers.RotatingFileHandler',
            'filename': str(ORCHESTRATOR_LOG_FILE),
            'maxBytes': 10485760,  # 10MB
            'backupCount': 5,
            'formatter': 'standard'
        }
    },
    'loggers': {
        'huey': {
            'handlers': ['console', 'file'],
            'level': 'WARNING',
        },
        'orchestrator': {
            'handlers': ['console', 'file'],
            'level': 'INFO',
        }
    }
}
