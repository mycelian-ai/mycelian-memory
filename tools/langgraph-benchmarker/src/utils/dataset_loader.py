"""LongMemEval dataset loading and processing utilities."""

import json
import os
from pathlib import Path
from typing import List, Dict, Any, Optional
import structlog

from ..types import LongMemEvalQuestion

logger = structlog.get_logger(__name__)


class LongMemEvalDatasetLoader:
    """
    Loader for LongMemEval datasets with support for local LongMemEval repository.
    
    This loader can work with the local clone of the LongMemEval GitHub repository
    and load the different dataset variants.
    """

    def __init__(self, longmemeval_repo_path: str):
        """
        Initialize the dataset loader.
        
        Args:
            longmemeval_repo_path: Path to the local LongMemEval GitHub repository
        """
        self.repo_path = Path(longmemeval_repo_path)
        self.data_path = self.repo_path / "data"
        
        if not self.repo_path.exists():
            raise ValueError(f"LongMemEval repository path does not exist: {longmemeval_repo_path}")
        
        if not self.data_path.exists():
            raise ValueError(f"Data directory not found in repository: {self.data_path}")
        
        logger.info(
            "LongMemEval dataset loader initialized", 
            repo_path=str(self.repo_path),
            data_path=str(self.data_path)
        )

    def list_available_datasets(self) -> List[str]:
        """
        List all available dataset files in the data directory.
        
        Returns:
            List of dataset filenames
        """
        try:
            dataset_files = []
            for file_path in self.data_path.glob("*.json"):
                dataset_files.append(file_path.name)
            
            logger.debug("Available datasets", files=dataset_files)
            return sorted(dataset_files)
            
        except Exception as e:
            logger.error("Error listing dataset files", error=str(e))
            return []

    def load_dataset(self, dataset_variant: str = "longmemeval_s.json") -> List[Dict[str, Any]]:
        """
        Load a specific LongMemEval dataset variant.
        
        Args:
            dataset_variant: Dataset file name (e.g., "longmemeval_s.json", "longmemeval_m.json", "longmemeval_oracle.json")
            
        Returns:
            List of conversation data from the dataset
        """
        dataset_file = self.data_path / dataset_variant
        
        # For test files, also check the test directory in the benchmarker
        if not dataset_file.exists() and dataset_variant.startswith("longmemeval_test"):
            # Try looking in the benchmarker test directory
            benchmarker_test_dir = Path(__file__).parent.parent.parent / "test"
            test_dataset_file = benchmarker_test_dir / dataset_variant
            if test_dataset_file.exists():
                dataset_file = test_dataset_file
        
        if not dataset_file.exists():
            available = self.list_available_datasets()
            raise FileNotFoundError(
                f"Dataset file not found: {dataset_file}\n"
                f"Available datasets: {available}"
            )

        logger.info(
            "Loading dataset", 
            variant=dataset_variant,
            file_path=str(dataset_file)
        )

        try:
            with open(dataset_file, 'r', encoding='utf-8') as f:
                dataset = json.load(f)
            
            logger.info(
                "Dataset loaded successfully",
                variant=dataset_variant,
                total_questions=len(dataset)
            )
            
            return dataset
            
        except json.JSONDecodeError as e:
            logger.error(
                "Invalid JSON in dataset file",
                file=str(dataset_file),
                error=str(e)
            )
            raise
        except Exception as e:
            logger.error(
                "Error loading dataset",
                file=str(dataset_file), 
                error=str(e)
            )
            raise

    def validate_dataset(self, dataset: List[Dict[str, Any]]) -> bool:
        """
        Validate that the loaded dataset has the expected LongMemEval format.
        
        Args:
            dataset: Loaded dataset to validate
            
        Returns:
            True if dataset is valid, False otherwise
        """
        if not isinstance(dataset, list):
            logger.error("Dataset should be a list")
            return False

        if len(dataset) == 0:
            logger.error("Dataset is empty")
            return False

        # Check a sample entry for required fields
        sample = dataset[0]
        required_fields = [
            "question_id",
            "question_type", 
            "question",
            "answer",
            "question_date",
            "haystack_session_ids",
            "haystack_dates",
            "haystack_sessions",
            "answer_session_ids"
        ]

        missing_fields = []
        for field in required_fields:
            if field not in sample:
                missing_fields.append(field)

        if missing_fields:
            logger.error(
                "Dataset missing required fields",
                missing=missing_fields,
                sample_keys=list(sample.keys())
            )
            return False

        # Validate haystack_sessions structure
        if not isinstance(sample["haystack_sessions"], list):
            logger.error("haystack_sessions should be a list")
            return False

        if len(sample["haystack_sessions"]) > 0:
            session = sample["haystack_sessions"][0]
            if not isinstance(session, list):
                logger.error("Each session should be a list of turns")
                return False
            
            if len(session) > 0:
                turn = session[0]
                if not isinstance(turn, dict) or "role" not in turn or "content" not in turn:
                    logger.error("Each turn should be a dict with 'role' and 'content'")
                    return False

        logger.info("Dataset validation passed")
        return True

    def get_dataset_stats(self, dataset: List[Dict[str, Any]]) -> Dict[str, Any]:
        """
        Get statistics about the dataset.
        
        Args:
            dataset: Loaded dataset
            
        Returns:
            Dictionary with dataset statistics
        """
        if not dataset:
            return {"error": "Empty dataset"}

        stats = {
            "total_questions": len(dataset),
            "question_types": {},
            "abstention_questions": 0,
            "average_haystack_sessions": 0,
            "average_turns_per_session": 0,
            "total_haystack_turns": 0
        }

        total_sessions = 0
        total_turns = 0

        for item in dataset:
            # Count question types
            qtype = item.get("question_type", "unknown")
            stats["question_types"][qtype] = stats["question_types"].get(qtype, 0) + 1
            
            # Count abstention questions
            if item.get("question_id", "").endswith("_abs"):
                stats["abstention_questions"] += 1
            
            # Count sessions and turns
            haystack_sessions = item.get("haystack_sessions", [])
            total_sessions += len(haystack_sessions)
            
            for session in haystack_sessions:
                session_turns = len(session) if isinstance(session, list) else 0
                total_turns += session_turns

        # Calculate averages
        if len(dataset) > 0:
            stats["average_haystack_sessions"] = total_sessions / len(dataset)
        
        if total_sessions > 0:
            stats["average_turns_per_session"] = total_turns / total_sessions
        
        stats["total_haystack_turns"] = total_turns

        return stats

    def sample_dataset(
        self, 
        dataset: List[Dict[str, Any]], 
        sample_size: int, 
        seed: Optional[int] = None
    ) -> List[Dict[str, Any]]:
        """
        Create a random sample of the dataset for testing.
        
        Args:
            dataset: Full dataset
            sample_size: Number of questions to sample
            seed: Optional random seed for reproducibility
            
        Returns:
            Sampled dataset
        """
        import random
        
        if seed is not None:
            random.seed(seed)
        
        if sample_size >= len(dataset):
            logger.warning(
                "Sample size larger than dataset",
                sample_size=sample_size,
                dataset_size=len(dataset)
            )
            return dataset
        
        sampled = random.sample(dataset, sample_size)
        
        logger.info(
            "Dataset sampled",
            original_size=len(dataset),
            sample_size=len(sampled)
        )
        
        return sampled

    def filter_by_question_type(
        self, 
        dataset: List[Dict[str, Any]], 
        question_types: List[str]
    ) -> List[Dict[str, Any]]:
        """
        Filter dataset by specific question types.
        
        Args:
            dataset: Full dataset
            question_types: List of question types to include
            
        Returns:
            Filtered dataset
        """
        filtered = [
            item for item in dataset 
            if item.get("question_type") in question_types
        ]
        
        logger.info(
            "Dataset filtered by question type",
            original_size=len(dataset),
            filtered_size=len(filtered),
            types=question_types
        )
        
        return filtered


