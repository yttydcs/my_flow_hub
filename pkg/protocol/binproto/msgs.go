package binproto

import (
	"errors"
	pb "myflowhub/pkg/protocol/pb"

	"google.golang.org/protobuf/proto"
)

// Stable Type IDs per DOCS.md (subset)
const (
	TypeOKResp  uint16 = 0
	TypeErrResp uint16 = 1
	TypeMsgSend uint16 = 10
	// Devices
	TypeQueryNodesReq   uint16 = 20
	TypeCreateDeviceReq uint16 = 21
	TypeUpdateDeviceReq uint16 = 22
	TypeDeleteDeviceReq uint16 = 23
	// Responses for device operations (reserve 120+ range for responses)
	TypeQueryNodesResp  uint16 = 120
	TypeManagerAuthReq  uint16 = 100
	TypeManagerAuthResp uint16 = 101
	// User/Auth flows
	TypeUserLoginReq   uint16 = 110
	TypeUserLoginResp  uint16 = 111
	TypeUserMeReq      uint16 = 112
	TypeUserMeResp     uint16 = 113
	TypeUserLogoutReq  uint16 = 114
	TypeUserLogoutResp uint16 = 115
	// System Log
	TypeSystemLogListReq  uint16 = 150
	TypeSystemLogListResp uint16 = 151
)

// ========== Parent Link Auth ==========
const (
	TypeParentAuthReq  uint16 = 130
	TypeParentAuthResp uint16 = 131
)

// ========== Keys Management ==========
const (
	TypeKeyListReq     uint16 = 170
	TypeKeyListResp    uint16 = 171
	TypeKeyCreateReq   uint16 = 172
	TypeKeyCreateResp  uint16 = 173
	TypeKeyUpdateReq   uint16 = 174
	TypeKeyDeleteReq   uint16 = 175
	TypeKeyDevicesReq  uint16 = 176
	TypeKeyDevicesResp uint16 = 177
)

// ========== Users Management (Admin + Self) ==========
const (
	// Admin-side user CRUD
	TypeUserListReq    uint16 = 180
	TypeUserListResp   uint16 = 181
	TypeUserCreateReq  uint16 = 182
	TypeUserCreateResp uint16 = 183
	TypeUserUpdateReq  uint16 = 184
	TypeUserDeleteReq  uint16 = 185
	// User permissions (nodes)
	TypeUserPermListReq   uint16 = 186
	TypeUserPermListResp  uint16 = 187
	TypeUserPermAddReq    uint16 = 188
	TypeUserPermRemoveReq uint16 = 189
	// Self-service
	TypeUserSelfUpdateReq   uint16 = 190
	TypeUserSelfPasswordReq uint16 = 191
)

// EncodeOKResp encodes {request_id:u64, code:i32, message:len16+utf8}
func EncodeOKResp(requestID uint64, code int32, message []byte) []byte {
	m := &pb.OKResp{RequestId: requestID, Code: code, Message: append([]byte(nil), message...)}
	b, _ := proto.Marshal(m)
	return b
}

// DecodeOKResp decodes payload of OK_RESP
func DecodeOKResp(b []byte) (requestID uint64, code int32, message []byte, err error) {
	var m pb.OKResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, 0, nil, err
	}
	return m.GetRequestId(), m.GetCode(), append([]byte(nil), m.GetMessage()...), nil
}

// EncodeErrResp mirrors OK but used for TypeErrResp
func EncodeErrResp(requestID uint64, code int32, message []byte) []byte {
	m := &pb.ErrResp{RequestId: requestID, Code: code, Message: append([]byte(nil), message...)}
	b, _ := proto.Marshal(m)
	return b
}

// DecodeErrResp mirrors OK
func DecodeErrResp(b []byte) (uint64, int32, []byte, error) {
	var m pb.ErrResp
	if err := proto.Unmarshal(b, &m); err != nil {
		return 0, 0, nil, err
	}
	return m.GetRequestId(), m.GetCode(), append([]byte(nil), m.GetMessage()...), nil
}

