> 重要：当前通信仅支持二进制子协议 myflowhub.bin.v1，所有 JSON 路径已移除。Server 二进制路由集中注册于 `server/internal/hub/register.go`（由 main 在启动时完成注入）。

## 日志（审计）

Manager API
- GET/POST /api/logs 获取审计日志，支持筛选：keyword, subjectType, decision, action, startAt, endAt, page, pageSize。
	- 需要权限：log.read 或 admin.manage。

Hub 消息
- log_list：同上，payload 附带 userKey。

# 文档索引

- 顶层 README：项目概览、运行方式与权限说明（README.md）
- 权限系统设计：节点语法、隐式权限、管理员与首启规则、API 约束（权限设计.md）
- Manager API 使用说明：REST 端点与示例（manager/API_USAGE.md）
- Web 前端说明：环境、启动、登录/权限、目录结构（web/README.md）

## 快速开始（PowerShell）

```powershell
# 1) 启动 Server
cd d:\rj\MyFlowHub\server
go run .\cmd\myflowhub

# 2) 启动 Manager
cd d:\rj\MyFlowHub\manager
go run .\cmd\manager

# 3) 启动 Web（开发）
cd d:\rj\MyFlowHub\web
npm install
npm run dev
```

## 认证与权限

- 除 /api/auth/login 外，其余管理 API 需 Authorization: Bearer <token>
- 登录返回 token、user、permissions；前端基于 `admin.manage` 控制管理员页面

## 默认管理员与首启规则

- 仅在“新建数据库或首次创建 users 表”时自动创建 admin，并授予 admin.manage 与 **
- 已有用户表时不创建、不赋权

---

## 二进制协议 v1（固定顺序优先）

目标
- 高性能、低开销、跨语言易实现。
- 固定字段顺序为主，不使用键值对；可选字段用位图；支持透传消息。
- 支持定长数值/浮点、变长字符串/字节、结构体/数组。

传输与字节序
- WebSocket 二进制帧或 TCP；仅二进制，不保留 JSON。
- Little-Endian。

握手与目标约定
- WebSocket 子协议：Sec-WebSocket-Protocol: myflowhub.bin.v1（建议）。
- HubDeviceUID：连接建立与认证成功后，Hub 会在响应中（如 auth/manager_auth）告知自身 DeviceUID；之后将 Header.Target 设为该 UID 即表示“发往 Hub 本地处理”。
- 广播：Header.Target=0 表示广播到所有下级客户端；由 Hub 负责转发。

帧结构
- Header（固定 38B）：
	- TypeID[2]=uint16；Reserved[4]=0；MsgID[8]=uint64；Source[8]=uint64；Target[8]=uint64；Timestamp[8]=int64
- Payload：按消息 Schema 的固定字段顺序编码；可选位图置于 Payload 开头（如使用）。

编码规则
- 定长基本类型（不写长度）：bool=1B；i32/u32=4B；i64/u64=8B；f32=4B；f64=8B（IEEE-754）。
- 变长类型：
	- string/bytes：Len16(u16)+Data；若 Len16=0xFFFF，则再写 Len32(u32)+Data。
	- array：Count(varint/u16)+逐元素编码（定长直写，变长各自带短长度）。
	- struct：LenStruct(u16/u32)+内部固定顺序字段（可一次跳过解析）。
- 可选字段位图：ceiling(N/8) 字节，bit0→第1个可选字段…；为 1 则该字段随后的顺序中出现，为 0 则不写。
- 压缩/校验：大包可按 Flags 开启压缩；如需校验可在 Payload 末尾追加 CRC32/64（默认关闭）。

长度与 Varint 细节
- Len16：无符号 16 位长度（不含自身），上限 65535；当值为 0xFFFF 时表示“扩展长度”，随后紧跟 Len32（u32）。
- Len32：无符号 32 位长度（不含自身）。
- Varint（可用于数组个数等）：7bit 编码，最低 7 位为数据位，最高位为续位标志（1=后续还有字节，0=结束）；小于 128 的数仅 1 字节。

位图示例
- 某消息有 6 个可选字段 C..H，则位图占 1 字节（ceil(6/8)=1）。
- 若仅 C、F 出现，位图=0b00100001（bit0=C，bit5=F）。
- 解码顺序：读位图→读所有必选→按 C..H 顺序，对置位字段依其类型读值，未置位则跳过。