def load_longmemeval_dataset(
    repo_path: str,
    dataset_variant: str = "longmemeval_s.json",
    sample_size: Optional[int] = None,
    question_types: Optional[List[str]] = None,
    validate: bool = True
) -> List[Dict[str, Any]]:
    """
    Convenience function to load and optionally process a LongMemEval dataset.
    
    Args:
        repo_path: Path to LongMemEval repository
        dataset_variant: Dataset file name to load
        sample_size: Optional sample size for testing
        question_types: Optional list of question types to filter
        validate: Whether to validate the dataset format
        
    Returns:
        Processed dataset ready for benchmarking
    """
    # Initialize loader
    loader = LongMemEvalDatasetLoader(repo_path)
    
    # Load dataset
    dataset = loader.load_dataset(dataset_variant)
    
    # Validate if requested
    if validate and not loader.validate_dataset(dataset):
        raise ValueError("Dataset validation failed")
    
    # Filter by question types if specified
    if question_types:
        dataset = loader.filter_by_question_type(dataset, question_types)
    
    # Sample if requested
    if sample_size:
        dataset = loader.sample_dataset(dataset, sample_size)
    
    # Log final dataset info
    stats = loader.get_dataset_stats(dataset)
    logger.info("Final dataset ready", stats=stats)
    
    return dataset