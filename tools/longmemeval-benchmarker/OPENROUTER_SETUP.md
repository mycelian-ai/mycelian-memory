# OpenRouter Setup Guide

## What is OpenRouter?
OpenRouter provides unified access to multiple LLM providers (OpenAI, Anthropic, Google, Meta, etc.) through a single API endpoint.

## Setup Steps

### 1. Get an API Key
1. Go to https://openrouter.ai
2. Sign up or log in
3. Navigate to API Keys section
4. Create a new API key
5. Add credits to your account

### 2. Set Environment Variable
```bash
export OPENROUTER_API_KEY="sk-or-v1-your-key-here"

# Optional: Set site/app name for tracking
export OPENROUTER_SITE_NAME="longmemeval-benchmarker"
export OPENROUTER_APP_NAME="LongMemEval"
```

### 3. Available Models
Popular models available through OpenRouter:
- `openrouter:anthropic/claude-3.5-sonnet`
- `openrouter:anthropic/claude-3-opus`
- `openrouter:google/gemini-pro-1.5`
- `openrouter:meta-llama/llama-3.1-405b-instruct`
- `openrouter:openai/gpt-4o`
- `openrouter:openai/gpt-5-nano-2025-08-07`

Check https://openrouter.ai/models for full list and pricing.

## Testing OpenRouter

### Quick Test with Debug QA
```bash
# Set your API key
export OPENROUTER_API_KEY="sk-or-v1-your-key-here"

# Test with Claude 3.5 Sonnet
python debug_qa.py \
  --memory-id 610f8553-a1fa-4710-b365-d1bc3c07b0cb \
  --vault-id e04ca555-b87e-490a-9807-5c7577c4e226 \
  --question "What color was Max the dog?" \
  --model "openrouter:anthropic/claude-3.5-sonnet"

# Test with Gemini Pro 1.5
python debug_qa.py \
  --memory-id 610f8553-a1fa-4710-b365-d1bc3c07b0cb \
  --vault-id e04ca555-b87e-490a-9807-5c7577c4e226 \
  --question "What color was Max the dog?" \
  --model "openrouter:google/gemini-pro-1.5"
```

### Using in Config Files
```toml
[models]
# Use OpenRouter models
agent = "openrouter:anthropic/claude-3.5-sonnet"
qa = "openrouter:openai/gpt-4o"
```

## Cost Tracking
OpenRouter shows usage and costs in real-time at:
https://openrouter.ai/activity

## Advantages
- Single API key for multiple providers
- Automatic fallback between providers
- Usage tracking and analytics
- Often cheaper than direct provider access
- No need for multiple provider accounts

## Troubleshooting

### API Key Not Working
- Ensure you have credits in your OpenRouter account
- Check the API key starts with `sk-or-v1-`
- Verify the key is set in environment: `echo $OPENROUTER_API_KEY`

### Model Not Found
- Check model ID at https://openrouter.ai/models
- Some models require explicit permission or higher tier
- Use exact model ID as shown on OpenRouter

### Rate Limits
- OpenRouter has its own rate limits separate from providers
- Check your tier limits at https://openrouter.ai/settings/limits