// ManagerAuth: Req {token:len16+utf8}
func EncodeManagerAuthReq(token string) []byte {
	m := &pb.ManagerAuthReq{Token: token}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeManagerAuthReq(b []byte) (string, error) {
	var m pb.ManagerAuthReq
	if err := proto.Unmarshal(b, &m); err != nil {
		return "", err
	}
	return m.GetToken(), nil
}

// ManagerAuth: Resp {request_id:u64, device_uid:u64, role(len16+utf8 optional, 0 长度视为缺省)}
func EncodeManagerAuthResp(requestID, deviceUID uint64, role string) []byte {
	m := &pb.ManagerAuthResp{RequestId: requestID, DeviceUid: deviceUID, Role: role}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeManagerAuthResp(b []byte) (reqID, deviceUID uint64, role string, err error) {
	var m pb.ManagerAuthResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, 0, "", err
	}
	return m.GetRequestId(), m.GetDeviceUid(), m.GetRole(), nil
}

// ParentAuthReq: {version:u8, ts:i64(ms), nonce:16B, hardware_id:len16+str, caps:len16+str, sig:32B}
func EncodeParentAuthReq(version uint8, ts int64, nonce [16]byte, hardwareID, caps string, sig [32]byte) []byte {
	m := &pb.ParentAuthReq{
		Version:    uint32(version),
		TsMs:       ts,
		Nonce:      append([]byte(nil), nonce[:]...),
		HardwareId: hardwareID,
		Caps:       caps,
		Sig:        append([]byte(nil), sig[:]...),
	}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeParentAuthReq(b []byte) (version uint8, ts int64, nonce [16]byte, hardwareID, caps string, sig [32]byte, err error) {
	var m pb.ParentAuthReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, 0, nonce, "", "", sig, err
	}
	version = uint8(m.GetVersion() & 0xff)
	ts = m.GetTsMs()
	nb := m.GetNonce()
	if len(nb) != 16 {
		return 0, 0, nonce, "", "", sig, errors.New("invalid nonce length")
	}
	copy(nonce[:], nb)
	hardwareID = m.GetHardwareId()
	caps = m.GetCaps()
	sb := m.GetSig()
	if len(sb) != 32 {
		return 0, 0, nonce, "", "", sig, errors.New("invalid sig length")
	}
	copy(sig[:], sb)
	return
}

// ParentAuthResp: {request_id:u64, device_uid:u64, session_id:16B, heartbeat_sec:u16, perms:[len16+str], exp:i64, sig:32B}
func EncodeParentAuthResp(requestID, deviceUID uint64, sessionID [16]byte, heartbeatSec uint16, perms []string, exp int64, sig [32]byte) []byte {
	m := &pb.ParentAuthResp{
		RequestId:    requestID,
		DeviceUid:    deviceUID,
		SessionId:    append([]byte(nil), sessionID[:]...),
		HeartbeatSec: uint32(heartbeatSec),
		Perms:        append([]string(nil), perms...),
		Exp:          exp,
		Sig:          append([]byte(nil), sig[:]...),
	}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeParentAuthResp(b []byte) (requestID, deviceUID uint64, sessionID [16]byte, heartbeatSec uint16, perms []string, exp int64, sig [32]byte, err error) {
	var m pb.ParentAuthResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, 0, sessionID, 0, nil, 0, sig, err
	}
	requestID = m.GetRequestId()
	deviceUID = m.GetDeviceUid()
	sid := m.GetSessionId()
	if len(sid) != 16 {
		return 0, 0, sessionID, 0, nil, 0, sig, errors.New("invalid sessionID length")
	}
	copy(sessionID[:], sid)
	heartbeatSec = uint16(m.GetHeartbeatSec() & 0xffff)
	perms = append([]string(nil), m.GetPerms()...)
	exp = m.GetExp()
	sb := m.GetSig()
	if len(sb) != 32 {
		return 0, 0, sessionID, 0, nil, 0, sig, errors.New("invalid sig length")
	}
	copy(sig[:], sb)
	return
}

// ========== Users Management payloads ==========

// UserItem 精简版用户对象（避免泄漏密码哈希）
type UserItem struct {
	ID           uint64
	Username     string
	DisplayName  string
	Disabled     bool
	CreatedAtSec int64
	UpdatedAtSec int64
}

// 用户对象与 PB 映射
func toPBUserItem(u UserItem) *pb.UserItem {
	return &pb.UserItem{
		Id:           u.ID,
		Username:     u.Username,
		DisplayName:  u.DisplayName,
		Disabled:     u.Disabled,
		CreatedAtSec: u.CreatedAtSec,
		UpdatedAtSec: u.UpdatedAtSec,
	}
}
func fromPBUserItem(m *pb.UserItem) UserItem {
	if m == nil {
		return UserItem{}
	}
	return UserItem{
		ID:           m.GetId(),
		Username:     m.GetUsername(),
		DisplayName:  m.GetDisplayName(),
		Disabled:     m.GetDisabled(),
		CreatedAtSec: m.GetCreatedAtSec(),
		UpdatedAtSec: m.GetUpdatedAtSec(),
	}
}

// EncodeUserListResp {request_id:u64, count:u32, items:[UserItem]}
func EncodeUserListResp(requestID uint64, items []UserItem) []byte {
	users := make([]*pb.UserItem, 0, len(items))
	for _, u := range items {
		users = append(users, toPBUserItem(u))
	}
	m := &pb.UserListResp{RequestId: requestID, Users: users}
	b, _ := proto.Marshal(m)
	return b
}

// DecodeUserListResp -> (request_id, items)
func DecodeUserListResp(b []byte) (requestID uint64, items []UserItem, err error) {
	var m pb.UserListResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, nil, err
	}
	requestID = m.GetRequestId()
	users := m.GetUsers()
	items = make([]UserItem, 0, len(users))
	for _, u := range users {
		items = append(items, fromPBUserItem(u))
	}
	return
}

// EncodeUserCreateReq {user_key:str, username:str, display_name:str, password:str}
func EncodeUserCreateReq(userKey, username, displayName, password string) []byte {
	m := &pb.UserCreateReq{UserKey: userKey, Username: username, DisplayName: displayName, Password: password}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeUserCreateReq(b []byte) (userKey, username, displayName, password string, err error) {
	var m pb.UserCreateReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", "", "", "", err
	}
	return m.GetUserKey(), m.GetUsername(), m.GetDisplayName(), m.GetPassword(), nil
}

// EncodeUserCreateResp {request_id:u64, id:u64}
func EncodeUserCreateResp(requestID, id uint64) []byte {
	m := &pb.UserCreateResp{RequestId: requestID, UserId: id}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeUserCreateResp(b []byte) (requestID, id uint64, err error) {
	var m pb.UserCreateResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, 0, err
	}
	return m.GetRequestId(), m.GetUserId(), nil
}

// EncodeUserUpdateReq {user_key:str, id:u64, bitmap: display_name(bit0), password(bit1), disabled(bit2); fields}
func EncodeUserUpdateReq(userKey string, id uint64, displayName *string, password *string, disabled *bool) []byte {
	m := &pb.UserUpdateReq{UserKey: userKey, Id: id}
	if displayName != nil {
		m.DisplayName = displayName
	}
	if password != nil {
		m.Password = password
	}
	if disabled != nil {
		m.Disabled = disabled
	}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeUserUpdateReq(b []byte) (userKey string, id uint64, displayName *string, password *string, disabled *bool, err error) {
	var m pb.UserUpdateReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", 0, nil, nil, nil, err
	}
	userKey = m.GetUserKey()
	id = m.GetId()
	if m.DisplayName != nil {
		v := m.GetDisplayName()
		displayName = &v
	}
	if m.Password != nil {
		v := m.GetPassword()
		password = &v
	}
	if m.Disabled != nil {
		v := m.GetDisabled()
		disabled = &v
	}
	return
}

