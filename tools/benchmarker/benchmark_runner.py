from __future__ import annotations

import asyncio
import logging
import argparse
import os
from pathlib import Path
from typing import Dict, List, Any
import json
import sys
from datetime import datetime

import anthropic
from model_provider import new_model_client

# Direct local imports for tool execution
from mycelian_client import MycelianMemoryClient
from collaborative_builder import ConversationIngestor
from system_prompt_builder import PromptAssembler
from session_simulator import SessionSimulator
from msc_loader import load_msc_dataset
from models import BenchmarkConversation, TestQuestion

logger = logging.getLogger(__name__)


class BenchmarkEvaluator:
    def __init__(
        self,
        ac: anthropic.Anthropic,
        mc: MycelianMemoryClient,
        *,
        model_name: str = "claude-3-haiku-20240307",
        verbose: bool = False,
    ):
        self._ac = ac
        self._mc = mc
        self._verbose = verbose
        self._model_name = model_name

    async def evaluate_conversation(self, conv: BenchmarkConversation, memory_id: str, max_questions: int | None = None) -> List[Dict[str, Any]]:
        results: List[Dict[str, Any]] = []
        questions_iter = conv.test_questions[:max_questions] if max_questions else conv.test_questions
        for q in questions_iter:
            res = await self._eval_question(memory_id, q)
            results.append(res)
        return results

    async def _eval_question(self, memory_id: str, tq: TestQuestion) -> Dict[str, Any]:
        # Retrieve top 5 matching entries (rich context: raw + summary)
        search = self._mc.search_memories(memory_id, tq.question, top_k=5)
        entries = search.get("entries", [])
        latest_ctx = search.get("latestContext")
        best_ctx = search.get("bestContext")
        ctx_lines: list[str] = []
        for idx, e in enumerate(entries, start=1):
            raw = e.get("rawEntry", "").replace("\n", " ")
            summary = e.get("summary", "")
            ctx_lines.append(f"[# {idx}] RAW: {raw}")
            ctx_lines.append(f"[# {idx}] SUMMARY: {summary}")

            # Detailed per-entry logs when verbose is enabled
            if self._verbose:
                logger.info("ENTRY %d RAW: %s", idx, raw)
                logger.info("ENTRY %d SUMMARY: %s", idx, summary)

        # Append latest and best context blocks if available
        if latest_ctx:
            try:
                latest_str = latest_ctx if isinstance(latest_ctx, str) else json.dumps(latest_ctx, ensure_ascii=False)
            except TypeError:
                latest_str = str(latest_ctx)
            ctx_lines.append("--- LATEST CONTEXT ---")
            ctx_lines.append(latest_str)
            if self._verbose:
                logger.info("LATEST CONTEXT: %s", latest_str)

        if best_ctx:
            try:
                best_str = best_ctx if isinstance(best_ctx, str) else json.dumps(best_ctx, ensure_ascii=False)
            except TypeError:
                best_str = str(best_ctx)
            ctx_lines.append("--- BEST CONTEXT ---")
            ctx_lines.append(best_str)
            if self._verbose:
                logger.info("BEST CONTEXT  : %s", best_str)

        ctx = "\n".join(ctx_lines)

        prompt = (
            "Answer the question using the memory context provided.\n"
            "The CONTEXT section contains, in order:\n"
            "  • The top-5 most relevant memory entries (RAW and SUMMARY).\n"
            "  • '--- LATEST CONTEXT ---' – the incremental conversation context, optimised for recency (lossy).\n"
            "  • '--- BEST CONTEXT ---' – an earlier context snapshot chosen for highest fidelity.\n"
            "Use any of this information to answer accurately.\n\n"
            "CONTEXT:\n" + ctx + "\n\n" +
            f"QUESTION: {tq.question}\nANSWER:"
        )

        logger.info("QUESTION        : %s", tq.question)
        logger.info("EXPECTED ANSWER : %s", tq.expected_answer)
        logger.info("Retrieved %d entries from search", len(entries))

        # Detailed debug – full context string
        logger.debug("Context sent to LLM:\n%s", ctx)

        resp = await self._ac.messages.create(
            model=self._model_name,
            messages=[{"role": "user", "content": prompt}],
            max_tokens=200,
        )
        answer = resp.content[0].text.strip()

        # ------------------------------------------------------------------
        # 2. Let the LLM judge correctness to allow semantic matches
        # ------------------------------------------------------------------
        judge_prompt = (
            "You are evaluating whether a model correctly answered a question. "
            "The model answer may be verbose or contain additional context, but focus on whether "
            "it contains the correct core information that matches the expected answer. "
            "Consider semantic equivalence - if the model provides the right information even "
            "with extra explanation, that should be marked as CORRECT.\n\n"
            f"QUESTION: {tq.question}\n"
            f"EXPECTED_ANSWER: {tq.expected_answer}\n"
            f"MODEL_ANSWER: {answer}\n\n"
            "Does the model answer contain the correct information to answer the question? "
            "Respond with exactly one word: 'CORRECT' or 'INCORRECT'.\n"
            "JUDGEMENT:"
        )

        judge_resp = await self._ac.messages.create(
            model=self._model_name,
            messages=[{"role": "user", "content": judge_prompt}],
            max_tokens=5,
        )

        judge_text = judge_resp.content[0].text.strip().lower()
        is_correct = "correct" in judge_text and "incorrect" not in judge_text

        logger.info("MODEL ANSWER    : %s", answer)
        logger.info("LLM JUDGEMENT   : %s", judge_text.upper())
        logger.info("EVALUATION      : %s", "CORRECT ✅" if is_correct else "INCORRECT ❌")

        return {
            "question": tq.question,
            "expected": tq.expected_answer,
            "answer": answer,
            "is_correct": is_correct,
            "question_type": tq.question_type,
        }


