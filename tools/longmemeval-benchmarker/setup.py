"""Setup script for LongMemEval integration."""

from setuptools import setup, find_packages

setup(
    name="longmemeval-integration",
    version="0.1.0",
    description="Clean LongMemEval integration with Mycelian Memory",
    packages=find_packages(where="src"),
    package_dir={"": "src"},
    install_requires=[
        "langgraph>=0.2.0",
        "langchain>=0.3.0",
        "langchain-openai>=0.2.0",
        "langchain-mcp-adapters>=0.1.0",
        "pandas>=2.0.0",
        "pytest>=7.0.0",
        "pytest-asyncio>=0.21.0",
    ],
)
