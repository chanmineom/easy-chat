# Easy Chat

基于 Go、go-zero、Kafka 和 WebSocket 构建的分布式即时通讯系统，覆盖认证、好友关系、群组管理、会话、消息持久化、实时推送、离线缓存和多节点消息路由。

项目重点不是实现一个简单的 WebSocket Demo，而是处理 IM 系统中的消息可靠性问题：

- 消息顺序号与幂等发送。
- MongoDB 消息持久化与事务性 Outbox。
- WebSocket 节点注册、发现和跨节点推送。
- 一次性 Ticket、心跳检测和断线重连。
- 送达/已读游标。
- Kafka 消费重试与死信队列。
- 前端 IndexedDB 离线缓存和离线发送队列。

> 当前消息类型为文本消息。项目定位是可运行的分布式 IM 工程实践，不包含音视频通话、富媒体消息和移动端客户端。

---

## 技术栈

### 后端

| 分类 | 技术 |
| --- | --- |
| 语言 | Go 1.26 |
| 微服务框架 | go-zero |
| HTTP | go-zero REST |
| 内部通信 | gRPC + Protobuf |
| 关系数据库 | MySQL 8 |
| 文档数据库 | MongoDB 7+，要求 Replica Set |
| 缓存与临时状态 | Redis |
| 消息队列 | Kafka |
| 服务发现 | etcd |
| 网关 | Kong |
| 实时通信 | WebSocket / Gorilla WebSocket |
| 部署 | Docker Compose |

### 前端

| 分类 | 技术 |
| --- | --- |
| UI 框架 | React 19 |
| 构建工具 | Vite |
| 组件库 | Material UI |
| 路由 | React Router |
| 状态管理 | Zustand |
| 本地存储 | IndexedDB / idb |
| 实时通信 | WebSocket |

---

## 系统架构

```mermaid
flowchart LR
    Browser[React Web Client]

    Browser -->|HTTP / WebSocket| Kong[Kong API Gateway]

    Kong --> UserAPI[user.api]
    Kong --> SocialAPI[social.api]
    Kong --> IMAPI[im.api]
    Kong -->|WebSocket| IMWSA[im.ws node A]
    Kong -->|WebSocket| IMWSB[im.ws node B]

    UserAPI --> UserRPC[user.rpc]
    SocialAPI --> SocialRPC[social.rpc]
    IMAPI --> IMRPC[im.rpc]

    UserRPC --> MySQL[(MySQL)]
    SocialRPC --> MySQL
    SocialRPC -->|Social Outbox| Kafka[(Kafka)]
    IMRPC --> Mongo[(MongoDB)]

    IMWSA --> Redis[(Redis)]
    IMWSB --> Redis
    IMWSA --> Etcd[(etcd)]
    IMWSB --> Etcd

    IMWSA -->|Chat Receipt Commands| Kafka
    IMWSB -->|Chat Receipt Commands| Kafka

    Kafka --> TaskMQ[task.mq]
    TaskMQ --> Mongo
    TaskMQ -->|Internal gRPC| IMWSA
    TaskMQ -->|Internal gRPC| IMWSB
```

### 数据职责

| 存储 | 数据 |
| --- | --- |
| MySQL | 用户、好友关系、好友申请、群组、群成员、Social Outbox |
| MongoDB | 会话、会话成员、消息、IM Outbox |
| Redis | WebSocket 会话、一次性连接 Ticket、内部 RPC 鉴权信息 |
| Kafka | 聊天命令、消息创建事件、回执命令、死信 |
| etcd | WebSocket 节点注册与服务发现 |

---

## 核心功能

### 用户与认证

- 手机号和密码注册、登录。
- bcrypt 密码哈希。
- JWT 身份认证。
- 内部 gRPC App/Token 鉴权。
- 批量用户资料查询。

### 好友关系

- 通过用户 ID 发送好友申请。
- 查看收到的申请。
- 接受或拒绝申请。
- 查看好友列表。
- 删除好友。

### 群组

- 创建群组。
- 查看群组列表、详情和群成员。
- 添加成员、移除成员和主动退群。
- 群主设置或取消管理员。
- 转让群主。
- 解散群组。

### 实时消息

- 单聊和群聊。
- 会话级单调递增消息顺序号。
- 最近消息和更早消息分页。
- 未读数、已读游标和送达游标。
- WebSocket 实时发送与接收。
- 消息提交、业务拒绝和持久化结果回执。
- 群消息发送人昵称展示。

### 可靠性