class BenchmarkRunner:
    def __init__(
        self,
        ac_client,
        mycelian_url: str,
        *,
        model_name: str = "claude-3-haiku-20240307",
        verbose: bool = False,
    ): 
        """Create a new benchmark runner for dev mode (no user management)."""

        # Use asynchronous Anthropic client for non-blocking IO.
        self._ac = ac_client
        self._mc = MycelianMemoryClient(mycelian_url)
        self._model_name = model_name
        self._builder = ConversationIngestor(self._ac, self._mc, model_name=self._model_name)
        self._evaluator = BenchmarkEvaluator(self._ac, self._mc, model_name=self._model_name, verbose=verbose)

    async def ingest_only(self, max_conversations: int | None, tracker_file: str, max_messages: int | None = None, *, vault_title: str = "benchmarker"):
        """Ingest conversations and write tracker; skip evaluation."""
        # Use a persistent benchmarker vault (get-or-create)
        vault_id = self._mc.get_or_create_vault(vault_title, "Persistent vault for benchmarker runs")
        logger.info("Using benchmark vault %s (%s) for %d conversations", vault_id, vault_title, max_conversations or 0)
        
        conversations = load_msc_dataset(max_conv=max_conversations)
        tracker_records: list[dict[str, str]] = []
        for conv in conversations:
            mem_id = await self._builder.process_conversation(
                conv,
                benchmark_name="MSC",
                max_messages_per_session=max_messages,
                vault_id=vault_id,
            )
            tracker_records.append({
                "conversation_id": conv.conversation_id,
                "memory_id": mem_id,
                "vault_id": vault_id,
            })
        Path(tracker_file).parent.mkdir(parents=True, exist_ok=True)
        with open(tracker_file, "w", encoding="utf-8") as f:
            json.dump(tracker_records, f, indent=2)
        logger.info("Ingestion complete for %d conversations. Tracker written to %s", len(conversations), tracker_file)

    async def evaluate_only(self, tracker_file: str, output: str | None = None, max_questions: int | None = None):
        """Evaluate existing ingested conversations using tracker file."""
        with open(tracker_file, "r", encoding="utf-8") as f:
            tracker_records = json.load(f)
        # User management removed - dev mode only
        # Build lookup map for tracker
        map_mem = {r["conversation_id"]: r for r in tracker_records}
        # Load full dataset to fetch questions
        conversations = load_msc_dataset()
        all_results: List[Dict[str, Any]] = []
        for conv in conversations:
            if conv.conversation_id not in map_mem:
                continue  # skip not ingested
            memory_id = map_mem[conv.conversation_id]["memory_id"]
            conv_res = await self._evaluator.evaluate_conversation(conv, memory_id, max_questions=max_questions)
            all_results.extend(conv_res)

            # ------------------------------------------------------------------
            # Pretty-print summary table per conversation (stdout – visible even without --verbose)
            # ------------------------------------------------------------------
            header = ["#", "Question", "Expected", "Model Answer", "Correct"]
            rows = []
            for idx, r in enumerate(conv_res, start=1):
                rows.append([
                    idx,
                    r["question"].strip().replace("\n", " ")[:60] + ("…" if len(r["question"]) > 60 else ""),
                    r["expected"].strip().replace("\n", " ")[:40] + ("…" if len(r["expected"]) > 40 else ""),
                    r["answer"].strip().replace("\n", " ")[:40] + ("…" if len(r["answer"]) > 40 else ""),
                    "✅" if r["is_correct"] else "❌",
                ])

            col_widths = [max(len(str(cell)) for cell in col) for col in zip(header, *rows)]
            fmt_row = " | ".join(f"{{:<{w}}}" for w in col_widths)
            separator = "-+-".join("-" * w for w in col_widths)

            table_lines = [fmt_row.format(*header), separator]
            for row in rows:
                table_lines.append(fmt_row.format(*row))

            logger.info("\nResults for conversation %s\n%s", conv.conversation_id, "\n".join(table_lines))
        if output:
            Path(output).parent.mkdir(parents=True, exist_ok=True)
            with open(output, "w", encoding="utf-8") as f:
                json.dump(all_results, f, indent=2)
        accuracy = sum(r["is_correct"] for r in all_results) / len(all_results) if all_results else 0.0
        logger.info("Accuracy %.1f%% over %d questions", accuracy * 100, len(all_results))

    async def run_msc(self, max_conversations: int | None = 1, output: str | None = None, tracker_file: str | None = None, *, vault_title: str = "benchmarker"):
        """Ingest and evaluate the MemGPT MSC-Self-Instruct benchmark."""
        # Use a persistent benchmarker vault (get-or-create)
        vault_id = self._mc.get_or_create_vault(vault_title, "Persistent vault for benchmarker runs")
        logger.info("Using benchmark vault %s (%s) for %d conversations", vault_id, vault_title, max_conversations or 0)
        
        conversations = load_msc_dataset(max_conv=max_conversations)
        all_results: List[Dict[str, Any]] = []
        tracker_records: list[dict[str, str]] = []
        for idx, conv in enumerate(conversations, start=1):
            logger.info("Processing conversation %d/%d (id=%s)", idx, len(conversations), conv.conversation_id)
            memory_id = await self._builder.process_conversation(conv, benchmark_name="MSC", vault_id=vault_id)
            conv_results = await self._evaluator.evaluate_conversation(conv, memory_id)
            all_results.extend(conv_results)
            # record mapping for downstream evaluation
            tracker_records.append({
                "conversation_id": conv.conversation_id,
                "memory_id": memory_id,
                "vault_id": vault_id,
            })

            # Print per-conversation summary table identical to evaluate_only
            header = ["#", "Question", "Expected", "Model Answer", "Correct"]
            rows = []
            for q_idx, r in enumerate(conv_results, start=1):
                rows.append([
                    q_idx,
                    r["question"].strip().replace("\n", " ")[:60] + ("…" if len(r["question"]) > 60 else ""),
                    r["expected"].strip().replace("\n", " ")[:40] + ("…" if len(r["expected"]) > 40 else ""),
                    r["answer"].strip().replace("\n", " ")[:40] + ("…" if len(r["answer"]) > 40 else ""),
                    "✅" if r["is_correct"] else "❌",
                ])

            col_widths = [max(len(str(cell)) for cell in col) for col in zip(header, *rows)]
            fmt_row = " | ".join(f"{{:<{w}}}" for w in col_widths)
            separator = "-+-".join("-" * w for w in col_widths)

            table_lines = [fmt_row.format(*header), separator]
            for row in rows:
                table_lines.append(fmt_row.format(*row))

            logger.info("\nResults for conversation %s\n%s", conv.conversation_id, "\n".join(table_lines))
        if output:
            Path(output).parent.mkdir(parents=True, exist_ok=True)
            with open(output, "w", encoding="utf-8") as f:
                json.dump({
                    "timestamp": datetime.utcnow().isoformat(),
                    "results": all_results,
                }, f, indent=2)
        # Persist tracker if requested
        if tracker_file:
            Path(tracker_file).parent.mkdir(parents=True, exist_ok=True)
            with open(tracker_file, "w", encoding="utf-8") as f:
                json.dump(tracker_records, f, indent=2)

        accuracy = sum(r["is_correct"] for r in all_results) / len(all_results) if all_results else 0.0
        logger.info("Accuracy %.1f%% over %d questions", accuracy * 100, len(all_results))
        return accuracy