// EncodeUserDeleteReq {user_key:str, id:u64}
func EncodeUserDeleteReq(userKey string, id uint64) []byte {
	m := &pb.UserDeleteReq{UserKey: userKey, Id: id}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeUserDeleteReq(b []byte) (userKey string, id uint64, err error) {
	var m pb.UserDeleteReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", 0, err
	}
	return m.GetUserKey(), m.GetId(), nil
}

// PermissionItem 权限节点条目
type PermissionItem struct{ Node string }

// EncodeUserPermListResp {request_id:u64, count:u32, items:[len16+utf8]}
func EncodeUserPermListResp(requestID uint64, items []PermissionItem) []byte {
	nodes := make([]string, 0, len(items))
	for _, it := range items {
		nodes = append(nodes, it.Node)
	}
	m := &pb.UserPermListResp{RequestId: requestID, Nodes: nodes}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeUserPermListResp(b []byte) (requestID uint64, items []PermissionItem, err error) {
	var m pb.UserPermListResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, nil, err
	}
	requestID = m.GetRequestId()
	ns := m.GetNodes()
	items = make([]PermissionItem, 0, len(ns))
	for _, n := range ns {
		items = append(items, PermissionItem{Node: n})
	}
	return
}

// EncodeUserPermListReq {user_key:str, user_id:u64}
func EncodeUserPermListReq(userKey string, userID uint64) []byte {
	m := &pb.UserPermListReq{UserKey: userKey, UserId: userID}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeUserPermListReq(b []byte) (userKey string, userID uint64, err error) {
	var m pb.UserPermListReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", 0, err
	}
	return m.GetUserKey(), m.GetUserId(), nil
}

// EncodeUserPermAddReq {user_key:str, user_id:u64, node:str}
func EncodeUserPermAddReq(userKey string, userID uint64, node string) []byte {
	m := &pb.UserPermAddReq{UserKey: userKey, UserId: userID, Node: node}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeUserPermAddReq(b []byte) (userKey string, userID uint64, node string, err error) {
	var m pb.UserPermAddReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", 0, "", err
	}
	return m.GetUserKey(), m.GetUserId(), m.GetNode(), nil
}

// EncodeUserPermRemoveReq 同 Add，仅语义不同
func EncodeUserPermRemoveReq(userKey string, userID uint64, node string) []byte {
	m := &pb.UserPermRemoveReq{UserKey: userKey, UserId: userID, Node: node}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeUserPermRemoveReq(b []byte) (userKey string, userID uint64, node string, err error) {
	var m pb.UserPermRemoveReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", 0, "", err
	}
	return m.GetUserKey(), m.GetUserId(), m.GetNode(), nil
}

// EncodeUserSelfUpdateReq {user_key:str, display_name:str}
func EncodeUserSelfUpdateReq(userKey string, displayName string) []byte {
	m := &pb.UserSelfUpdateReq{UserKey: userKey, DisplayName: displayName}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeUserSelfUpdateReq(b []byte) (userKey string, displayName string, err error) {
	var m pb.UserSelfUpdateReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", "", err
	}
	return m.GetUserKey(), m.GetDisplayName(), nil
}

// EncodeUserSelfPasswordReq {user_key:str, old_password:str, new_password:str}
func EncodeUserSelfPasswordReq(userKey, oldPassword, newPassword string) []byte {
	m := &pb.UserSelfPasswordReq{UserKey: userKey, OldPassword: oldPassword, NewPassword: newPassword}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeUserSelfPasswordReq(b []byte) (userKey, oldPassword, newPassword string, err error) {
	var m pb.UserSelfPasswordReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", "", "", err
	}
	return m.GetUserKey(), m.GetOldPassword(), m.GetNewPassword(), nil
}

// ========== User/Auth ==========
// UserLoginReq {username:str, password:str}
func EncodeUserLoginReq(username, password string) []byte {
	m := &pb.UserLoginReq{Username: username, Password: password}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeUserLoginReq(b []byte) (username, password string, err error) {
	var m pb.UserLoginReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", "", err
	}
	return m.GetUsername(), m.GetPassword(), nil
}

// UserLoginResp {request_id:u64, token:str, key_id:u64, user_id:u64, username:str, display_name:str, perms:[str]}
func EncodeUserLoginResp(requestID, keyID, userID uint64, token, username, displayName string, perms []string) []byte {
	m := &pb.UserLoginResp{
		RequestId:   requestID,
		KeyId:       keyID,
		UserId:      userID,
		Secret:      token,
		Username:    username,
		DisplayName: displayName,
		Permissions: append([]string(nil), perms...),
	}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeUserLoginResp(b []byte) (requestID, keyID, userID uint64, token, username, displayName string, perms []string, err error) {
	var m pb.UserLoginResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, 0, 0, "", "", "", nil, err
	}
	return m.GetRequestId(), m.GetKeyId(), m.GetUserId(), m.GetSecret(), m.GetUsername(), m.GetDisplayName(), append([]string(nil), m.GetPermissions()...), nil
}

