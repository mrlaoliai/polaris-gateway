# 🛰️ Polaris Gateway v2.0

**自治式 AI 协议编排网关**

**Polaris Gateway v2.0 是一套高性能、高可用的 AI 基础设施，旨在通过语义理解、自愈能力及跨厂商协议深度对齐，为 Claude Code、Aider 等下一代自治 Agent 提供完美的底层支持 **^^。

## 📑 项目核心哲学 (Axioms)

* **Zero-CGO** **: 采用纯 Go 实现，拒绝 CGO 依赖，确保极致的可移植性与系统稳定性 **^^。
* **State-in-DB** **: 系统状态、配置与密钥完全内聚于 SQLite (WAL 模式)，支持高并发下的审计与自愈 **^^。
* **Zero-Poetry** **: 严格遵循技术确定性，剔除所有 AI 人格化声明与冗余输出 **^^。
* **Protocol-Agnostic** **: 厂商协议无关性，通过动态 DSL 引擎实时适配各种异构 API **^^。

## ✨ 核心特性

* **物理执行器深度解耦 (Physical Executor Decoupling)** **: 厂商逻辑隔离：将 Google AI Studio 与 Google Cloud Vertex AI 彻底拆分为独立 Provider。支持不同维度的认证流（API Key 注入 vs. OAuth2 握手）。
* **自适应动作映射** **: 网关自动根据客户端 stream 参数，在后端实时切换 :streamGenerateContent 与 :generateContent，实现配置层面的“零感知”切换。

* **Bifrost 2.0 语义流引擎** :
* **MCP 原生桥接** **: 自动执行 Model Context Protocol 工具调用与目标协议的降级转换 **^^。
* **思维链签名 (Thinking Signature)** **: 为异构模型生成的推理块注入“影子签名”，维持客户端（如 Claude Code）的推理连续性 **^^。
* **SSE 心跳注入 (Heartbeat Injection)** **: 在模型长时思考期主动注入心跳事件，防止连接超时断开 **^^。
* **智能调度与流控** :
* **Sentinel 自动拨测** **: 后台实时维护物理密钥池健康度 **^^。
* **语义事务序列化** **: 确保并发工具调用的输出流符合因果逻辑，避免语义竞态 **^^。
* **动态 DSL 转换** **: 基于 CEL-go，支持在不重启网关的情况下热更新协议映射规则 **^^。
* **高性能存储** :
* **L2 磁盘溢出** **: 当推理块或上下文累积超过阈值时，自动由内存溢出至磁盘缓存，防止 OOM **^^。

* **异步状态机与存储** • DB-Writer 中台：采用单向 Channel 串行化所有数据库写操作，完美解决 SQLite 在超高并发下的锁竞争（Database Locked）问题。• L2 Spiller (磁盘溢出)：当推理上下文超过内存阈值时，自动溢出至 VFS (虚拟文件系统)，确保系统在极端压力下不会发生 OOM。

## 🏗️ 系统架构

**系统采用四层逻辑模型设计 **^^：

1. **入口平面 (Entry & Control)** **: 包含基于 VFS 的 Web Dashboard 及 Guardian 多租户配额中心 **^^。
2. **协议平面 (Bifrost 2.0)** **: 负责语义转换、MCP 路由及心跳管理 **^^。
3. **调度平面 (Orchestrator)** **: 实现物理映射、健康检测及多模态回退策略 **^^。
4. **执行平面 (Physical Providers)** **: 纯 Go 实现的 OpenAI、Vertex AI、Anthropic、DeepSeek 等执行器 **^^。



## 📂 目录结构 (Go 规范)

**Plaintext**

```
polaris-gateway/  
├── main.go            # 初始化与服务启动
├── internal/  
│   ├── database/      # DAO 层与 L2 溢出管理
│   ├── state/         # Session Buffer 与逻辑锁管理
│   ├── dashboard/     # Web 逻辑与 VFS 静态资源
│   ├── orchestrator/  # 调度引擎：Sentinel 与智能路由
│   └── bridge/        # Bifrost 2.0：Transformer、DSL、Heartbeat 核心实现
├── pkg/  
│   ├── provider/      # 物理厂商（OpenAI/Vertex/Claude 等）执行器
│   └── middleware/    # 认证、限流与 Response Post-Processor
└── ui/                # 后台管理系统源代码
```

^^

## 🛠️ 数据库设计 (State-in-DB)

**网关状态完全由 SQLite 持久化，核心表包括 **^^：

| **表名**    | **职责描述**                 |
| ----------------- | ---------------------------------- |
| `providers`     | 定义物理厂商及协议基准             |
| `accounts`      | 物理密钥池，由 Sentinel 维护健康度 |
| `routing_rules` | 虚拟模型到物理模型的路由核心映射   |
| `usage_stats`   | 记录完整请求演化路径的审计追踪     |
| `gateway_keys`  | 分发给 Agent 使用的逻辑凭证        |

## ⚖️ 安全与确定性

* **模型指纹保护** **: 针对敏感客户端动态修正响应格式，防止后端被替换后的识别风险 **^^。
* **审计性** **: 所有影响决策的记忆碎片与 Trace Path 均可回溯，满足合规要求 **^^^^^^。

---

 **作者** **: mrlaoliai **^^
 **参考查询** : `gmail: Polaris Gateway v2.0 最终版设计文档 V3`