async def _cli_run_validate_tools(args):
    """Interactive validation that Claude can exercise every MCP tool."""
    import os
    import anthropic
    from session_simulator import SessionSimulator

    # ------------------------------------------------------------------
    # 1. Set up clients
    # ------------------------------------------------------------------
    provider = (getattr(args, "provider", None) or os.getenv("BENCHMARK_PROVIDER", "anthropic")).lower()
    anthropic_key = os.environ.get("ANTHROPIC_API_KEY")
    aws_region = getattr(args, "aws_region", None) or os.getenv("AWS_REGION")

    if provider == "anthropic" and not anthropic_key:
        raise SystemExit("ANTHROPIC_API_KEY must be set for provider=anthropic")
    ac = new_model_client(provider, anthropic_key, aws_region)

    sc = MycelianMemoryClient(base_url=args.mycelian_url)

    # ------------------------------------------------------------------
    # 2. Build initial system prompt (no memory yet – will be created by LLM)
    # ------------------------------------------------------------------
    spb = PromptAssembler(
        benchmark_name="validate-tools",
        memory_id="",  # blank – LLM will create memory via create_memory
        context_doc="",
        recent_entries=[],
    )

    sim = SessionSimulator(ac, sc, spb, model_name=args.model_name)

    # ------------------------------------------------------------------
    # 3. Seed conversation instructing the model to call every tool
    # ------------------------------------------------------------------
    CONTROL_PREFIX = "control:test_harness"
    seed = (
        f"{CONTROL_PREFIX} VALIDATE_TOOLS "
        "You are running an integration self-check for the Mycelian tools. "
        "First, call get_tools_schema to fetch the live tools schema. "
        "Then, call get_default_prompts with memory_type='chat' and use the returned context_summary_rules and templates. "
        "Next, exercise the key tools in a logical sequence: create storage, add content, verify consistency, and retrieve data. "
        "Use real UUIDs returned by each tool call - never use placeholder strings. "
        "When calling add_entry, include the role parameter with either 'speaker_1' or 'speaker_2'. "
        "After successfully exercising the tools, reply with 'DONE'."
    )

    print("Sending validation prompt to Claude …")
    reply = await sim.step(seed)
    print("Model reply:", reply)

    # Simple loop: continue until DONE or 15 turns (configurable via VALIDATE_TURNS env)
    max_turns = int(os.getenv("VALIDATE_TURNS", "15"))
    for _ in range(max_turns - 1):
        if "DONE" in reply.upper():
            print("All tools exercised. Validation PASS ✅")
            return
        reply = await sim.step(f"{CONTROL_PREFIX} continue")
        print("Model reply:", reply)

    # Final check in case DONE was returned on the last allowed turn
    if "DONE" in reply.upper():
        print("All tools exercised. Validation PASS ✅")
        return

    print(f"Validation FAILED – model did not finish within {max_turns} turns", file=sys.stderr)
    raise SystemExit(1)


