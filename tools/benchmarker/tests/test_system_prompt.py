from benchmarks.python.system_prompt_builder import PromptAssembler


def test_system_prompt_contains_sections():
    builder = PromptAssembler(
        benchmark_name="DMR",
        memory_id="mem-456",
        context_doc="Sample context",
        recent_entries=[
            {"raw_entry": "User: hi", "summary": "greeting"},
            {"raw_entry": "Assistant: hello", "summary": "reply"},
        ],
        debug=True,
    )
    prompt = builder.build()
    assert "DMR" in prompt
    assert "Memory ID: mem-456" in prompt
    assert "Sample context" in prompt
    # Asset list present
    assert "ctx_rules" in prompt
    # Tool schema header still present
    assert "# MCP TOOL SCHEMA" in prompt 