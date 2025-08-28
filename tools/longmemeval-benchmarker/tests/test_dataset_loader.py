import os
import tempfile
import pytest

# Ensure package import works when running tests directly
import sys
sys.path.insert(0, os.path.dirname(os.path.dirname(__file__)))

from longmemeval_benchmarker.dataset_loader import load_longmemeval


def test_load_raises_on_invalid_path():
    with pytest.raises(ValueError):
        _ = list(load_longmemeval("/path/does/not/exist"))


def test_load_empty_on_valid_dir():
    with tempfile.TemporaryDirectory() as tmpdir:
        items = list(load_longmemeval(tmpdir))
        assert items == []