// UserMeReq {user_key:str}
func EncodeUserMeReq(userKey string) []byte {
	m := &pb.UserMeReq{UserKey: userKey}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeUserMeReq(b []byte) (string, error) {
	var m pb.UserMeReq
	if err := proto.Unmarshal(b, &m); err != nil {
		return "", err
	}
	return m.GetUserKey(), nil
}

// UserMeResp {request_id:u64, user_id:u64, username:str, display_name:str, perms:[str]}
func EncodeUserMeResp(requestID, userID uint64, username, displayName string, perms []string) []byte {
	m := &pb.UserMeResp{
		RequestId:   requestID,
		UserId:      userID,
		Username:    username,
		DisplayName: displayName,
		Permissions: append([]string(nil), perms...),
	}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeUserMeResp(b []byte) (requestID, userID uint64, username, displayName string, perms []string, err error) {
	var m pb.UserMeResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, 0, "", "", nil, err
	}
	return m.GetRequestId(), m.GetUserId(), m.GetUsername(), m.GetDisplayName(), append([]string(nil), m.GetPermissions()...), nil
}

// UserLogoutReq {user_key:str}
func EncodeUserLogoutReq(userKey string) []byte {
	m := &pb.UserLogoutReq{UserKey: userKey}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeUserLogoutReq(b []byte) (string, error) {
	var m pb.UserLogoutReq
	if err := proto.Unmarshal(b, &m); err != nil {
		return "", err
	}
	return m.GetUserKey(), nil
}

// ========== System Log ==========
// SystemLogListReq {user_key:str, level?:str, source?:str, keyword?:str, start_at:i64, end_at:i64, page:i32, page_size:i32}
func EncodeSystemLogListReq(userKey, level, source, keyword string, startAt, endAt int64, page, pageSize int32) []byte {
	m := &pb.SystemLogListReq{
		UserKey:  userKey,
		StartAt:  startAt,
		EndAt:    endAt,
		Page:     page,
		PageSize: pageSize,
	}
	if level != "" {
		m.Level = &level
	}
	if source != "" {
		m.Source = &source
	}
	if keyword != "" {
		m.Keyword = &keyword
	}
	b, _ := proto.Marshal(m)
	return b
}

type SystemLogItem struct {
	Level, Source, Message, Details string
	At                              int64
}

// SystemLogListResp {request_id:u64, total:i64, page:i32, page_size:i32, logs:[{level,source,message,details,at}]}
func EncodeSystemLogListResp(requestID uint64, total int64, page, pageSize int32, logs []SystemLogItem) []byte {
	items := make([]*pb.SystemLogItem, 0, len(logs))
	for _, lg := range logs {
		items = append(items, &pb.SystemLogItem{
			Level:   lg.Level,
			Source:  lg.Source,
			Message: lg.Message,
			Details: lg.Details,
			At:      lg.At,
		})
	}
	m := &pb.SystemLogListResp{RequestId: requestID, Total: total, Page: page, PageSize: pageSize, Logs: items}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeSystemLogListReq(b []byte) (userKey, level, source, keyword string, startAt, endAt int64, page, pageSize int32, err error) {
	var m pb.SystemLogListReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", "", "", "", 0, 0, 0, 0, err
	}
	userKey = m.GetUserKey()
	level = m.GetLevel()
	source = m.GetSource()
	keyword = m.GetKeyword()
	startAt = m.GetStartAt()
	endAt = m.GetEndAt()
	page = m.GetPage()
	pageSize = m.GetPageSize()
	return
}
func DecodeSystemLogListResp(b []byte) (requestID uint64, total int64, page, pageSize int32, logs []SystemLogItem, err error) {
	var m pb.SystemLogListResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, 0, 0, 0, nil, err
	}
	requestID = m.GetRequestId()
	total = m.GetTotal()
	page = m.GetPage()
	pageSize = m.GetPageSize()
	logs = make([]SystemLogItem, 0, len(m.GetLogs()))
	for _, it := range m.GetLogs() {
		logs = append(logs, SystemLogItem{Level: it.GetLevel(), Source: it.GetSource(), Message: it.GetMessage(), Details: it.GetDetails(), At: it.GetAt()})
	}
	return
}

// ========== Devices: Query/List ==========
// DeviceItem is a compact device representation for binary protocol
type DeviceItem struct {
	ID           uint64
	DeviceUID    uint64
	HardwareID   string
	Role         string
	Name         string
	ParentID     *uint64
	OwnerUserID  *uint64
	LastSeenSec  *int64 // epoch seconds
	CreatedAtSec int64  // epoch seconds
	UpdatedAtSec int64  // epoch seconds
	Approved     *bool  // 审批状态（可空）
}

// protobuf mapping helpers for DeviceItem
func toPBDeviceItem(d DeviceItem) *pb.DeviceItem {
	var parentID *uint64
	if d.ParentID != nil {
		v := *d.ParentID
		parentID = &v
	}
	var ownerID *uint64
	if d.OwnerUserID != nil {
		v := *d.OwnerUserID
		ownerID = &v
	}
	var lastSeen *int64
	if d.LastSeenSec != nil {
		v := *d.LastSeenSec
		lastSeen = &v
	}
	var approved *bool
	if d.Approved != nil {
		v := *d.Approved
		approved = &v
	}
	return &pb.DeviceItem{
		Id:           d.ID,
		DeviceUid:    d.DeviceUID,
		HardwareId:   d.HardwareID,
		Role:         d.Role,
		Name:         d.Name,
		ParentId:     parentID,
		OwnerUserId:  ownerID,
		LastSeenSec:  lastSeen,
		CreatedAtSec: d.CreatedAtSec,
		UpdatedAtSec: d.UpdatedAtSec,
		Approved:     approved,
	}
}

func fromPBDeviceItem(p *pb.DeviceItem) DeviceItem {
	it := DeviceItem{
		ID:           p.GetId(),
		DeviceUID:    p.GetDeviceUid(),
		HardwareID:   p.GetHardwareId(),
		Role:         p.GetRole(),
		Name:         p.GetName(),
		CreatedAtSec: p.GetCreatedAtSec(),
		UpdatedAtSec: p.GetUpdatedAtSec(),
	}
	if p.ParentId != nil {
		v := p.GetParentId()
		it.ParentID = &v
	}
	if p.OwnerUserId != nil {
		v := p.GetOwnerUserId()
		it.OwnerUserID = &v
	}
	if p.LastSeenSec != nil {
		v := p.GetLastSeenSec()
		it.LastSeenSec = &v
	}
	if p.Approved != nil {
		v := p.GetApproved()
		it.Approved = &v
	}
	return it
}

// EncodeQueryNodesReq {bitmap(1)=user_key(bit0), user_key?:str}
func EncodeQueryNodesReq(userKey string) []byte {
	m := &pb.QueryNodesReq{}
	if userKey != "" {
		m.UserKey = &userKey
	}
	b, _ := proto.Marshal(m)
	return b
}

// DecodeQueryNodesReq -> userKey (optional)
func DecodeQueryNodesReq(b []byte) (string, error) {
	var m pb.QueryNodesReq
	if err := proto.Unmarshal(b, &m); err != nil {
		return "", err
	}
	if m.UserKey == nil {
		return "", nil
	}
	return m.GetUserKey(), nil
}

// helper: encode one DeviceItem with inner bitmap for optional fields

// EncodeQueryNodesResp {request_id:u64, count:varint, devices:[DeviceItem]}
func EncodeQueryNodesResp(requestID uint64, devices []DeviceItem) []byte {
	items := make([]*pb.DeviceItem, 0, len(devices))
	for _, d := range devices {
		items = append(items, toPBDeviceItem(d))
	}
	m := &pb.QueryNodesResp{RequestId: requestID, Devices: items}
	b, _ := proto.Marshal(m)
	return b
}

func DecodeQueryNodesResp(b []byte) (requestID uint64, devices []DeviceItem, err error) {
	var m pb.QueryNodesResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, nil, err
	}
	requestID = m.GetRequestId()
	devices = make([]DeviceItem, 0, len(m.GetDevices()))
	for _, it := range m.GetDevices() {
		devices = append(devices, fromPBDeviceItem(it))
	}
	return
}

