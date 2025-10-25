# Qwen Chat Template Support

## Overview

The Qwen3 model requires specific chat template formatting. This is handled automatically by llama-server when started with the `--jinja` flag.

## llama-server Configuration

The llama-server automatically applies the Qwen chat template when:
1. Started with `--jinja` flag
2. The model file contains Qwen template metadata
3. Messages are sent via OpenAI-compatible API format

## Message Format

The agent sends messages in OpenAI format:

```json
{
  "role": "system|user|assistant|tool",
  "content": "message content"
}
```

llama-server converts these to Qwen's format automatically:

```
<|im_start|>system
{content}<|im_end|>
<|im_start|>user
{content}<|im_end|>
<|im_start|>assistant
{content}<|im_end|>
```

## Tool Calling

For tool/function calling, llama-server handles the conversion based on the Qwen model's training:
- Function definitions in OpenAI format
- Function calls in JSON format
- Tool results as separate messages

## Troubleshooting

If you encounter chat template issues:

1. Verify llama-server started with `--jinja` flag
2. Check llama-server logs for template loading
3. Ensure model file has proper metadata
4. Test with simple completion first before using tools

## Custom Template

If needed, you can provide a custom template file to llama-server:
```bash
llama-server --chat-template /path/to/template.jinja
```

However, the bundled Qwen template should work out of the box.
