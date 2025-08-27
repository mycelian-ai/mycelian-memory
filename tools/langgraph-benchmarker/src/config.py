"""Simple configuration utility for the LangGraph LongMemEval benchmarker."""

from typing import Optional, Dict, Any
from pathlib import Path
import structlog
import tomli

from .types import BenchmarkConfig

logger = structlog.get_logger(__name__)


def create_config(
    longmemeval_repo_path: str = "/Users/deesam/workspace/LongMemEval",
    dataset_variant: str = "longmemeval_s.json",
    mcp_url: str = "http://localhost:11546/mcp",
    mcp_transport: str = "streamable_http",
    memory_agent_model: str = "gpt-4o-mini-2024-07-18",
    qa_agent_model: str = "gpt-4o-2024-08-06", 
    evaluation_model: str = "anthropic.claude-3-haiku-20240307-v1:0",
    aws_region: str = "us-east-1",
    aws_access_key_id: Optional[str] = None,
    aws_secret_access_key: Optional[str] = None,
    vault_name: str = "longmemeval-benchmark",
    output_dir: str = "./results",
    max_concurrent_conversations: int = 1,
    enable_debug_logging: bool = False
) -> BenchmarkConfig:
    """
    Create benchmark configuration with simple parameter-based approach.
    
    Args:
        longmemeval_repo_path: Path to local LongMemEval GitHub repository
        dataset_variant: Dataset file name to use
        mcp_url: Mycelian MCP server URL
        mcp_transport: MCP transport type
        memory_agent_model: Model for Mycelian Memory Agent
        qa_agent_model: Model for QA Agent
        evaluation_model: Claude Haiku model ID for evaluation
        aws_region: AWS region for Bedrock
        aws_access_key_id: Optional AWS access key
        aws_secret_access_key: Optional AWS secret key
        vault_name: Name of benchmark vault
        output_dir: Directory for results
        max_concurrent_conversations: Concurrency limit
        enable_debug_logging: Enable debug logging
        
    Returns:
        Validated BenchmarkConfig
    """
    config = BenchmarkConfig(
        longmemeval_repo_path=longmemeval_repo_path,
        dataset_variant=dataset_variant,
        mcp_url=mcp_url,
        mcp_transport=mcp_transport,
        memory_agent_model=memory_agent_model,
        qa_agent_model=qa_agent_model,
        evaluation_model=evaluation_model,
        aws_region=aws_region,
        aws_access_key_id=aws_access_key_id,
        aws_secret_access_key=aws_secret_access_key,
        vault_name=vault_name,
        output_dir=output_dir,
        max_concurrent_conversations=max_concurrent_conversations,
        enable_debug_logging=enable_debug_logging
    )

    # Validate configuration
    validate_config(config)
    
    logger.info(
        "Configuration created",
        repo_path=config.longmemeval_repo_path,
        dataset_variant=config.dataset_variant,
        vault_name=config.vault_name
    )

    return config


def validate_config(config: BenchmarkConfig) -> None:
    """
    Validate the benchmark configuration.
    
    Args:
        config: Configuration to validate
        
    Raises:
        ValueError: If configuration is invalid
    """
    # Validate LongMemEval repository path
    repo_path = Path(config.longmemeval_repo_path)
    if not repo_path.exists():
        raise ValueError(f"LongMemEval repository path does not exist: {config.longmemeval_repo_path}")
    
    if not (repo_path / "data").exists():
        raise ValueError(f"Data directory not found in repository: {repo_path / 'data'}")

    # Validate dataset variant
    dataset_file = repo_path / "data" / config.dataset_variant
    if not dataset_file.exists():
        available_files = list((repo_path / "data").glob("*.json"))
        raise ValueError(
            f"Dataset variant not found: {config.dataset_variant}\n"
            f"Available files: {[f.name for f in available_files]}"
        )

    # Validate output directory can be created
    output_path = Path(config.output_dir)
    try:
        output_path.mkdir(parents=True, exist_ok=True)
    except Exception as e:
        raise ValueError(f"Cannot create output directory {config.output_dir}: {e}")

    # Validate AWS credentials if provided
    if config.aws_access_key_id and not config.aws_secret_access_key:
        raise ValueError("AWS_SECRET_ACCESS_KEY is required when AWS_ACCESS_KEY_ID is provided")
    
    if config.aws_secret_access_key and not config.aws_access_key_id:
        raise ValueError("AWS_ACCESS_KEY_ID is required when AWS_SECRET_ACCESS_KEY is provided")

    # Validate concurrency setting
    if config.max_concurrent_conversations < 1:
        raise ValueError("max_concurrent_conversations must be at least 1")

    logger.info("Configuration validation passed")


def setup_logging(config: BenchmarkConfig) -> None:
    """
    Set up structured logging based on configuration.
    
    Args:
        config: Benchmark configuration
    """
    import structlog

    level = "DEBUG" if config.enable_debug_logging else "INFO"
    
    structlog.configure(
        processors=[
            structlog.stdlib.filter_by_level,
            structlog.stdlib.add_logger_name,
            structlog.stdlib.add_log_level,
            structlog.stdlib.PositionalArgumentsFormatter(),
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.StackInfoRenderer(),
            structlog.processors.format_exc_info,
            structlog.processors.UnicodeDecoder(),
            structlog.processors.JSONRenderer()
        ],
        context_class=dict,
        logger_factory=structlog.stdlib.LoggerFactory(),
        wrapper_class=structlog.stdlib.BoundLogger,
        cache_logger_on_first_use=True,
    )

    # Set log level
    import logging
    logging.basicConfig(level=getattr(logging, level))
    
    logger.info(
        "Logging configured",
        level=level,
        debug_enabled=config.enable_debug_logging
    )


def get_default_config() -> BenchmarkConfig:
    """
    Get the default configuration for development/testing.
    
    Returns:
        Default BenchmarkConfig
    """
    return create_config()


def load_config_from_toml(config_file: str) -> BenchmarkConfig:
    """
    Load configuration from TOML file.
    
    Args:
        config_file: Path to TOML config file (.toml)
        
    Returns:
        Loaded BenchmarkConfig
    """
    config_path = Path(config_file)
    
    if not config_path.exists():
        raise FileNotFoundError(f"Config file not found: {config_file}")
    
    # Load TOML configuration
    with open(config_path, 'rb') as f:
        config_data = tomli.load(f)
    
    # Create config from loaded data
    config = BenchmarkConfig(**config_data)
    validate_config(config)
    
    logger.info(
        "Configuration loaded from TOML file",
        config_file=config_file,
        repo_path=config.longmemeval_repo_path,
        dataset_variant=config.dataset_variant
    )
    
    return config