# omni-agent-kit

面向企业 AI 工具的 Go 公共基础库。它提供跨项目稳定的错误、流和 Tool
协议，并提供 Eino、Hertz 的薄适配层。

当前阶段：v0 骨架。API 在第一个真实产品 omni-agent-knowledge 接入验证前，
仍可能进行小幅调整。

## 为什么有这个项目

omni-agent-knowledge、omni-agent-nl2sql、omni-agent-bi 等产品需要复用一些
基础协议，但它们不应该共享一套臃肿的“万能 Agent”业务模型。

这个仓库只抽取已经明确稳定的边界：

- 错误如何安全地跨传输层返回；
- 流式结果如何被框架和 HTTP 层共同消费；
- AI Tool 如何脱离具体 Agent 框架定义；
- Eino 和 Hertz 如何接入上述协议。

模型、Retriever、Graph、Workflow 和具体产品的应用接口继续使用 Eino 或由
产品仓库自行定义。

## 包结构

| 包 | 作用 | 外部依赖 |
|---|---|---|
| fault | 稳定错误码、安全消息和 cause 包装 | 无 |
| stream | 泛型拉式流、EOF 和幂等关闭 | 无 |
| tool | 基于 JSON Schema/JSON 的 Tool 契约 | 无 |
| adapter/eino/tool | 将 kit Tool 适配为 Eino InvokableTool | Eino |
| adapter/eino/stream | 将 Eino StreamReader 适配为 kit Reader | Eino |
| transport/hertz/problem | 输出 RFC 9457 风格的安全错误 | Hertz |
| transport/hertz/sse | 将 kit Reader 顺序写入 Hertz SSE | Hertz |

依赖方向保持单向：

~~~text
产品应用
  ├── Eino 编排 ──> adapter/eino ──> tool / stream
  └── Hertz API  ──> transport/hertz ──> fault / stream
~~~

核心的 fault、stream、tool 只依赖 Go 标准库。

## 安装

~~~bash
go get github.com/qiuyier/omni-agent-kit
~~~

当前模块固定使用：

- Go 1.23+
- Eino v0.9.13
- Hertz v0.10.5

## 定义并适配一个 Eino Tool

~~~go
source, err := kittool.New(kittool.Definition{
    Name:        "lookup_order",
    Description: "Look up an order by identifier.",
    InputSchema: json.RawMessage(`{
        "type": "object",
        "properties": {
            "order_id": {"type": "string"}
        },
        "required": ["order_id"]
    }`),
}, func(ctx context.Context, arguments json.RawMessage) (json.RawMessage, error) {
    // 调用真实订单系统；凭证由运行时注入，不能放入 Definition。
    return json.RawMessage(`{"status":"paid"}`), nil
})
if err != nil {
    return err
}

einoTool, err := einotool.Wrap(source)
~~~

完整可运行代码见 examples/eino-tool。

## 在 Hertz 中输出 SSE

应用服务应先创建流；创建失败时输出普通 HTTP Problem。进入 sse.Write 后必须
保持 handler 存活，不能把 Hertz RequestContext 或 SSE Writer 交给后台协程。

~~~go
reader, err := service.StreamAnswer(ctx, request)
if err != nil {
    _ = problem.Write(c, err)
    return
}

started, err := sse.Write(ctx, c, reader, func(
    ctx context.Context,
    event AnswerEvent,
) (sse.Event, error) {
    data, err := json.Marshal(event)
    return sse.Event{Type: "answer.delta", Data: data}, err
})
if err != nil && !started {
    _ = problem.Write(c, err)
}
~~~

流开始后再发生错误，适配器会尽可能发送：

~~~text
event: error
data: {"code":"unavailable","message":"model unavailable"}
~~~

内部 cause、模型供应商响应、连接串和密钥不会进入该事件。

生产环境的 Hertz Server 应启用客户端断连感知：

~~~go
h := server.Default(server.WithSenseClientDisconnection(true))
~~~

完整示例见 examples/hertz-sse。

## 错误边界

fault.Error 有两类信息：

- safe message：可以返回客户端；
- cause：只供日志、errors.Is 和 errors.As 使用。

未知错误统一对外显示 internal error。HTTP 状态映射只存在于
transport/hertz/problem，核心错误不携带 HTTP 状态码。

## v0 明确不做

- 不再封装一套 ChatModel、Retriever、Embedding 或 Graph；
- 不提供通用 Agent.Runner；
- 不提供模型供应商注册中心和 API Key 管理；
- 不实现动态 .so 插件；
- 不在 kit 中建立通用 Connector SPI；
- 不实现产品级租户、RBAC、Prompt 管理和工作流；
- 不在公共库中放 Thrift IDL。

这些能力应先在真实产品中落地。至少两个产品出现完全一致的需求后，再提取到
kit，避免公共 API 被假设驱动。

## 本地验证

~~~bash
go test ./...
go test -race ./...
go vet ./...
~~~

更详细的边界与决策见 docs/architecture.md。