- `client_message_id` 幂等去重。
- MongoDB 事务保证顺序号、消息和 Outbox 原子写入。
- Kafka 消息消费重试。
- 非法或持续失败消息写入死信 Topic。
- WebSocket 一次性 Ticket。
- Ping/Pong 心跳与断线检测。
- 指数退避重连。
- WebSocket 会话恢复。
- 前端 IndexedDB 消息缓存。
- 断线期间消息增量同步。
- 离线发送队列与相同 `client_message_id` 重试。

### 网关

- HTTP 与 WebSocket 统一入口。
- JWT 透传。
- CORS。
- 请求体大小限制。
- 登录、注册和各业务接口限流。
- Kong Admin API 仅监听本机。

---

## 服务与端口

### 对外端口

| 端口 | 服务 | 说明 |
| --- | --- | --- |
| `8000` | Kong Proxy | 前端 HTTP 与 WebSocket 统一入口 |
| `8001` | Kong Admin | 仅绑定 `127.0.0.1` |
| `5173` | Vite | 前端开发服务器 |

### 内部服务

| 服务 | 默认端口 | 说明 |
| --- | --- | --- |
| `user.api` | `8888` | 用户 HTTP API |
| `social.api` | `8889` | 好友和群组 HTTP API |
| `im.api` | `8890` | 会话、消息和 Ticket HTTP API |
| `user.rpc` | `8083` | 用户 gRPC |
| `social.rpc` | `8081` | 好友和群组 gRPC |
| `im.rpc` | `8080` | 会话和消息 gRPC |
| `im.ws` | `10090` | WebSocket 接入 |
| `im.ws.rpc` | `10091` | 跨节点推送 gRPC |
| `task.mq` | `9400` | Kafka 消费服务 |

Prometheus 指标端口：

| 服务 | 端口 |
| --- | --- |
| `user.api` | `9201` |
| `social.api` | `9203` |
| `im.api` | `9200` |
| `user.rpc` | `9303` |
| `social.rpc` | `9301` |
| `im.rpc` | `9300` |
| `im.ws` | `9100` |
| `task.mq` | `9400` |

---

## 消息链路

```mermaid
sequenceDiagram
    participant C as Client
    participant K as Kong
    participant API as im.api
    participant WS as im.ws
    participant Q as Kafka
    participant MQ as task.mq
    participant DB as MongoDB

    C->>K: POST /v1/im/ws/ticket
    K->>API: IssueWSTicket
    API-->>C: One-time Ticket

    C->>K: WebSocket /ws?ticket=...
    K->>WS: Upgrade
    WS-->>C: connection.ready

    C->>WS: message.send
    WS->>Q: Publish chat command
    WS-->>C: message.accepted

    Q->>MQ: Consume chat command
    MQ->>DB: Append message transactionally
    MQ->>DB: Write message Outbox
    DB->>Q: Publish im.message.created

    Q->>MQ: Consume message.created
    MQ->>WS: PushToUsers
    WS-->>C: message.created

    C->>WS: message.delivered / message.read
    WS->>Q: Publish receipt command
    Q->>MQ: Consume receipt command
    MQ->>DB: Advance deliveredSeq / readSeq
```

### WebSocket 消息格式

```json
{
  "type": "message.send",
  "request_id": "request-id",
  "data": {
    "conversation_id": "dc_xxx",
    "client_message_id": "client-message-id",
    "content_type": "text",
    "content": "hello"
  }
}
```

服务端支持的主要事件：

| 事件 | 方向 | 说明 |
| --- | --- | --- |
| `connection.ready` | Server → Client | 连接建立成功 |
| `message.send` | Client → Server | 发送消息 |
| `message.accepted` | Server → Client | 命令已进入 Kafka |
| `message.rejected` | Server → Client | 消息被业务拒绝 |
| `message.created` | Server → Client | 消息已持久化并广播 |
| `message.delivered` | Client → Server | 送达回执 |
| `message.read` | Client → Server | 已读回执 |
| `receipt.accepted` | Server → Client | 回执命令已接收 |
| `ping` / `pong` | 双向 | 连接心跳 |

---

## 一致性设计

### 1. 消息幂等

客户端发送消息时生成：

```text
client_message_id
```

服务端以：

```text
sender_id + client_message_id
```

判断重复消息。网络重试、断线重发和离线队列重放都使用同一个 ID，因此不会生成重复消息。

### 2. 会话顺序号

消息写入时通过 MongoDB 原子更新递增：

```text
conversations.lastSeq
```

消息、顺序号和 Outbox 事件在同一个事务中写入。

### 3. Outbox

业务数据提交时同步写入 Outbox，后台 Worker 再发布 Kafka 事件，避免：

