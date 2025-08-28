import argparse
import tomllib
from typing import Any


def main() -> None:
    parser = argparse.ArgumentParser(description="Run LongMemEval benchmarker")
    parser.add_argument("config", help="Path to TOML config")
    parser.add_argument("--mode", choices=["ingestion", "qa", "eval"], default=None)
    args = parser.parse_args()

    with open(args.config, "rb") as f:
        cfg: dict[str, Any] = tomllib.load(f)

    mode = args.mode or "end_to_end"
    print(f"Loaded config for vault_title={cfg.get('vault_title')} provider={cfg.get('provider', {}).get('type')} mode={mode}")


if __name__ == "__main__":
    main()
