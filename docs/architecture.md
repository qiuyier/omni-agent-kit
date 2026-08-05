# omni-agent-kit v0 架构决策

## 目标

为多个独立的 omni-agent-* 产品提供少量、稳定、可组合的 Go 基础协议，同时
避免把某个产品的领域模型或某个框架的快速变化扩散到所有仓库。

## 分层

~~~mermaid
flowchart LR
    P["产品应用层"] --> E["Eino 编排"]
    P --> H["Hertz 传输层"]
    E --> EA["adapter/eino"]
    H --> HT["transport/hertz"]
    EA --> T["tool"]
    EA --> S["stream"]
    HT --> S
    HT --> F["fault"]
~~~

约束：

1. fault、stream、tool 只依赖标准库。
2. adapter/eino 不依赖 Hertz。
3. transport/hertz 不依赖 Eino。
4. 产品负责组装框架、传输层和领域服务。
5. 框架类型不能进入产品对外 API。

## 为什么不抽象 ChatModel

Eino 已经提供 ChatModel、ToolCallingChatModel、Retriever、Indexer、Graph、
Workflow 和 ADK。kit 再复制一层接口会产生三个问题：

1. Eino 新能力需要在 kit 中重复映射；
2. 最小公分母接口会丢失流式、多模态或 Tool Calling 能力；
3. 模型配置与凭证容易进入公共协议。

因此，具体产品直接构造 Eino 组件并注入应用服务。kit 只在需要跨边界时进行
薄适配。

## Tool 契约

Tool 使用 JSON Schema 描述输入/输出，并使用 JSON RawMessage 调用：

~~~go
type Tool interface {
    Definition(ctx context.Context) (Definition, error)
    Invoke(ctx context.Context, arguments json.RawMessage) (json.RawMessage, error)
}
~~~

约定：

- error 表示执行失败或基础设施失败；
- 模型可以理解并继续处理的业务结果，应返回结构化 JSON；
- Definition 不得包含密码、Token、连接串等运行时秘密；
- v0 只检查 JSON 边界，不负责完整 JSON Schema 数据校验；
- Eino Tool options 不自动穿透到 kit Tool。

连接器未来可以根据安全配置创建一个或多个具体 Tool，但管理面的连接测试、
凭证保存和运行面的 Tool 执行不应混在同一个万能接口中。

## Stream 契约

~~~go
type Reader[T any] interface {
    Recv() (T, error)
    Close() error
}
~~~

使用拉式流的原因：

- Hertz 按客户端写入速度读取，天然形成背压；
- 可以直接适配 Eino StreamReader；
- 不要求无界 channel；
- 资源所有权清晰。

正常结束必须返回 io.EOF。消费者必须 Close；实现必须允许重复 Close。创建
Reader 的上层必须把 context 传给生产者，使取消或客户端断连能够解除阻塞的
Recv。

Hertz SSE 适配器在 handler 内同步消费 Reader，不会在 handler 返回后继续
写 RequestContext。

## 流式错误状态机

~~~text
创建应用流失败
  -> HTTP Problem

首个事件读取/编码失败，SSE 尚未建立
  -> HTTP Problem

至少一个 SSE 事件已经写出后失败
  -> event: error
  -> 关闭 Reader 和 SSE Writer
  -> 不再尝试修改 HTTP 状态
~~~

Hertz v0.10.5 的官方 SSE Writer 自带 chunked writer、Content-Type、no-cache
和每事件 Flush；kit 不重复实现 SSE 帧或额外 Flush。

## 错误与安全

fault 将错误分类和传输协议解耦。只有 safe message 能进入外部响应，cause
只用于内部诊断。

安全约束：

- 未分类错误对外统一为 internal；
- 不默认记录 Prompt、文档、SQL 结果或 Tool 参数；
- tenant_id 不能由请求体直接决定；
- 身份必须由产品鉴权层验证后通过 context 传递；
- SSE 开始后禁止调用 Problem Writer；
- Tool Definition 不能携带运行时秘密。

## Hertz 与 Thrift

kit 是库，不是服务，因此 v0 不包含 Thrift IDL。

具体产品可以使用 hz + Thrift 生成 Hertz 路由，也可以使用 OpenAPI。生成类型
必须留在产品 transport 层并映射到领域请求。内部 RPC 若采用 Kitex/Thrift，
同样不能让生成类型渗透进 tool、stream 或 fault。

## 版本策略

v0 采用单 Go module，并固定稳定依赖版本，不使用 latest 或预发布版本。只有
当依赖体积或发布节奏产生真实问题时，才考虑拆分 Eino/Hertz 子模块。

公共能力的提取规则：

1. 先在具体产品落地；
2. 至少两个产品出现相同语义和生命周期；
3. API 能用具体测试描述；
4. 提取后不要求产品依赖另一个产品的领域类型。
