"""Main entry point for the LangGraph LongMemEval benchmarker."""

import asyncio
import json
import time
from pathlib import Path
from typing import Dict, Any, Optional
import structlog

from .config import create_config, setup_logging, load_config_from_toml
from .utils.dataset_loader import load_longmemeval_dataset
from .workflows.end_to_end import create_longmemeval_workflow

logger = structlog.get_logger(__name__)


async def run_benchmark(
    config_file: str = "config.toml",
    sample_size: Optional[int] = None
) -> Dict[str, Any]:
    """
    Run the complete LongMemEval benchmark.
    
    Args:
        config_file: Path to TOML configuration file
        sample_size: Optional sample size for testing
        
    Returns:
        Benchmark results dictionary
    """
    start_time = time.time()
    
    # Load configuration from TOML file
    config = load_config_from_toml(config_file)
    
    # Setup logging
    setup_logging(config)
    
    logger.info(
        "Starting LongMemEval benchmark",
        config=config.model_dump(),
        sample_size=sample_size
    )
    
    try:
        # Load dataset
        dataset = load_longmemeval_dataset(
            repo_path=config.longmemeval_repo_path,
            dataset_variant=config.dataset_variant,
            sample_size=sample_size,
            validate=True
        )
        
        logger.info(
            "Dataset loaded",
            total_questions=len(dataset),
            variant=config.dataset_variant
        )
        
        # Create and initialize workflow
        workflow = await create_longmemeval_workflow(config)
        
        try:
            # Run the benchmark
            results = await workflow.run_full_benchmark(dataset)
            
            # Add timing information
            end_time = time.time()
            results["total_runtime_seconds"] = end_time - start_time
            results["average_time_per_question"] = results["total_runtime_seconds"] / len(dataset)
            
            # Save results
            await save_results(results, config.output_dir, config.dataset_variant)
            
            logger.info(
                "Benchmark completed successfully",
                total_questions=results["total_questions"],
                correct_answers=results["correct_answers"], 
                overall_accuracy=results["overall_accuracy"],
                runtime_seconds=results["total_runtime_seconds"]
            )
            
            return results
            
        finally:
            # Clean up workflow
            await workflow.close()
            
    except Exception as e:
        logger.error("Benchmark failed", error=str(e))
        raise


async def save_results(
    results: Dict[str, Any], 
    output_dir: str, 
    dataset_variant: str
) -> None:
    """
    Save benchmark results to output directory.
    
    Args:
        results: Benchmark results
        output_dir: Output directory
        dataset_variant: Dataset variant name for filename
    """
    # Create output directory
    output_path = Path(output_dir)
    output_path.mkdir(parents=True, exist_ok=True)
    
    # Generate filename with timestamp
    timestamp = time.strftime("%Y%m%d_%H%M%S")
    variant_name = dataset_variant.replace(".json", "")
    filename = f"longmemeval_results_{variant_name}_{timestamp}.json"
    
    results_file = output_path / filename
    
    # Save results
    with open(results_file, 'w', encoding='utf-8') as f:
        json.dump(results, f, indent=2, default=str)
    
    # Also save a summary file
    summary = {
        "dataset_variant": dataset_variant,
        "timestamp": timestamp,
        "total_questions": results["total_questions"],
        "correct_answers": results["correct_answers"],
        "overall_accuracy": results["overall_accuracy"],
        "accuracy_by_type": results["accuracy_by_type"],
        "runtime_seconds": results["total_runtime_seconds"],
        "avg_time_per_question": results["average_time_per_question"]
    }
    
    summary_file = output_path / f"summary_{variant_name}_{timestamp}.json"
    with open(summary_file, 'w', encoding='utf-8') as f:
        json.dump(summary, f, indent=2)
    
    logger.info(
        "Results saved",
        results_file=str(results_file),
        summary_file=str(summary_file)
    )


async def quick_test(
    config_file: str = "config.toml",
    sample_size: int = 5
) -> Dict[str, Any]:
    """
    Run a quick test with a small sample.
    
    Args:
        config_file: Path to TOML configuration file
        sample_size: Number of questions to test
        
    Returns:
        Test results
    """
    logger.info("Running quick test", sample_size=sample_size)
    
    return await run_benchmark(
        config_file=config_file,
        sample_size=sample_size
    )


def print_results_summary(results: Dict[str, Any]) -> None:
    """
    Print a formatted summary of benchmark results.
    
    Args:
        results: Benchmark results
    """
    print("\n" + "="*60)
    print("LONGMEMEVAL BENCHMARK RESULTS")
    print("="*60)
    config_dict = results.get('config') if hasattr(results, 'get') else results.config.model_dump()
    print(f"Dataset Variant: {config_dict.get('dataset_variant', 'unknown')}")
    print(f"Total Questions: {results['total_questions']}")
    print(f"Correct Answers: {results['correct_answers']}")
    print(f"Overall Accuracy: {results['overall_accuracy']:.4f}")
    print(f"Runtime: {results['total_runtime_seconds']:.2f} seconds")
    print(f"Avg Time/Question: {results['average_time_per_question']:.2f} seconds")
    
    print("\nAccuracy by Question Type:")
    print("-" * 30)
    for qtype, accuracy in results['accuracy_by_type'].items():
        count = results['counts_by_type'][qtype]
        print(f"{qtype:25s}: {accuracy:.4f} ({count} questions)")
    
    print("\n" + "="*60)


if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser(description="LangGraph LongMemEval Benchmarker")
    parser.add_argument(
        "--config", 
        default="config.toml",
        help="Path to TOML configuration file"
    )
    parser.add_argument(
        "--mode",
        choices=["full", "quick-test"],
        default="full",
        help="Execution mode: full benchmark or quick test"
    )
    parser.add_argument(
        "--sample-size", 
        type=int,
        help="Sample size for testing (overrides config file)"
    )
    
    args = parser.parse_args()
    
    async def main():
        if args.mode == "quick-test":
            sample_size = args.sample_size if args.sample_size else 5
            results = await quick_test(args.config, sample_size=sample_size)
        else:
            results = await run_benchmark(
                config_file=args.config,
                sample_size=args.sample_size
            )
        
        print_results_summary(results)
    
    asyncio.run(main())