```text
数据库写入成功，但 Kafka 发布失败
```

### 4. 跨节点推送

WebSocket 节点将自身地址注册到 etcd。消费服务根据 Redis 中的用户 Session 找到目标节点，再通过内部 gRPC 执行跨节点推送。

### 5. 前端可靠性

- IndexedDB 保存正式消息和同步游标。
- 消息按 `message_id` 或 `client_message_id` 去重。
- 断线时消息写入待发送队列。
- 重连后重新申请 Ticket 并重发。
- WebSocket 使用 ping/pong 检测假连接。

---

## 目录结构

```text
.
├── apps
│   ├── im
│   │   ├── api          # 会话、消息、WebSocket Ticket HTTP API
│   │   ├── model        # 会话、消息、命令、回执模型
│   │   ├── outbox       # IM Outbox
│   │   ├── repository   # MongoDB 仓储
│   │   ├── rpc          # 会话和消息 gRPC
│   │   ├── session      # Redis WebSocket Session
│   │   ├── ticket       # 一次性 Ticket
│   │   └── ws           # WebSocket 网关与跨节点推送
│   ├── social
│   │   ├── api          # 好友和群组 HTTP API
│   │   ├── model        # 好友和群组事件
│   │   ├── outbox       # Social Outbox
│   │   ├── repository   # MySQL 仓储
│   │   └── rpc          # 好友和群组 gRPC
│   ├── task
│   │   └── mq           # Kafka 消费者、路由、重试和死信
│   └── user
│       ├── api          # 用户 HTTP API
│       ├── auth         # JWT
│       ├── repository   # MySQL 仓储
│       └── rpc          # 用户 gRPC
├── deploy
│   ├── docker           # 各服务 Dockerfile
│   ├── env              # 环境变量模板
│   ├── kong             # Kong 声明式配置
│   └── docker-compose.yaml
├── scripts              # 构建和本地启动脚本
└── web
    ├── src/api          # HTTP API
    ├── src/components   # 联系人、群组、消息面板
    ├── src/layouts      # 认证与聊天布局
    ├── src/realtime     # WebSocket 客户端
    ├── src/storage      # IndexedDB
    └── src/stores       # Zustand
```

---

## 快速启动

### 环境要求

- Go 1.26+
- Node.js 20+
- Docker Engine / Docker Desktop
- MySQL 8
- MongoDB Replica Set
- Redis 7
- Kafka 3.x
- etcd 3.x

> 项目当前不负责启动外部基础设施。MySQL、MongoDB、Redis、Kafka 和 etcd 可以是本机服务、独立虚拟机或云服务。

MongoDB 必须使用 Replica Set，因为消息写入和会话管理依赖事务。

### 1. 配置环境变量

```bash
cd deploy
cp env/.env.example env/.env
```

编辑 `deploy/env/.env`：

```dotenv
MYSQL_DSN='user:password@tcp(host:3306)/easy-chat?charset=utf8mb4&parseTime=true&loc=UTC'
MONGO_URL='mongodb://user:password@host:27017/easy-chat?authSource=admin&replicaSet=rs0'
REDIS_HOST=host:6379
KAFKA_BROKER=host:9092
ETCD_HOST=host:2379
```

包含 `&` 的 DSN 必须保留单引号。

本地直接运行 Go 服务时加载环境变量：

```bash
set -a
source deploy/env/.env
set +a
```

Docker Compose 会通过 `env_file` 自动加载该文件。

### 2. 初始化 MySQL

```bash
mysql -h 127.0.0.1 -u user -p easy-chat \
  < apps/user/migrations/001_users.sql

mysql -h 127.0.0.1 -u user -p easy-chat \
  < apps/social/migrations/001_group_core.sql

mysql -h 127.0.0.1 -u user -p easy-chat \
  < apps/social/migrations/002_friends.sql
```

### 3. 准备 Kafka Topic

```text
im.chat.commands
im.message.created
im.receipt.commands
im.dead.letter
```

### 4. Docker Compose 启动

```bash
cd deploy
docker compose up -d --build
```

查看状态：

```bash
docker compose ps
```

查看日志：

```bash
docker compose logs -f im-api
docker compose logs -f im-ws
docker compose logs -f task-mq
```

### 5. 启动前端

```bash
cd web
npm ci
npm run dev
```

访问：

```text
http://127.0.0.1:5173
```

Vite 将 `/v1` 和 `/ws` 代理到：

```text
http://127.0.0.1:8000
ws://127.0.0.1:8000
```