// ========== Devices: Create/Update/Delete ==========
// Create/Update: {bitmap(1)=user_key(bit0), user_key?:str, device:DeviceItem}
func EncodeCreateDeviceReq(userKey string, d DeviceItem) []byte {
	m := &pb.CreateDeviceReq{Device: toPBDeviceItem(d)}
	if userKey != "" {
		m.UserKey = &userKey
	}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeCreateDeviceReq(b []byte) (userKey string, d DeviceItem, err error) {
	var m pb.CreateDeviceReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", DeviceItem{}, err
	}
	if m.UserKey != nil {
		userKey = m.GetUserKey()
	}
	if m.Device == nil {
		return userKey, DeviceItem{}, errors.New("missing device")
	}
	d = fromPBDeviceItem(m.Device)
	return
}
func EncodeUpdateDeviceReq(userKey string, d DeviceItem) []byte {
	m := &pb.UpdateDeviceReq{Device: toPBDeviceItem(d)}
	if userKey != "" {
		m.UserKey = &userKey
	}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeUpdateDeviceReq(b []byte) (userKey string, d DeviceItem, err error) {
	var m pb.UpdateDeviceReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", DeviceItem{}, err
	}
	if m.UserKey != nil {
		userKey = m.GetUserKey()
	}
	if m.Device == nil {
		return userKey, DeviceItem{}, errors.New("missing device")
	}
	d = fromPBDeviceItem(m.Device)
	return
}

// Delete: {bitmap(1)=user_key(bit0), id:u64, user_key?:str}
func EncodeDeleteDeviceReq(id uint64, userKey string) []byte {
	m := &pb.DeleteDeviceReq{Id: id}
	if userKey != "" {
		m.UserKey = &userKey
	}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeDeleteDeviceReq(b []byte) (id uint64, userKey string, err error) {
	var m pb.DeleteDeviceReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, "", err
	}
	id = m.GetId()
	if m.UserKey != nil {
		userKey = m.GetUserKey()
	}
	return
}

// ========== Variables: Update/Delete ==========
// Type IDs
// Note: keep in sync with server routes
const (
	TypeVarListReq   uint16 = 160 // reserved for later
	TypeVarListResp  uint16 = 161 // reserved for later
	TypeVarUpdateReq uint16 = 162
	TypeVarDeleteReq uint16 = 163
)

// ========== Variables: List/Query ==========
// VarListReq {bitmap(1)= user_key(bit0), device_uid(bit1); user_key?:str, device_uid?:u64}
func EncodeVarListReq(userKey string, deviceUID *uint64) []byte {
	m := &pb.VarListReq{}
	if userKey != "" {
		m.UserKey = &userKey
	}
	if deviceUID != nil {
		v := *deviceUID
		m.DeviceUid = &v
	}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeVarListReq(b []byte) (userKey string, deviceUID *uint64, err error) {
	var m pb.VarListReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", nil, err
	}
	if m.UserKey != nil {
		userKey = m.GetUserKey()
	}
	if m.DeviceUid != nil {
		v := m.GetDeviceUid()
		deviceUID = &v
	}
	return
}

// VarListItem 表示一个变量条目
type VarListItem struct {
	ID             uint64
	OwnerDeviceID  uint64
	OwnerDeviceUID uint64
	Name           string
	Value          []byte // JSON bytes
	CreatedAtSec   int64
	UpdatedAtSec   int64
}