async def _cli_main():
    logging.basicConfig(format="%(asctime)s [%(levelname)s] %(message)s", level=logging.INFO, datefmt="%H:%M:%S")
    parser = argparse.ArgumentParser(description="Run Mycelian MSC benchmark")
    parser.add_argument("--mycelian-url", required=True, help="Base URL of the Mycelian service, e.g. http://localhost:11545")
    subparsers = parser.add_subparsers(dest="mode", required=True)

    subparsers.add_parser("both")
    subparsers.add_parser("ingest")
    subparsers.add_parser("eval")
    subparsers.add_parser("validate-tools")
    # Provider-specific configuration is taken from environment variables.
    # For provider=anthropic: require ANTHROPIC_API_KEY.
    # For provider=bedrock: require standard AWS credentials and AWS_REGION.
    parser.add_argument("--provider", default=os.getenv("BENCHMARK_PROVIDER", "anthropic"), choices=["anthropic", "bedrock"], help="Model provider to use (anthropic or bedrock)")
    parser.add_argument("--aws-region", default=os.getenv("AWS_REGION"), help="AWS region for Bedrock (e.g., us-east-1)")
    parser.add_argument("--model-name", default="claude-3-haiku-20240307", help="Anthropic Claude model to use for simulation (default: claude-3-haiku-20240307)")

    parser.add_argument("--conversations", type=int, default=1, help="Number of MSC conversations to process (default: 1)")
    parser.add_argument("--vault-title", default=os.getenv("BENCH_VAULT_TITLE", "benchmarker"), help="Fixed title of the persistent benchmarker vault (default: 'benchmarker')")
    parser.add_argument("--output", help="Optional path to write results JSON")
    parser.add_argument("--questions", type=int, help="Limit number of test questions per conversation during evaluation")
    parser.add_argument("--max-messages", type=int, help="Truncate each session to N messages (test mode)")
    parser.add_argument("--tracker-file", help="Path to write/read conversation→memory mapping JSON (required for ingest/eval modes)")

    parser.add_argument("--validate-prompts", action="store_true", help="Run a single prompt to validate assistant's understanding of rules")
    parser.add_argument("--verbose", action="store_true", help="Enable verbose logging for diagnostics")
    if hasattr(parser, "parse_intermixed_args"):
        try:
            args = parser.parse_intermixed_args()
        except TypeError as te:
            logging.debug("parse_intermixed_args failed (%s); falling back to parse_args", te)
            args = parser.parse_args()
    else:
        args = parser.parse_args()

    anthropic_key = os.getenv("ANTHROPIC_API_KEY")
    if args.provider == "anthropic" and not anthropic_key:
        parser.error("ANTHROPIC_API_KEY must be set for provider=anthropic")

    # User management removed - dev mode only

    ac_client = new_model_client(args.provider, anthropic_key, args.aws_region)

    runner = BenchmarkRunner(
        ac_client,
        args.mycelian_url,
        model_name=args.model_name or "claude-3-haiku-20240307",
        verbose=args.verbose,
    )

    if args.validate_prompts:
        # Create a throwaway memory and ask assistant to echo full rules.
        dummy_conv = BenchmarkConversation(conversation_id="validate", messages=[], test_questions=[], metadata={})
        mem_id = runner._mc.create_memory(title="prompt-validation", memory_type="conversation")  # Uses auto-vault creation
        sys_builder = PromptAssembler(
            benchmark_name="MSC",
            memory_id=mem_id,
            context_doc="",
            recent_entries=[],
        )
        sim = SessionSimulator(runner._ac, runner._mc, sys_builder, model_name=args.model_name)
        # Reduce CLI noise while keeping assistant output
        logging.getLogger("urllib3").setLevel(logging.WARNING)

        # Bootstrap control signal – expect ACK_RULES (discard)
        await sim.step("control:test_harness BOOTSTRAP")

        # Ask for full rules
        resp_rules = await sim.step(
            "Please list verbatim all operational rules you will follow, without omissions.",
            max_tokens=4096,
        )
        print(resp_rules)
        return

    if args.mode == "both":
        await runner.run_msc(max_conversations=args.conversations, output=args.output, tracker_file=args.tracker_file, vault_title=args.vault_title)
    elif args.mode == "ingest":
        if not args.tracker_file:
            parser.error("--tracker-file is required in ingest mode")
        await runner.ingest_only(max_conversations=args.conversations, tracker_file=args.tracker_file, max_messages=args.max_messages, vault_title=args.vault_title)
    elif args.mode == "eval":
        if not args.tracker_file:
            parser.error("--tracker-file is required in eval mode")
        await runner.evaluate_only(tracker_file=args.tracker_file, output=args.output, max_questions=args.questions)
    elif args.mode == "validate-tools":
        await _cli_run_validate_tools(args)


if __name__ == "__main__":
    asyncio.run(_cli_main()) 