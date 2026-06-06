# LLM Mock Server

> 模拟多 LLM 供应商 API 的 mock server，用于协议兼容性测试。支持 OpenAI、Anthropic、DeepSeek 等多种协议标准。

## 使用方法

### 基本启动

```bash
./llm-mock-server --server-port 3000
```

### 启用鉴权

```bash
./llm-mock-server --auth sk-test-key-123
```

启动后，所有请求必须携带鉴权信息：
- **OpenAI 协议**：`Authorization: Bearer sk-test-key-123`
- **Anthropic 协议**：`x-api-key: sk-test-key-123`

不传 `--auth` 则跳过鉴权，所有请求均可访问。

### 指定配置文件

```bash
./llm-mock-server --config /path/to/config.yaml
```

### 指定单一供应商（兼容旧模式）

```bash
./llm-mock-server --provider-type deepseek
```

### 参数一览

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--server-port` | `3000` | 服务监听端口 |
| `--auth` | (空) | API 鉴权密钥，留空则不鉴权 |
| `--config` | (空) | YAML 配置文件路径，覆盖 `--provider-type` |
| `--provider-type` | (空) | 单一供应商类型（旧模式） |

## 标准端点

以下三个端点**始终可用**，不受配置文件影响：

| 协议 | 端点 | 说明 |
|------|------|------|
| OpenAI Chat Completions | `POST /v1/chat/completions` | 标准 Chat Completions API |
| Anthropic Messages | `POST /v1/messages` | Anthropic Messages API |
| OpenAI Responses | `POST /v1/responses` | OpenAI Responses API |

## 配置文件

通过 YAML 配置文件控制启用哪些额外供应商的协议端点。默认配置文件 `config.yaml`：

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

## 可选供应商

- 360 智脑、Cloudflare、Dify、Gemini、GitHub、Groq、MiniMax、Together AI
- 百川智能、豆包、零一万物、文心一言、智谱 AI、阶跃星辰

## 特性

### 动态 Token 计算

基于输入字符数动态计算 token 用量，**不再返回固定值**：

| 字段 | 计算规则 |
|------|----------|
| `prompt_tokens` | = 输入字符数 |
| `completion_tokens` | = 输入字符数 × 10%（最少 1） |
| `total_tokens` | = prompt_tokens + completion_tokens |

### 缓存 Token 支持（90% 命中率）

缓存命中比例固定为输入 token 的 **90%**，按协议格式适配字段。

**OpenAI / DeepSeek 协议**（`/v1/chat/completions`、`/v1/responses`）：

```json
{
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 10,
    "total_tokens": 110,
    "prompt_tokens_details": {
      "cached_tokens": 90
    },
    "prompt_cache_hit_tokens": 90,
    "prompt_cache_miss_tokens": 10
  }
}
```

**Anthropic 协议**（`/v1/messages`）：

```json
{
  "usage": {
    "input_tokens": 100,
    "output_tokens": 10,
    "cache_creation_input_tokens": 10,
    "cache_read_input_tokens": 90
  }
}
```

### 输出策略

响应内容从输入文本中**随机截取 10% 长度的连续子串**，模拟真实输出场景。

### 任意 Model ID

请求中的 `model` 字段可以是任意值，不做校验，方便各类协议测试。

### DeepSeek 思考模式

请求中设置 `"thinking": {"type": "enabled"}` 时，响应包含 `reasoning_content` 字段：

```json
{
  "choices": [{
    "message": {
      "content": "截取的输出内容...",
      "reasoning_content": "Thinking process: analyzing the input..."
    }
  }]
}
```

流式响应中先发送 `reasoning_content` 分块，再发送 `content` 分块。

### 流式响应

所有端点均支持流式（`"stream": true`），流式最后一个 chunk 始终包含完整 usage 信息。

## Docker

```bash
docker build -t llm-mock-server .
docker run -p 3000:3000 llm-mock-server --auth sk-test-key
```
