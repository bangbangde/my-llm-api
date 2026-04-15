# LLM Gateway - 大模型接口调度网关

一个完全兼容 OpenAI API 规范的大模型接口调度网关，支持硅基流动等推理平台。

## 功能特性

- ✅ 完全兼容 OpenAI API 规范
- ✅ 支持硅基流动推理平台
- ✅ 支持流式响应 (Streaming Response)
- ✅ 可扩展的调度架构（工厂模式）
- ✅ 多供应商、多模型、多账号支持
- ✅ 账号池健康管理与自动故障转移
- ✅ Bearer Token API 认证
- ✅ CORS 白名单控制
- ✅ 请求重试机制（指数退避）
- ✅ 标准化错误处理
- ✅ 运行时统计监控
- ✅ OpenAI 官方 SDK 直接调用支持

## 项目结构

```
my-llm-api/
├── config/              # 配置管理
│   └── config.go        # YAML 配置加载
├── errors/              # 错误处理
│   └── errors.go        # 标准化错误定义
├── handlers/            # HTTP 处理器
│   └── chat.go          # 聊天补全接口
├── middleware/          # 中间件
│   └── middleware.go    # 认证、CORS、日志、恢复
├── models/              # 数据模型
│   └── openai.go        # OpenAI 兼容模型定义
├── providers/           # 供应商适配器
│   ├── provider.go      # Provider 接口定义、HTTPDoer 接口
│   └── siliconflow.go   # 硅基流动实现
├── scheduler/           # 调度系统
│   ├── scheduler.go     # 调度器核心
│   ├── account_pool.go  # 账号池管理
│   ├── factory.go       # 工厂模式依赖注入
│   └── retry.go        # 重试机制
├── config.yaml          # 配置文件
├── main.go              # 主程序入口
└── go.mod               # Go 模块定义
```

## 快速开始

### 1. 环境要求

- Go 1.21 或更高版本
- 硅基流动 API 密钥

### 2. 安装依赖

```bash
# 使用国内代理加速
go env GOPROXY=https://goproxy.cn,direct
go mod tidy
```

### 3. 配置

复制 `config.yaml.example` 为 `config.yaml` 并配置：

```yaml
server:
  port: 8080
  log_level: info
  # API keys for authenticating incoming requests.
  # Leave empty to disable authentication (not recommended for production).
  api_keys:
    - your-gateway-api-key-here
  # CORS allowed origins. Use ["*"] to allow all (dev only).
  allowed_origins:
    - https://your-frontend.example.com

providers:
  siliconflow:
    base_url: https://api.siliconflow.cn/v1
    accounts:
      - id: primary
        api_key: your-api-key-here
        weight: 1
        enabled: true
    models:
      - Qwen/Qwen2.5-7B-Instruct
      - Qwen/Qwen2.5-72B-Instruct
```

### 4. 构建

```bash
go build -o llm-gateway
```

### 5. 运行

```bash
./llm-gateway
```

服务器将在 `http://localhost:8080` 启动。

## API 端点

### 健康检查

```bash
GET /health
```

### 模型列表

```bash
GET /v1/models
```

### 聊天补全

```bash
POST /v1/chat/completions
```

请求示例：

```json
{
  "model": "Qwen/Qwen2.5-7B-Instruct",
  "messages": [
    {"role": "user", "content": "你好"}
  ],
  "max_tokens": 100
}
```

### 流式响应

```json
{
  "model": "Qwen/Qwen2.5-7B-Instruct",
  "messages": [
    {"role": "user", "content": "数到10"}
  ],
  "stream": true,
  "max_tokens": 100
}
```

## 使用 OpenAI SDK

### Python

```python
from openai import OpenAI

client = OpenAI(
    api_key="your-gateway-api-key",  # 使用配置的 API key
    base_url="http://localhost:8080/v1"
)

response = client.chat.completions.create(
    model="Qwen/Qwen2.5-7B-Instruct",
    messages=[
        {"role": "user", "content": "你好"}
    ]
)

print(response.choices[0].message.content)
```

### 流式响应