// VarListResp {request_id:u64, count:varint, items:[{id:u64, owner_id:u64, owner_uid:u64, name:str, value:len16+json, created_at:i64, updated_at:i64}]}
func EncodeVarListResp(requestID uint64, items []VarListItem) []byte {
	arr := make([]*pb.VarListItem, 0, len(items))
	for _, it := range items {
		arr = append(arr, &pb.VarListItem{
			Id:             it.ID,
			OwnerDeviceId:  it.OwnerDeviceID,
			OwnerDeviceUid: it.OwnerDeviceUID,
			Name:           it.Name,
			Value:          append([]byte(nil), it.Value...),
			CreatedAtSec:   it.CreatedAtSec,
			UpdatedAtSec:   it.UpdatedAtSec,
		})
	}
	m := &pb.VarListResp{RequestId: requestID, Items: arr}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeVarListResp(b []byte) (requestID uint64, items []VarListItem, err error) {
	var m pb.VarListResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, nil, err
	}
	requestID = m.GetRequestId()
	items = make([]VarListItem, 0, len(m.GetItems()))
	for _, p := range m.GetItems() {
		items = append(items, VarListItem{
			ID:             p.GetId(),
			OwnerDeviceID:  p.GetOwnerDeviceId(),
			OwnerDeviceUID: p.GetOwnerDeviceUid(),
			Name:           p.GetName(),
			Value:          append([]byte(nil), p.GetValue()...),
			CreatedAtSec:   p.GetCreatedAtSec(),
			UpdatedAtSec:   p.GetUpdatedAtSec(),
		})
	}
	return
}

type VarUpdateItem struct {
	DeviceUID uint64
	Name      string
	Value     []byte // JSON bytes
}

// VarUpdateReq {user_key:str (len16, 0 表示缺省), count:varint, items:[{device_uid:u64, name:len16+utf8, value:len16+json}]}
func EncodeVarUpdateReq(userKey string, items []VarUpdateItem) []byte {
	arr := make([]*pb.VarUpdateItem, 0, len(items))
	for _, it := range items {
		arr = append(arr, &pb.VarUpdateItem{DeviceUid: it.DeviceUID, Name: it.Name, Value: append([]byte(nil), it.Value...)})
	}
	m := &pb.VarUpdateReq{Items: arr}
	if userKey != "" {
		m.UserKey = &userKey
	}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeVarUpdateReq(b []byte) (userKey string, items []VarUpdateItem, err error) {
	var m pb.VarUpdateReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", nil, err
	}
	if m.UserKey != nil {
		userKey = m.GetUserKey()
	}
	items = make([]VarUpdateItem, 0, len(m.GetItems()))
	for _, it := range m.GetItems() {
		items = append(items, VarUpdateItem{DeviceUID: it.GetDeviceUid(), Name: it.GetName(), Value: append([]byte(nil), it.GetValue()...)})
	}
	return
}

type VarDeleteItem struct {
	DeviceUID uint64
	Name      string
}

// VarDeleteReq {user_key:str (len16, 0 表示缺省), count:varint, items:[{device_uid:u64, name:len16+utf8}]}
func EncodeVarDeleteReq(userKey string, items []VarDeleteItem) []byte {
	arr := make([]*pb.VarDeleteItem, 0, len(items))
	for _, it := range items {
		arr = append(arr, &pb.VarDeleteItem{DeviceUid: it.DeviceUID, Name: it.Name})
	}
	m := &pb.VarDeleteReq{Items: arr}
	if userKey != "" {
		m.UserKey = &userKey
	}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeVarDeleteReq(b []byte) (userKey string, items []VarDeleteItem, err error) {
	var m pb.VarDeleteReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", nil, err
	}
	if m.UserKey != nil {
		userKey = m.GetUserKey()
	}
	items = make([]VarDeleteItem, 0, len(m.GetItems()))
	for _, it := range m.GetItems() {
		items = append(items, VarDeleteItem{DeviceUID: it.GetDeviceUid(), Name: it.GetName()})
	}
	return
}

// ================= Keys Management (codecs) =================
// KeyItem represents a key record in binary payloads.
type KeyItem struct {
	ID              uint64
	OwnerUserID     *uint64
	BindSubjectType *string
	BindSubjectID   *uint64
	SecretHash      string
	ExpiresAtSec    *int64
	MaxUses         *int32
	RemainingUses   *int32
	Revoked         bool
	IssuedBy        *uint64
	IssuedAtSec     int64
	Meta            []byte // raw JSON
}

// helper: encode/decode KeyItem with bitmap for optionals
// bitmap bits:
//
//	bit0 OwnerUserID
//	bit1 BindSubjectType
//	bit2 BindSubjectID
//	bit3 ExpiresAtSec
//	bit4 MaxUses
//	bit5 RemainingUses
//	bit6 IssuedBy
//	bit7 Meta