TypeID 分配（节选）
- 0 OK_RESP：success:bool(=1)
- 1 ERROR_RESP：success:bool(=0)，error:string，可选 original_id:u64
- 10 MSG_SEND（条件透传）：
	- 当 Target ≠ Hub 时：透传 Opaque bytes（Len16/Len32+Data），Hub 不解析，仅路由。
	- 当 Target = Hub 时：非透传，由 Hub 端对应处理器解析负载（负载布局由具体处理器定义，可为固定顺序字段或内部自定义结构）。
- 20 QUERY_NODES_REQ：可选 user_key:string
- 21 CREATE_DEVICE_REQ：device:Device；可选 user_key:string
- 22 UPDATE_DEVICE_REQ：device:Device；可选 user_key:string
- 23 DELETE_DEVICE_REQ：id:u64；可选 user_key:string
- 110 USER_LOGIN_REQ：username:string，password:string
- 111 USER_LOGIN_RESP
- 112 USER_ME_REQ
- 113 USER_ME_RESP
- 114 USER_LOGOUT_REQ
- 115 USER_LOGOUT_RESP
- 130 PARENT_AUTH_REQ：version:u8, ts:i64(ms), nonce:16B, hardware_id:len16+str, caps:len16+str, sig:32B(HMAC)
- 131 PARENT_AUTH_RESP：request_id:u64, device_uid:u64, session_id:16B, heartbeat_sec:u16, perms:[len16+str], exp:i64, sig:32B
- 150 SYSTEMLOG_LIST_REQ：level:string，source:string，keyword:string，start_at:i64，end_at:i64，page:i32，page_size:i32

结构体示例：Device（作为单字段 struct）
- 顺序：ID:u64, DeviceUID:u64, HardwareID:string, Role:string, Name:string,
	ParentID:u64(可选), OwnerUserID:u64(可选), LastSeen:string(可选), CreatedAt:string, UpdatedAt:string
- 编码：LenStruct + 按以上固定顺序逐字段编码；内部可选字段可用内嵌位图。

“透传”消息
- MSG_SEND(10) 仅在 Target ≠ Hub 时透传；若 Target = Hub，需由 Hub 解析负载。
- 其他声明为透传的类型同理：仅在面向非 Hub 目标的路径上透传；面向 Hub 的请求需明确负载 Schema 并由处理器解析。

文件/媒体传输（建议）
- 分片协议（示例 TypeID）：
	- 300 FILE_INIT_REQ：transfer_id:u64, total_size:u64, mime:string, filename:string, file_hash:bytes；可选 thumbnail:bytes
	- 301 FILE_INIT_RESP：ok:bool(=1)；可选 resume_offset:u64
	- 302 FILE_CHUNK：transfer_id:u64, offset:u64, chunk:bytes；可选 crc32:u32
	- 303 FILE_COMPLETE_REQ：transfer_id:u64
	- 304 FILE_COMPLETE_RESP：ok:bool(=1)
	- 305 FILE_CANCEL：transfer_id:u64, reason:string
- 策略：小文件直传；大文件分片（64–256KB），支持断点续传（resume_offset）与整文件哈希校验。
- 视频：点播用 FILE_REF(url+鉴权)；直播建议专用协议（如 WebRTC），或 STREAM_* 走透传小帧。

文件分片状态机（建议）
- 发送端：INIT → (CHUNK × N) → COMPLETE → 结束；任意时刻可 CANCEL。
- 接收端：收到 INIT 校验并回复 INIT_RESP（可返回 resume_offset 断点）；逐片验证偏移/长度/可选 CRC；收到 COMPLETE 后做整文件哈希校验并回复 OK。
- 重传：CHUNK 超时未确认则重发；最大重试次数后失败并 CANCEL。
- 参数：片大小 64–256KB；窗口 1–4；总超时与最大并发可配置。

示例帧（十六进制节选）
- USER_LOGIN_REQ（出现可选 max_uses=10，expires_at 缺省）：
	- 位图(1B)=0x02；username="alice"→Len16=0x0005+数据；password="p@ss"→Len16=0x0004+数据；max_uses=0x0A000000（i32 LE）。
	- 负载（不含 Header）≈ 01 05 00 61 6c 69 63 65 04 00 70 40 73 73 0a 00 00 00