如果 Kong 运行在其他主机，需要修改 `web/vite.config.js` 中的代理目标。

### 本地直接运行 Go 服务

```bash
set -a
source deploy/env/.env
set +a

go run ./apps/user/rpc -f apps/user/rpc/etc/user.yaml
go run ./apps/user/api -f apps/user/api/etc/user-api.yaml

go run ./apps/social/rpc -f apps/social/rpc/etc/social.yaml
go run ./apps/social/api -f apps/social/api/etc/social-api.yaml

go run ./apps/im/rpc -f apps/im/rpc/etc/im.yaml
go run ./apps/im/api -f apps/im/api/etc/im.yaml

go run ./apps/im/ws -f apps/im/ws/etc/im.yaml
go run ./apps/task/mq -f apps/task/mq/etc/task.yaml
```

---

## API 概览

所有业务 HTTP 请求统一经过 Kong：

```text
http://127.0.0.1:8000
```

统一响应：

```json
{
  "code": 200,
  "msg": "",
  "data": {}
}
```

### 用户

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/v1/user/register` | 注册 |
| POST | `/v1/user/login` | 登录 |
| GET | `/v1/user/detail` | 当前用户 |
| GET | `/v1/user/batch?user_ids=...` | 批量用户资料 |

### 好友

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/v1/social/friend/requests` | 发送申请 |
| GET | `/v1/social/friend/requests/incoming` | 收到申请 |
| PUT | `/v1/social/friend/requests` | 接受或拒绝 |
| GET | `/v1/social/friends` | 好友列表 |
| DELETE | `/v1/social/friends/:friendId` | 删除好友 |

### 群组

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/v1/social/groups` | 创建群 |
| GET | `/v1/social/groups` | 群列表 |
| GET | `/v1/social/groups/:groupId` | 群详情 |
| GET | `/v1/social/groups/:groupId/members` | 群成员 |
| POST | `/v1/social/groups/:groupId/members` | 添加成员 |
| DELETE | `/v1/social/groups/:groupId/members/:userId` | 移除成员 |
| POST | `/v1/social/groups/:groupId/leave` | 退出群 |
| PUT | `/v1/social/groups/:groupId/members/:userId/role` | 修改角色 |
| PUT | `/v1/social/groups/:groupId/owner` | 转让群主 |
| DELETE | `/v1/social/groups/:groupId` | 解散群 |

### 会话与消息

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/v1/im/conversations` | 会话列表 |
| GET | `/v1/im/messages?after_seq=...` | 向后同步消息 |
| GET | `/v1/im/messages?before_seq=...` | 加载更早消息 |
| POST | `/v1/im/ws/ticket` | 申请一次性 WebSocket Ticket |
| WS | `/ws?ticket=...` | WebSocket 连接 |

---

## 构建与检查

### 后端

```bash
go build ./apps/...
```

需要 MongoDB 的测试：

```bash
MONGO_TEST_URL="$MONGO_URL" go test ./apps/im/...
```

### 前端

```bash
cd web
npm run build
npm run lint
```

---

## 工程亮点

- 使用 go-zero 同时管理 REST、gRPC 和 WebSocket 服务，保持接口定义与生成代码职责清晰。
- 使用 MongoDB 事务实现消息顺序号、消息记录与 Outbox 的原子写入。
- 使用 Kafka 解耦消息接收、持久化和在线推送。
- 使用 `client_message_id` 实现跨网络重试的幂等消息。
- 使用 Redis 和 etcd 完成 WebSocket 会话定位、节点注册和跨节点推送。
- 使用一次性 Ticket 避免长期 JWT 暴露在 WebSocket URL 中。
- 前端通过 IndexedDB 保存消息游标、缓存和待发送队列，支持断线恢复。
- 使用 Kong 统一处理 HTTP/WS 路由、CORS、请求大小限制和入口限流。
- 提供 Kafka 死信队列，避免异常消息无限阻塞主消费链路。

---

## 当前边界与后续规划

以下能力尚未实现，README 不将其视为现有功能：

- 从联系人主动创建无历史消息的单聊。
- 手机号模糊搜索用户。
- 修改用户昵称、头像和密码。
- 修改群名称、群头像和群公告。
- 图片、文件、语音和视频消息。
- 消息撤回、编辑、转发和删除。
- 消息全文搜索。
- 对方已送达/已读状态的实时回显。
- 正在输入、用户在线状态和最后在线时间。
- 会话置顶、免打扰、隐藏和清空。
- 浏览器桌面通知和移动端推送。
- Refresh Token、登录设备管理和强制下线。
- 移动端响应式布局与原生客户端。

