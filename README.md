# LLM Gateway - 大模型接口调度网关

一个完全兼容 OpenAI API 规范的大模型接口调度网关，支持硅基流动等推理平台。

## 功能特性

- ✅ 完全兼容 OpenAI API 规范
- ✅ 支持硅基流动推理平台
- ✅ 支持流式响应 (Streaming Response)
- ✅ 可扩展的调度架构
- ✅ 多供应商、多模型、多账号支持
- ✅ 完整的错误处理与日志记录
- ✅ OpenAI 官方 SDK 直接调用支持

## 项目结构

```
my-llm-api/
├── config/          # 配置管理
├── handlers/        # HTTP 处理器
├── middleware/      # 中间件
├── models/          # 数据模型
├── providers/       # 供应商适配器
├── scheduler/       # 调度系统
├── main.go          # 主程序入口
├── .env             # 环境变量配置
└── go.mod           # Go 模块定义
```

## 快速开始

### 1. 环境要求

- Go 1.21 或更高版本
- 硅基流动 API 密钥

### 2. 安装依赖

```bash
# 使用国内代理加速
$env:GOPROXY="https://goproxy.cn,direct"
go mod tidy
```

### 3. 配置

复制 `.env.example` 为 `.env` 并配置：

```env
SILICONFLOW_API_KEY=your-api-key-here
SILICONFLOW_BASE_URL=https://api.siliconflow.cn/v1
SERVER_PORT=8080
LOG_LEVEL=info
```

### 4. 构建

```bash
go build -o llm-gateway.exe
```

### 5. 运行

```bash
./llm-gateway.exe
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
    api_key="any-key",  # 网关不验证 API Key
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
./llm-gateway.exe

# 在另一个终端运行测试
go test -run TestIntegration -v
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
- 简单的轮询负载均衡
- 故障转移机制

### Provider 接口

```go
type Provider interface {
    Name() string
    ChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error)
    ChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (<-chan *ChatCompletionStreamResponse, error)
}
```

### 扩展新供应商

1. 实现 `Provider` 接口
2. 在调度器中注册供应商
3. 配置模型映射

## 部署

### Docker (可选)

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o llm-gateway

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/llm-gateway .
COPY .env .
EXPOSE 8080
CMD ["./llm-gateway"]
```

### Kubernetes

可以部署到 Kubernetes 集群，支持水平扩展。

## 注意事项

1. API 密钥安全：请妥善保管 `.env` 文件，不要提交到版本控制
2. 生产环境建议使用 HTTPS
3. 建议添加认证中间件
4. 建议添加限流和配额管理

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
