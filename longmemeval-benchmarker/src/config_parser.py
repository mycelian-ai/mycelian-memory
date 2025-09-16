"""Configuration parsing utilities extracted from the old benchmarker.

This module contains only the configuration parsing logic needed by the orchestrator.
"""

import time
from typing import Any, Dict


class _SimpleConfig:
    def __init__(self, raw_cfg: Dict[str, Any]):
        # Required fields
        self.dataset_file_path = raw_cfg["dataset_file_path"]
        self.vault_title = raw_cfg["vault_title"]

        # Models configuration - both ingest and qa are mandatory
        models_cfg = raw_cfg.get("models", {})
        if not models_cfg:
            # If no models section, use defaults
            models_cfg = {}

        # Ingest and QA models with defaults
        ingest_model = models_cfg.get("ingest", "openai:gpt-5-nano-2025-08-07")
        qa_model = models_cfg.get("qa", "openai:gpt-5-2025-08-07")

        self.models = type("Models", (), {
            "ingest": ingest_model,
            "qa": qa_model
        })()

        # Search configuration - optional with defaults
        search_cfg = raw_cfg.get("search", {})
        self.use_two_pass_search = search_cfg.get("use_two_pass", True)  # Default to True

        # Agent configuration - optional with defaults
        agent_cfg = raw_cfg.get("agent", {})
        self.context_only = agent_cfg.get("context_only", True)  # Default to True for faster processing

        # Auto-generated fields
        self.run_id = str(int(time.time()))

        # Fixed defaults (removed from config)
        self.provider = {"type": "model-provider"}  # Supports OpenAI and Vertex AI
        self.vault_id = None  # Always auto-generated from vault_title
        self.memory_title_template = "{question_id}__{run_id}"  # Standard format

        # Fixed internal params (not exposed in config)
        self.params = type(
            "Params",
            (),
            {
                # Fixed internal defaults
                "top_k": 10,  # Search results limit
                "max_tool_calls_per_turn": 5,  # Legacy, unused
                "dump_state": False,  # Debug feature, removed
                "use_two_pass_search": self.use_two_pass_search,  # From config or default True

                # Removed limits - control via dataset file instead
                "question_limit": None,  # Always process all questions
                "max_sessions_per_question": None,  # Always process all sessions
                "max_turns_per_session": None,  # Always process all turns
            },
        )()


def parse_config(raw_cfg: Dict[str, Any]) -> _SimpleConfig:
    return _SimpleConfig(raw_cfg)


def build_memory_title(template: str, question_id: str, run_id: str) -> str:
    return template.format(question_id=question_id, run_id=run_id)
