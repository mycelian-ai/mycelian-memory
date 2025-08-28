import os
import tempfile
import textwrap
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(__file__)))

from longmemeval_benchmarker.runner import main, parse_config, build_memory_title


def _toml_text(tmp_dataset_dir: str) -> str:
    return textwrap.dedent(
        f"""
        dataset_repo_path = "{tmp_dataset_dir}"
        vault_title = "longmemeval"
        memory_title_template = "{{question_id}}__{{run_id}}"
        [provider]
        type = "openai"
        [models]
        agent = "gpt-4o-mini"
        qa = "gpt-4o-mini"
        eval = "gpt-4o-mini"
        [params]
        top_k = 10
        use_llm_judge = true
        max_tool_calls_per_turn = 5
        """
    ).strip()


def test_parse_config_and_title_builder(tmp_path):
    cfg_path = tmp_path / "run.toml"
    dspath = tmp_path / "data"
    dspath.mkdir()
    cfg_text = _toml_text(str(dspath))
    cfg_path.write_text(cfg_text, encoding="utf-8")

    import tomllib
    raw_cfg = tomllib.loads(cfg_text)
    cfg = parse_config(raw_cfg)

    title = build_memory_title(cfg.memory_title_template, "Q-123", "R-1")
    assert title == "Q-123__R-1"


def test_runner_prints_loaded_config(capfd, tmp_path):
    cfg_path = tmp_path / "run.toml"
    dspath = tmp_path / "data"
    dspath.mkdir()
    cfg_text = _toml_text(str(dspath))
    cfg_path.write_text(cfg_text, encoding="utf-8")

    sys.argv = ["runner.py", str(cfg_path)]
    main()
    out, err = capfd.readouterr()
    assert "Loaded config:" in out