- MSG_SEND 透传（Target≠Hub）：
	- Opaque：Len16/Len32 + 原始数据字节；Hub 不解析，仅路由。

实现要点（表驱动优先）
- 以 TypeID → Schema 的映射表驱动通用编解码器：
	- Schema 记录字段顺序、类型（定长/变长/struct/array）、是否可选以及是否使用位图。
	- 解码：读 Header→查 Schema→（可选）读位图→逐字段读值；struct/array 按其规则处理。
	- 编码：按 Schema 顺序写出；定长直写、变长写短长度；位图只为出现的可选字段置位。
- 热路径可用代码生成将 Schema 展开为内联读写代码，进一步降低分支与拷贝开销。

最小 Schema 表（节选）
- 说明：以下为便于落地的字段顺序定义，名称与现有 JSON 字段对齐；可根据实现逐步补全。

0 OK_RESP
- 固定：request_id(u32), code(i32), message(len16+bytes)

1 ERROR_RESP
- 固定：request_id(u32), code(i32), message(len16+bytes)

2 USER_LOGIN_REQ
- 位图：max_uses(exists bit0), expires_at(exists bit1)
- 固定：username(len16+utf8), password(len16+utf8)
- 可选：max_uses(i32), expires_at(i64 epoch ms)

3 USER_LOGIN_RESP
- 固定：request_id(u32), token(len16+bytes), username(len16+utf8), role(i32)
- 可选位图：expires_at(bit0)
- 可选：expires_at(i64 ms)

---

## 配置说明与安全建议

server/config.json 关键字段

- Server.ListenAddr：核心服务监听地址，如 :8080
- Server.HardwareID：本节点硬件 ID/唯一名
- Server.ManagerToken：ManagerAuth 的密钥（Manager→Server 管理面）
- Server.RelayToken：ParentAuth 校验密钥（上级 Server 用）
- Server.DefaultAdmin：默认管理员（仅首次建表时生效）
- Relay.Enabled：以中继模式运行当前进程
- Relay.ParentAddr：上级 WebSocket 地址（如 ws://hub:8080/ws）
- Relay.ListenAddr：本地监听给下级的地址
- Relay.HardwareID：本中继硬件 ID
- Relay.SharedToken：ParentAuth 发起密钥（下级用）。应与上级 Server.RelayToken 一致

安全建议
- Manager 仅为 BFF，不具备系统级特权；请将 ManagerToken 与 RelayToken 分离
- 使用高熵随机值作为 RelayToken/SharedToken；避免在日志与代码中泄露
- 在完成切换后，尽量不要依赖回退到 ManagerToken 的兼容逻辑

10 MSG_SEND（条件透传）
- 固定：channel(len16+utf8)
- 位图：tag(bit0), content_type(bit1)
- 可选：tag(len16+utf8), content_type(len16+utf8)
- 负载：opaque(len16/len32+bytes)，当 Header.Target≠Hub 时透传；=Hub 时按具体子类型 Schema 解码。

20 FILE_INIT
- 固定：file_id(len16+utf8), size(i64), mime(len16+utf8)
- 位图：name(bit0), sha256(bit1), meta(bit2)
- 可选：name(len16+utf8), sha256(len16+32), meta(len16+json/utf8)

21 FILE_CHUNK
- 固定：file_id(len16+utf8), offset(i64), data(len16/len32+bytes)
- 位图：crc32(bit0)
- 可选：crc32(u32)

22 FILE_COMPLETE
- 固定：file_id(len16+utf8)
- 位图：sha256(bit0)
- 可选：sha256(len16+32)

23 FILE_CANCEL
- 固定：file_id(len16+utf8), reason(len16+utf8)

兼容与演进
- 仅允许“末尾追加字段”；不重排、不删除。新增可选字段扩展位图长度。
- Header.Version 用于大版本规则切换；TypeID 白名单与长度边界强校验。

实现提示
- 解码：读 Header→选 Schema→读位图（如有）→按顺序读必选与位图置位的可选字段；struct/array 先读长度/计数。
- 编码：按 Schema 顺序写；定长直写，变长写短长度；位图中仅对出现的可选字段置位。