// KeyListReq: {user_key:str}
func EncodeKeyListReq(userKey string) []byte {
	m := &pb.KeyListReq{UserKey: userKey}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeKeyListReq(b []byte) (string, error) {
	var m pb.KeyListReq
	if err := proto.Unmarshal(b, &m); err != nil {
		return "", err
	}
	return m.GetUserKey(), nil
}

// KeyListResp: {request_id:u64, count:varint, items:[KeyItem]}
func EncodeKeyListResp(requestID uint64, items []KeyItem) []byte {
	arr := make([]*pb.KeyItem, 0, len(items))
	for _, it := range items {
		var owner *uint64
		if it.OwnerUserID != nil {
			v := *it.OwnerUserID
			owner = &v
		}
		var bst *string
		if it.BindSubjectType != nil {
			v := *it.BindSubjectType
			bst = &v
		}
		var bsid *uint64
		if it.BindSubjectID != nil {
			v := *it.BindSubjectID
			bsid = &v
		}
		var exp *int64
		if it.ExpiresAtSec != nil {
			v := *it.ExpiresAtSec
			exp = &v
		}
		var mx *int32
		if it.MaxUses != nil {
			v := *it.MaxUses
			mx = &v
		}
		var rem *int32
		if it.RemainingUses != nil {
			v := *it.RemainingUses
			rem = &v
		}
		var issuedBy *uint64
		if it.IssuedBy != nil {
			v := *it.IssuedBy
			issuedBy = &v
		}
		arr = append(arr, &pb.KeyItem{
			Id:              it.ID,
			OwnerUserId:     owner,
			BindSubjectType: bst,
			BindSubjectId:   bsid,
			SecretHash:      it.SecretHash,
			ExpiresAtSec:    exp,
			MaxUses:         mx,
			RemainingUses:   rem,
			Revoked:         it.Revoked,
			IssuedBy:        issuedBy,
			IssuedAtSec:     it.IssuedAtSec,
			Meta:            append([]byte(nil), it.Meta...),
		})
	}
	m := &pb.KeyListResp{RequestId: requestID, Items: arr}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeKeyListResp(b []byte) (requestID uint64, items []KeyItem, err error) {
	var m pb.KeyListResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, nil, err
	}
	requestID = m.GetRequestId()
	items = make([]KeyItem, 0, len(m.GetItems()))
	for _, it := range m.GetItems() {
		var owner *uint64
		if it.OwnerUserId != nil {
			v := it.GetOwnerUserId()
			owner = &v
		}
		var bst *string
		if it.BindSubjectType != nil {
			v := it.GetBindSubjectType()
			bst = &v
		}
		var bsid *uint64
		if it.BindSubjectId != nil {
			v := it.GetBindSubjectId()
			bsid = &v
		}
		var exp *int64
		if it.ExpiresAtSec != nil {
			v := it.GetExpiresAtSec()
			exp = &v
		}
		var mx *int32
		if it.MaxUses != nil {
			v := it.GetMaxUses()
			mx = &v
		}
		var rem *int32
		if it.RemainingUses != nil {
			v := it.GetRemainingUses()
			rem = &v
		}
		var issuedBy *uint64
		if it.IssuedBy != nil {
			v := it.GetIssuedBy()
			issuedBy = &v
		}
		items = append(items, KeyItem{
			ID:              it.GetId(),
			OwnerUserID:     owner,
			BindSubjectType: bst,
			BindSubjectID:   bsid,
			SecretHash:      it.GetSecretHash(),
			ExpiresAtSec:    exp,
			MaxUses:         mx,
			RemainingUses:   rem,
			Revoked:         it.GetRevoked(),
			IssuedBy:        issuedBy,
			IssuedAtSec:     it.GetIssuedAtSec(),
			Meta:            append([]byte(nil), it.GetMeta()...),
		})
	}
	return
}

// KeyCreateReq: {user_key:str, bitmap: bindType(bit0), bindId(bit1), expiresAtSec(bit2), maxUses(bit3), meta(bit4); fields..., nodes:[str]}
func EncodeKeyCreateReq(userKey string, bindType *string, bindID *uint64, expiresAtSec *int64, maxUses *int32, meta []byte, nodes []string) []byte {
	m := &pb.KeyCreateReq{UserKey: userKey, Nodes: append([]string(nil), nodes...)}
	if bindType != nil {
		v := *bindType
		m.BindSubjectType = &v
	}
	if bindID != nil {
		v := *bindID
		m.BindSubjectId = &v
	}
	if expiresAtSec != nil {
		v := *expiresAtSec
		m.ExpiresAtSec = &v
	}
	if maxUses != nil {
		v := *maxUses
		m.MaxUses = &v
	}
	m.Meta = append([]byte(nil), meta...)
	b, _ := proto.Marshal(m)
	return b
}
func DecodeKeyCreateReq(b []byte) (userKey string, bindType *string, bindID *uint64, expiresAtSec *int64, maxUses *int32, meta []byte, nodes []string, err error) {
	var m pb.KeyCreateReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", nil, nil, nil, nil, nil, nil, err
	}
	userKey = m.GetUserKey()
	if m.BindSubjectType != nil {
		v := m.GetBindSubjectType()
		bindType = &v
	}
	if m.BindSubjectId != nil {
		v := m.GetBindSubjectId()
		bindID = &v
	}
	if m.ExpiresAtSec != nil {
		v := m.GetExpiresAtSec()
		expiresAtSec = &v
	}
	if m.MaxUses != nil {
		v := m.GetMaxUses()
		maxUses = &v
	}
	meta = append([]byte(nil), m.GetMeta()...)
	nodes = append([]string(nil), m.GetNodes()...)
	return
}

