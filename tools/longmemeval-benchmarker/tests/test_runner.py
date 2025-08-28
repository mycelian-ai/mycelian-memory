import os
import tempfile
import textwrap
from contextlib import redirect_stdout
from io import StringIO

# Ensure package import works
import sys
sys.path.insert(0, os.path.dirname(os.path.dirname(__file__)))

from longmemeval_benchmarker.runner import main


def test_runner_prints_loaded_config(capfd):
    toml_text = textwrap.dedent(
        """
        dataset_repo_path = "/tmp"
        vault_title = "longmemeval"
        [provider]
        type = "openai"
        [models]
        agent = "gpt-4o-mini"
        qa = "gpt-4o-mini"
        eval = "gpt-4o-mini"
        """
    ).strip()

    with tempfile.TemporaryDirectory() as tmpdir:
        cfg_path = os.path.join(tmpdir, "run.toml")
        with open(cfg_path, "w", encoding="utf-8") as f:
            f.write(toml_text)
        # Simulate CLI
        sys.argv = ["runner.py", cfg_path]
        main()
        out, err = capfd.readouterr()
        assert "Loaded config for vault_title=longmemeval" in out
