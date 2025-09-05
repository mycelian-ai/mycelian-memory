#!/usr/bin/env python3
"""
Huey configuration for LongMemEval benchmarker orchestration.
Uses SqliteHuey for persistent task queue with proper retry logic.
"""

import os
from huey import SqliteHuey

# Get configuration from environment or use defaults
HUEY_DB_PATH = os.environ.get('HUEY_DB_PATH', 'huey_tasks.db')
HUEY_IMMEDIATE = os.environ.get('HUEY_IMMEDIATE', 'false').lower() == 'true'

# Create Huey instance with SqliteHuey backend
huey = SqliteHuey(
    filename=HUEY_DB_PATH,
    immediate=HUEY_IMMEDIATE,  # Set to True for testing (runs tasks synchronously)
)

# Configuration constants
DEFAULT_TASK_RETRIES = 3
DEFAULT_RETRY_DELAY = 60  # seconds
SESSION_PROCESSING_TIMEOUT = 300  # 5 minutes per session
QUESTION_PROCESSING_TIMEOUT = 7200  # 2 hours per question

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
            'filename': 'huey_orchestrator.log',
            'maxBytes': 10485760,  # 10MB
            'backupCount': 5,
            'formatter': 'standard'
        }
    },
    'loggers': {
        'huey': {
            'handlers': ['console', 'file'],
            'level': 'INFO',
        },
        'orchestrator': {
            'handlers': ['console', 'file'],
            'level': 'INFO',
        }
    }
}