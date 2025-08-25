from __future__ import annotations

import json
import asyncio
from typing import Any, Dict, List, Optional


class _SimpleTextBlock:
    def __init__(self, text: str):
        self.type = "text"
        self.text = text


class _SimpleToolUseBlock:
    def __init__(self, name: str, input_obj: Dict[str, Any], block_id: Optional[str] = None):
        self.type = "tool_use"
        self.name = name
        self.input = input_obj
        self.id = block_id


class _SimpleResponse:
    def __init__(self, content: List[Any], stop_reason: Optional[str] = None):
        self.content = content
        self.stop_reason = stop_reason


class BedrockAnthropicAdapter:
    """Async-compatible adapter that mimics anthropic.AsyncAnthropic(messages.create).

    It wraps AWS Bedrock Runtime `invoke_model` for Anthropic Claude models and returns
    response objects with `.content` blocks compatible with the benchmarker's
    SessionSimulator expectations.
    """

    class _Messages:
        def __init__(self, client, region_name: str):
            self._client = client
            self._region = region_name

        async def create(self, *, model: str, messages: List[Dict[str, Any]], tools: Optional[List[Dict[str, Any]]] = None, max_tokens: int = 200, system: Optional[str] = None, **kwargs):
            # Convert messages to Bedrock content-block format when needed
            def _to_blocks(content: Any) -> List[Dict[str, Any]]:
                if isinstance(content, list):
                    return content  # assume already in block format
                # Wrap plain string as a single text block
                return [{"type": "text", "text": str(content)}]

            bedrock_messages: List[Dict[str, Any]] = []
            for m in messages:
                bedrock_messages.append({
                    "role": m.get("role", "user"),
                    "content": _to_blocks(m.get("content", "")),
                })

            payload: Dict[str, Any] = {
                "anthropic_version": "bedrock-2023-05-31",
                "messages": bedrock_messages,
                "max_tokens": max_tokens,
            }
            if system:
                # Bedrock accepts system as either a string or content blocks. Use string for simplicity.
                payload["system"] = system
            if tools:
                payload["tools"] = tools

            body = json.dumps(payload)

            # Run the synchronous boto3 call in a thread to keep async contract
            import boto3  # local import to avoid hard dependency when unused

            def _invoke() -> Dict[str, Any]:
                brt = boto3.client("bedrock-runtime", region_name=self._region)
                resp = brt.invoke_model(
                    modelId=model,
                    contentType="application/json",
                    accept="application/json",
                    body=body,
                )
                # For non-streaming, body is a StreamingBody
                data = resp.get("body")
                raw = data.read().decode("utf-8") if hasattr(data, "read") else str(data)
                return json.loads(raw)

            resp_obj = await asyncio.to_thread(_invoke)

            # Map Bedrock response to simple blocks
            content_blocks: List[Any] = []
            for blk in resp_obj.get("content", []):
                blk_type = blk.get("type")
                if blk_type == "text":
                    content_blocks.append(_SimpleTextBlock(blk.get("text", "")))
                elif blk_type == "tool_use":
                    content_blocks.append(_SimpleToolUseBlock(
                        name=blk.get("name", ""),
                        input_obj=blk.get("input", {}) or {},
                        block_id=blk.get("id"),
                    ))

            return _SimpleResponse(content_blocks, stop_reason=resp_obj.get("stop_reason"))

    def __init__(self, region_name: Optional[str] = None):
        if not region_name:
            # boto3 will fall back to environment/default config if region not provided
            region_name = None
        self.messages = BedrockAnthropicAdapter._Messages(client=None, region_name=region_name or "us-east-1")


def new_model_client(provider: str, anthropic_key: Optional[str], aws_region: Optional[str]):
    provider_lc = (provider or "anthropic").lower()
    if provider_lc == "anthropic":
        import anthropic  # defer import
        if not anthropic_key:
            raise ValueError("ANTHROPIC_API_KEY is required for provider=anthropic")
        return anthropic.AsyncAnthropic(api_key=anthropic_key)
    if provider_lc == "bedrock":
        return BedrockAnthropicAdapter(region_name=aws_region)
    raise ValueError(f"Unsupported provider: {provider}")