```python
stream = client.chat.completions.create(
    model="Qwen/Qwen2.5-7B-Instruct",
    messages=[
        {"role": "user", "content": "数到10"}
    ],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

## 测试

### 单元测试

```bash
go test ./... -v
```

### 端到端测试

```bash
# 启动服务器
./llm-gateway

# 在另一个终端运行测试（需要 build tag）
go test -tags=integration -run TestIntegration -v
```

### 使用测试客户端

```bash
python test_client.py
```

## 支持的模型

- Qwen/Qwen2.5-7B-Instruct
- Qwen/Qwen2.5-72B-Instruct
- deepseek-ai/DeepSeek-V2.5
- Pro/Qwen/Qwen2.5-7B-Instruct

更多模型请参考硅基流动官方文档。

## 架构设计

### 调度系统

调度系统采用可扩展架构，支持：

- 多供应商注册
- 多模型配置
- 权重负载均衡
- 账号池健康管理与自动故障转移
- 失败账号自动恢复

### 核心组件

#### 1. Provider 接口

```go
type Provider interface {
    Name() string
    ChatCompletion(ctx context.Context, req *ChatCompletionRequest, apiKey string) (*ChatCompletionResponse, error)
    ChatCompletionStream(ctx context.Context, req *ChatCompletionRequest, apiKey string) (<-chan *ChatCompletionStreamResponse, error)
}
```

#### 2. 工厂模式 (Factory)

使用工厂模式进行依赖注入，便于测试和扩展：

```go
// 创建工厂
factory := scheduler.NewFactory()

// 注册 provider 构建器
factory.RegisterProviderBuilder("siliconflow", func(cfg config.ProviderConfig) providers.Provider {
    return providers.NewSiliconFlowProvider(cfg.BaseURL)
})

// 构建调度器
sched, err := factory.BuildScheduler(config.AppConfig)
```

#### 3. 重试机制

内置指数退避重试机制：

```go
// 使用默认配置重试
resp, err := sched.ChatCompletionWithRetry(ctx, req, scheduler.DefaultRetryConfig)

// 自定义重试配置
cfg := scheduler.RetryConfig{
    MaxRetries:     5,
    InitialBackoff: 200 * time.Millisecond,
    MaxBackoff:     5 * time.Second,
}
resp, err := sched.ChatCompletionWithRetry(ctx, req, cfg)
```

#### 4. 运行时统计

获取调度器运行时统计：

```go
stats := sched.GetStats()
log.Printf("Total: %d, Success: %d, Failed: %d, Retries: %d",
    stats.TotalRequests, stats.SuccessCount, stats.FailedCount, stats.RetryCount)
```

### 扩展新供应商

1. 实现 `Provider` 接口
2. 在工厂中注册 provider 构建器
3. 在 `config.yaml` 中配置模型

示例：

```go
// 1. 实现 Provider 接口
type MyProvider struct {
    baseURL    string
    httpClient providers.HTTPDoer
}

func (p *MyProvider) Name() string { return "myprovider" }
func (p *MyProvider) ChatCompletion(ctx context.Context, req *models.ChatCompletionRequest, apiKey string) (*models.ChatCompletionResponse, error) {
    // 实现逻辑...
}
func (p *MyProvider) ChatCompletionStream(ctx context.Context, req *models.ChatCompletionRequest, apiKey string) (<-chan *models.ChatCompletionStreamResponse, error) {
    // 实现逻辑...
}

// 2. 注册到工厂
factory.RegisterProviderBuilder("myprovider", func(cfg config.ProviderConfig) providers.Provider {
    return NewMyProvider(cfg.BaseURL)
})

// 3. 在 config.yaml 中添加
providers:
  myprovider:
    base_url: https://api.myprovider.com/v1
    accounts:
      - id: default
        api_key: your-key
        weight: 1
        enabled: true
    models:
      - my-model-v1
```

## 部署

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o llm-gateway

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/llm-gateway .
COPY config.yaml .
EXPOSE 8080
CMD ["./llm-gateway"]
```

### Kubernetes

可以部署到 Kubernetes 集群，支持水平扩展。

## 安全注意事项

1. API 密钥安全：请妥善保管 `config.yaml` 文件，不要提交到版本控制
2. 生产环境建议使用 HTTPS
3. 建议配置严格的 CORS 白名单
4. 建议添加限流和配额管理

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
