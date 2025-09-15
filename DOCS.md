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
重要：当前通信仅支持二进制子协议 myflowhub.bin.v1，所有 JSON 路径已移除。二进制帧的帧头保持不变，负载部分已全面切换为 Protobuf（proto3）。Server 二进制路由集中注册于 `server/internal/hub/register.go`（由 main 在启动时完成注入）。

目标
- 高性能、低开销、跨语言易实现。
- 帧头与 TypeID 继续沿用；负载为 Protobuf（proto3）编码的消息；支持必要的透传消息。
- Schema 统一在 `pkg/protocol/pb/myflowhub.proto` 中定义，跨语言生成代码。

传输与字节序
- WebSocket 二进制帧或 TCP；仅二进制，不保留 JSON。
- 帧头字段使用 Little-Endian；负载为 Protobuf（与字节序无关）。
npm install
帧结构
- Header（固定 38B）：
	- TypeID[2]=uint16；Reserved[4]=0；MsgID[8]=uint64；Source[8]=uint64；Target[8]=uint64；Timestamp[8]=int64
- Payload：对应 TypeID 的 Protobuf 消息（详见下表）。

负载规则（Proto）
- 采用 proto3；可选字段使用 `optional` 语义，未设置时使用零值或省略。
- 消息字段与命名均以业务含义为主；所有消息定义见 `pkg/protocol/pb/myflowhub.proto`。
- 跨语言时统一使用 Protobuf 官方或社区生成器；Go 侧由 `google.golang.org/protobuf` 提供运行时。
- 仅在“新建数据库或首次创建 users 表”时自动创建 admin，并授予 admin.manage 与 **
长度与 Varint
- 帧头字段仍为定长小端；负载交由 Protobuf 处理（采用 Varint/定长/长度前缀等内置编码）。

关于可选字段
- 通过 proto3 optional 表达；具体在 `myflowhub.proto` 中以 `optional` 关键字标注。
- 高性能、低开销、跨语言易实现。
TypeID → Protobuf 消息（节选）
- 0  OK_RESP            → pb.OKResp
- 1  ERR_RESP           → pb.ErrResp
- 10 MSG_SEND           → 透传或内部子类型（尽量也使用 Protobuf 定义）
- 20 QUERY_NODES_REQ    → pb.QueryNodesReq
- 21 CREATE_DEVICE_REQ  → pb.CreateDeviceReq
- 22 UPDATE_DEVICE_REQ  → pb.UpdateDeviceReq
- 23 DELETE_DEVICE_REQ  → pb.DeleteDeviceReq
- 100 MANAGER_AUTH_REQ  → pb.ManagerAuthReq
- 101 MANAGER_AUTH_RESP → pb.ManagerAuthResp
- 110 USER_LOGIN_REQ    → pb.UserLoginReq
- 111 USER_LOGIN_RESP   → pb.UserLoginResp
- 112 USER_ME_REQ       → pb.UserMeReq
- 113 USER_ME_RESP      → pb.UserMeResp
- 114 USER_LOGOUT_REQ   → pb.UserLogoutReq
- 115 USER_LOGOUT_RESP  → pb.OKResp/pb.ErrResp（按处理结果）
- 130 PARENT_AUTH_REQ   → pb.ParentAuthReq（字段长度有固定约束，详见 proto 注释）
- 131 PARENT_AUTH_RESP  → pb.ParentAuthResp（字段长度有固定约束，详见 proto 注释）
- 150 SYSTEMLOG_LIST_REQ  → pb.SystemLogListReq
- 151 SYSTEMLOG_LIST_RESP → pb.SystemLogListResp
- 170 KEY_LIST_REQ        → pb.KeyListReq
- 171 KEY_LIST_RESP       → pb.KeyListResp
- 172 KEY_CREATE_REQ      → pb.KeyCreateReq
- 173 KEY_CREATE_RESP     → pb.KeyCreateResp
- 174 KEY_UPDATE_REQ      → pb.KeyUpdateReq
- 175 KEY_DELETE_REQ      → pb.KeyDeleteReq
- 176 KEY_DEVICES_REQ     → pb.KeyDevicesReq
- 177 KEY_DEVICES_RESP    → pb.KeyDevicesResp
- 180 USER_LIST_REQ       → pb.UserListReq（服务端返回 pb.UserListResp）
- 181 USER_LIST_RESP      → pb.UserListResp
- 182 USER_CREATE_REQ     → pb.UserCreateReq（返回 pb.UserCreateResp）
- 183 USER_CREATE_RESP    → pb.UserCreateResp
- 184 USER_UPDATE_REQ     → pb.UserUpdateReq
- 185 USER_DELETE_REQ     → pb.UserDeleteReq
- 186 USER_PERM_LIST_REQ  → pb.UserPermListReq（返回 pb.UserPermListResp）
- 187 USER_PERM_LIST_RESP → pb.UserPermListResp
- 188 USER_PERM_ADD_REQ   → pb.UserPermAddReq
- 189 USER_PERM_REMOVE_REQ→ pb.UserPermRemoveReq
- 190 USER_SELF_UPDATE_REQ   → pb.UserSelfUpdateReq
- 191 USER_SELF_PASSWORD_REQ → pb.UserSelfPasswordReq
- Little-Endian。
结构体说明：Device
- 见 `pb.DeviceItem`；服务侧存在 Go 内部模型与 pb 之间的映射辅助（fromPB/toPB）。
- WebSocket 子协议：Sec-WebSocket-Protocol: myflowhub.bin.v1（建议）。
“透传”消息
- MSG_SEND(10) 仅在 Target≠Hub 时透传；若 Target = Hub，建议也采用 Protobuf 子类型并由 Hub 解析。

文件/媒体传输（建议）
- 如需文件分片协议，亦建议使用 Protobuf 定义消息结构；TypeID 另行分配。
- 变长类型：
示例帧
- USER_LOGIN_REQ：负载为 `pb.UserLoginReq{username,password}` 的 Protobuf 二进制；Header.Type=110。
- 同理，所有请求/响应均对应到上述 `pb.*` 消息。
- 可选字段位图：ceiling(N/8) 字节，bit0→第1个可选字段…；为 1 则该字段随后的顺序中出现，为 0 则不写。
实现要点
- 以 TypeID → Protobuf 消息 映射为准；解码：读 Header→按 TypeID 将负载 `proto.Unmarshal` 为对应消息；编码：构造 pb.* → `proto.Marshal` 写入负载。
- 热路径仍可通过消息复用与 `[]byte` 池优化内存拷贝。
- Len32：无符号 32 位长度（不含自身）。

- 某消息有 6 个可选字段 C..H，则位图占 1 字节（ceil(6/8)=1）。
- 解码顺序：读位图→读所有必选→按 C..H 顺序，对置位字段依其类型读值，未置位则跳过。
- 1 ERROR_RESP：success:bool(=0)，error:string，可选 original_id:u64
Schema 参考
- 全量字段、可选与注释均在 `pkg/protocol/pb/myflowhub.proto` 内维护，作为权威文档。
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