// KeyCreateResp: {request_id:u64, secret:str, item:KeyItem, nodes:[str]}
func EncodeKeyCreateResp(requestID uint64, secret string, item KeyItem, nodes []string) []byte {
	// reuse EncodeKeyListResp mapping logic for KeyItem
	arr := []KeyItem{item}
	enc := EncodeKeyListResp(requestID, arr)
	// Decode then repackage with secret and nodes
	var kl pb.KeyListResp
	_ = proto.Unmarshal(enc, &kl)
	ki := kl.Items[0]
	m := &pb.KeyCreateResp{RequestId: requestID, Secret: secret, Item: ki, Nodes: append([]string(nil), nodes...)}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeKeyCreateResp(b []byte) (requestID uint64, secret string, item KeyItem, nodes []string, err error) {
	var m pb.KeyCreateResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, "", KeyItem{}, nil, err
	}
	requestID = m.GetRequestId()
	secret = m.GetSecret()
	// map KeyItem
	it := m.GetItem()
	if it != nil {
		var owner *uint64
		if it.OwnerUserId != nil {
			v := it.GetOwnerUserId()
			owner = &v
		}
		var bst *string
		if it.BindSubjectType != nil {
			v := it.GetBindSubjectType()
			bst = &v
		}
		var bsid *uint64
		if it.BindSubjectId != nil {
			v := it.GetBindSubjectId()
			bsid = &v
		}
		var exp *int64
		if it.ExpiresAtSec != nil {
			v := it.GetExpiresAtSec()
			exp = &v
		}
		var mx *int32
		if it.MaxUses != nil {
			v := it.GetMaxUses()
			mx = &v
		}
		var rem *int32
		if it.RemainingUses != nil {
			v := it.GetRemainingUses()
			rem = &v
		}
		var issuedBy *uint64
		if it.IssuedBy != nil {
			v := it.GetIssuedBy()
			issuedBy = &v
		}
		item = KeyItem{
			ID:              it.GetId(),
			OwnerUserID:     owner,
			BindSubjectType: bst,
			BindSubjectID:   bsid,
			SecretHash:      it.GetSecretHash(),
			ExpiresAtSec:    exp,
			MaxUses:         mx,
			RemainingUses:   rem,
			Revoked:         it.GetRevoked(),
			IssuedBy:        issuedBy,
			IssuedAtSec:     it.GetIssuedAtSec(),
			Meta:            append([]byte(nil), it.GetMeta()...),
		}
	}
	nodes = append([]string(nil), m.GetNodes()...)
	return
}

// KeyUpdateReq: {user_key:str, item:KeyItem}
func EncodeKeyUpdateReq(userKey string, item KeyItem) []byte {
	// map KeyItem to pb.KeyItem by reusing EncodeKeyListResp
	arr := []KeyItem{item}
	enc := EncodeKeyListResp(0, arr)
	var kl pb.KeyListResp
	_ = proto.Unmarshal(enc, &kl)
	ki := kl.Items[0]
	m := &pb.KeyUpdateReq{UserKey: userKey, Item: ki}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeKeyUpdateReq(b []byte) (userKey string, item KeyItem, err error) {
	var m pb.KeyUpdateReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", KeyItem{}, err
	}
	userKey = m.GetUserKey()
	it := m.GetItem()
	if it != nil {
		var owner *uint64
		if it.OwnerUserId != nil {
			v := it.GetOwnerUserId()
			owner = &v
		}
		var bst *string
		if it.BindSubjectType != nil {
			v := it.GetBindSubjectType()
			bst = &v
		}
		var bsid *uint64
		if it.BindSubjectId != nil {
			v := it.GetBindSubjectId()
			bsid = &v
		}
		var exp *int64
		if it.ExpiresAtSec != nil {
			v := it.GetExpiresAtSec()
			exp = &v
		}
		var mx *int32
		if it.MaxUses != nil {
			v := it.GetMaxUses()
			mx = &v
		}
		var rem *int32
		if it.RemainingUses != nil {
			v := it.GetRemainingUses()
			rem = &v
		}
		var issuedBy *uint64
		if it.IssuedBy != nil {
			v := it.GetIssuedBy()
			issuedBy = &v
		}
		item = KeyItem{
			ID:              it.GetId(),
			OwnerUserID:     owner,
			BindSubjectType: bst,
			BindSubjectID:   bsid,
			SecretHash:      it.GetSecretHash(),
			ExpiresAtSec:    exp,
			MaxUses:         mx,
			RemainingUses:   rem,
			Revoked:         it.GetRevoked(),
			IssuedBy:        issuedBy,
			IssuedAtSec:     it.GetIssuedAtSec(),
			Meta:            append([]byte(nil), it.GetMeta()...),
		}
	}
	return
}

// KeyDeleteReq: {user_key:str, id:u64}
func EncodeKeyDeleteReq(userKey string, id uint64) []byte {
	m := &pb.KeyDeleteReq{UserKey: userKey, Id: id}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeKeyDeleteReq(b []byte) (userKey string, id uint64, err error) {
	var m pb.KeyDeleteReq
	if err = proto.Unmarshal(b, &m); err != nil {
		return "", 0, err
	}
	return m.GetUserKey(), m.GetId(), nil
}

// KeyDevicesReq: {user_key:str}
func EncodeKeyDevicesReq(userKey string) []byte {
	m := &pb.KeyDevicesReq{UserKey: userKey}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeKeyDevicesReq(b []byte) (string, error) {
	var m pb.KeyDevicesReq
	if err := proto.Unmarshal(b, &m); err != nil {
		return "", err
	}
	return m.GetUserKey(), nil
}

// KeyDevicesResp: {request_id:u64, count:varint, devices:[DeviceItem]}
func EncodeKeyDevicesResp(requestID uint64, devices []DeviceItem) []byte {
	items := make([]*pb.DeviceItem, 0, len(devices))
	for _, d := range devices {
		items = append(items, toPBDeviceItem(d))
	}
	m := &pb.KeyDevicesResp{RequestId: requestID, Devices: items}
	b, _ := proto.Marshal(m)
	return b
}
func DecodeKeyDevicesResp(b []byte) (requestID uint64, devices []DeviceItem, err error) {
	var m pb.KeyDevicesResp
	if err = proto.Unmarshal(b, &m); err != nil {
		return 0, nil, err
	}
	requestID = m.GetRequestId()
	devices = make([]DeviceItem, 0, len(m.GetDevices()))
	for _, it := range m.GetDevices() {
		devices = append(devices, fromPBDeviceItem(it))
	}
	return
}
