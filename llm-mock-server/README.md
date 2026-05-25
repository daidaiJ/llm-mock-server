# LLM Mock Server

> 模拟所有 LLM 供应商 API 的 mock server，专为 Higress e2e 测试设计。

## 使用方法

### 默认启动（使用 config.yaml）

```bash
./llm-mock-server --server-port 3000
```

默认启用 `config.yaml` 中配置的供应商（deepseek、openai、anthropic）。

### 指定配置文件

```bash
./llm-mock-server --config /path/to/config.yaml
```

### 指定单一供应商（兼容旧模式）

```bash
./llm-mock-server --provider-type deepseek
```

## 配置文件

通过 YAML 配置文件控制启用哪些供应商的协议端点。默认配置文件 `config.yaml`：

```yaml
providers:
  - name: openai
    enabled: true
  - name: deepseek
    enabled: true
  - name: anthropic
    enabled: true
  - name: qwen
    enabled: false
  # ... 更多供应商
```

## 支持的供应商

### 默认启用

| 供应商 | 端点 | 说明 |
|--------|------|------|
| OpenAI | `POST /v1/chat/completions` | Chat Completions API |
| OpenAI | `POST /v1/responses` | Responses API |
| DeepSeek | `POST /v1/chat/completions` | 含 `reasoning_content`、`thinking` 支持 |
| Anthropic | `POST /v1/messages` | 含缓存 token 字段 |

### 可选供应商

- 360 智脑
- Cloudflare
- Dify
- Gemini
- GitHub
- Groq
- MiniMax
- Together AI
- 百川智能
- 豆包
- 零一万物
- 文心一言
- 智谱 AI
- 阶跃星辰

## 特性

### 缓存 Token 支持

所有 OpenAI 兼容端点的流式和非流式响应均包含缓存 token 信息：

```json
{
  "usage": {
    "prompt_tokens": 9,
    "completion_tokens": 1,
    "total_tokens": 10,
    "prompt_tokens_details": {
      "cached_tokens": 5
    },
    "prompt_cache_hit_tokens": 5,
    "prompt_cache_miss_tokens": 4
  }
}
```

### DeepSeek 思考模式

请求中设置 `"thinking": {"type": "enabled"}` 时，响应包含 `reasoning_content` 字段：

```json
{
  "choices": [{
    "message": {
      "content": "Hello!",
      "reasoning_content": "Thinking process: analyzing the input..."
    }
  }]
}
```

流式响应中先发送 `reasoning_content` 分块，再发送 `content` 分块。

### Anthropic 缓存 Token

Anthropic 端点响应中包含缓存 token 字段：

```json
{
  "usage": {
    "input_tokens": 9,
    "output_tokens": 1,
    "cache_creation_input_tokens": 5,
    "cache_read_input_tokens": 4
  }
}
```
