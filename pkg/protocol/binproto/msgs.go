package binproto

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
	w := NewWriter(64)
	w.WriteU64(uint64(requestID))
	w.WriteI32(code)
	w.WriteLen16(len(message))
	w.WriteBytes(message)
	return w.Bytes()
}

// DecodeOKResp decodes payload of OK_RESP
func DecodeOKResp(b []byte) (requestID uint64, code int32, message []byte, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		return 0, 0, nil, e
	} else {
		requestID = v
	}
	if v, e := r.ReadI32(); e != nil {
		return 0, 0, nil, e
	} else {
		code = v
	}
	ln, e := r.ReadLen16()
	if e != nil {
		return 0, 0, nil, e
	}
	p, e := r.Read(ln)
	if e != nil {
		return 0, 0, nil, e
	}
	message = append([]byte(nil), p...)
	return
}

// EncodeErrResp mirrors OK but used for TypeErrResp
func EncodeErrResp(requestID uint64, code int32, message []byte) []byte {
	return EncodeOKResp(requestID, code, message)
}

// DecodeErrResp mirrors OK
func DecodeErrResp(b []byte) (uint64, int32, []byte, error) { return DecodeOKResp(b) }

// ManagerAuth: Req {token:len16+utf8}
func EncodeManagerAuthReq(token string) []byte {
	w := NewWriter(64)
	bs := []byte(token)
	w.WriteLen16(len(bs))
	w.WriteBytes(bs)
	return w.Bytes()
}

func DecodeManagerAuthReq(b []byte) (string, error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		return "", e
	}
	p, e := r.Read(ln)
	if e != nil {
		return "", e
	}
	return string(p), nil
}

// ManagerAuth: Resp {request_id:u64, device_uid:u64, role(len16+utf8 optional, 0 长度视为缺省)}
func EncodeManagerAuthResp(requestID, deviceUID uint64, role string) []byte {
	w := NewWriter(64)
	w.WriteU64(requestID)
	w.WriteU64(deviceUID)
	if role != "" {
		w.WriteLen16(len(role))
		w.WriteBytes([]byte(role))
	} else {
		w.WriteLen16(0)
	}
	return w.Bytes()
}

func DecodeManagerAuthResp(b []byte) (reqID, deviceUID uint64, role string, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		reqID = v
	}
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		deviceUID = v
	}
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if ln > 0 {
		if p, e2 := r.Read(ln); e2 == nil {
			role = string(p)
		} else {
			err = e2
			return
		}
	}
	return
}

// ParentAuthReq: {version:u8, ts:i64(ms), nonce:16B, hardware_id:len16+str, caps:len16+str, sig:32B}
func EncodeParentAuthReq(version uint8, ts int64, nonce [16]byte, hardwareID, caps string, sig [32]byte) []byte {
	w := NewWriter(128)
	w.WriteBytes([]byte{version})
	w.WriteI64(ts)
	w.WriteBytes(nonce[:])
	hb := []byte(hardwareID)
	w.WriteLen16(len(hb))
	w.WriteBytes(hb)
	cb := []byte(caps)
	w.WriteLen16(len(cb))
	w.WriteBytes(cb)
	w.WriteBytes(sig[:])
	return w.Bytes()
}

func DecodeParentAuthReq(b []byte) (version uint8, ts int64, nonce [16]byte, hardwareID, caps string, sig [32]byte, err error) {
	r := NewReader(b)
	if bb, e := r.Read(1); e != nil {
		err = e
		return
	} else {
		version = bb[0]
	}
	if v, e := r.ReadI64(); e != nil {
		err = e
		return
	} else {
		ts = v
	}
	if nb, e := r.Read(16); e != nil {
		err = e
		return
	} else {
		copy(nonce[:], nb)
	}
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	hb, e := r.Read(ln)
	if e != nil {
		err = e
		return
	}
	hardwareID = string(hb)
	ln, e = r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	cb, e := r.Read(ln)
	if e != nil {
		err = e
		return
	}
	caps = string(cb)
	if sb, e := r.Read(32); e != nil {
		err = e
		return
	} else {
		copy(sig[:], sb)
	}
	return
}

// ParentAuthResp: {request_id:u64, device_uid:u64, session_id:16B, heartbeat_sec:u16, perms:[len16+str], exp:i64, sig:32B}
func EncodeParentAuthResp(requestID, deviceUID uint64, sessionID [16]byte, heartbeatSec uint16, perms []string, exp int64, sig [32]byte) []byte {
	w := NewWriter(128)
	w.WriteU64(requestID)
	w.WriteU64(deviceUID)
	w.WriteBytes(sessionID[:])
	w.WriteU16(heartbeatSec)
	w.WriteU16(uint16(len(perms)))
	for _, p := range perms {
		pb := []byte(p)
		w.WriteLen16(len(pb))
		w.WriteBytes(pb)
	}
	w.WriteI64(exp)
	w.WriteBytes(sig[:])
	return w.Bytes()
}

func DecodeParentAuthResp(b []byte) (requestID, deviceUID uint64, sessionID [16]byte, heartbeatSec uint16, perms []string, exp int64, sig [32]byte, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		requestID = v
	}
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		deviceUID = v
	}
	if sb, e := r.Read(16); e != nil {
		err = e
		return
	} else {
		copy(sessionID[:], sb)
	}
	if v, e := r.ReadU16(); e != nil {
		err = e
		return
	} else {
		heartbeatSec = v
	}
	n, e := r.ReadU16()
	if e != nil {
		err = e
		return
	}
	perms = make([]string, 0, int(n))
	for i := 0; i < int(n); i++ {
		ln, e := r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		pb, e := r.Read(ln)
		if e != nil {
			err = e
			return
		}
		perms = append(perms, string(pb))
	}
	if v, e := r.ReadI64(); e != nil {
		err = e
		return
	} else {
		exp = v
	}
	if sb, e := r.Read(32); e != nil {
		err = e
		return
	} else {
		copy(sig[:], sb)
	}
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

// EncodeUserListResp {request_id:u64, count:u32, items:[UserItem]}
func EncodeUserListResp(requestID uint64, items []UserItem) []byte {
	w := NewWriter(128 + len(items)*64)
	w.WriteU64(requestID)
	w.WriteU32(uint32(len(items)))
	for _, u := range items {
		w.WriteU64(u.ID)
		ub := []byte(u.Username)
		w.WriteLen16(len(ub))
		w.WriteBytes(ub)
		db := []byte(u.DisplayName)
		w.WriteLen16(len(db))
		w.WriteBytes(db)
		if u.Disabled {
			w.WriteBytes([]byte{1})
		} else {
			w.WriteBytes([]byte{0})
		}
		w.WriteI64(u.CreatedAtSec)
		w.WriteI64(u.UpdatedAtSec)
	}
	return w.Bytes()
}

// DecodeUserListResp -> (request_id, items)
func DecodeUserListResp(b []byte) (requestID uint64, items []UserItem, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		return 0, nil, e
	} else {
		requestID = v
	}
	n32, e := r.ReadU32()
	if e != nil {
		return 0, nil, e
	}
	n := int(n32)
	items = make([]UserItem, 0, n)
	for i := 0; i < n; i++ {
		var it UserItem
		if v, e := r.ReadU64(); e != nil {
			return 0, nil, e
		} else {
			it.ID = v
		}
		ln, e := r.ReadLen16()
		if e != nil {
			return 0, nil, e
		}
		ub, e := r.Read(ln)
		if e != nil {
			return 0, nil, e
		}
		it.Username = string(ub)
		ln, e = r.ReadLen16()
		if e != nil {
			return 0, nil, e
		}
		db, e := r.Read(ln)
		if e != nil {
			return 0, nil, e
		}
		it.DisplayName = string(db)
		if bb, e := r.Read(1); e != nil {
			return 0, nil, e
		} else {
			it.Disabled = bb[0] != 0
		}
		if v, e := r.ReadI64(); e != nil {
			return 0, nil, e
		} else {
			it.CreatedAtSec = v
		}
		if v, e := r.ReadI64(); e != nil {
			return 0, nil, e
		} else {
			it.UpdatedAtSec = v
		}
		items = append(items, it)
	}
	return
}

// EncodeUserCreateReq {user_key:str, username:str, display_name:str, password:str}
func EncodeUserCreateReq(userKey, username, displayName, password string) []byte {
	w := NewWriter(128)
	uk := []byte(userKey)
	w.WriteLen16(len(uk))
	w.WriteBytes(uk)
	un := []byte(username)
	w.WriteLen16(len(un))
	w.WriteBytes(un)
	dn := []byte(displayName)
	w.WriteLen16(len(dn))
	w.WriteBytes(dn)
	pw := []byte(password)
	w.WriteLen16(len(pw))
	w.WriteBytes(pw)
	return w.Bytes()
}

func DecodeUserCreateReq(b []byte) (userKey, username, displayName, password string, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		return "", "", "", "", e
	}
	uk, e := r.Read(ln)
	if e != nil {
		return "", "", "", "", e
	}
	ln, e = r.ReadLen16()
	if e != nil {
		return "", "", "", "", e
	}
	un, e := r.Read(ln)
	if e != nil {
		return "", "", "", "", e
	}
	ln, e = r.ReadLen16()
	if e != nil {
		return "", "", "", "", e
	}
	dn, e := r.Read(ln)
	if e != nil {
		return "", "", "", "", e
	}
	ln, e = r.ReadLen16()
	if e != nil {
		return "", "", "", "", e
	}
	pw, e := r.Read(ln)
	if e != nil {
		return "", "", "", "", e
	}
	return string(uk), string(un), string(dn), string(pw), nil
}

// EncodeUserCreateResp {request_id:u64, id:u64}
func EncodeUserCreateResp(requestID, id uint64) []byte {
	w := NewWriter(32)
	w.WriteU64(requestID)
	w.WriteU64(id)
	return w.Bytes()
}

func DecodeUserCreateResp(b []byte) (requestID, id uint64, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		return 0, 0, e
	} else {
		requestID = v
	}
	if v, e := r.ReadU64(); e != nil {
		return 0, 0, e
	} else {
		id = v
	}
	return
}

// EncodeUserUpdateReq {user_key:str, id:u64, bitmap: display_name(bit0), password(bit1), disabled(bit2); fields}
func EncodeUserUpdateReq(userKey string, id uint64, displayName *string, password *string, disabled *bool) []byte {
	w := NewWriter(128)
	uk := []byte(userKey)
	w.WriteLen16(len(uk))
	w.WriteBytes(uk)
	w.WriteU64(id)
	var bm uint8
	if displayName != nil {
		bm |= 1 << 0
	}
	if password != nil {
		bm |= 1 << 1
	}
	if disabled != nil {
		bm |= 1 << 2
	}
	w.WriteBytes([]byte{bm})
	if displayName != nil {
		b := []byte(*displayName)
		w.WriteLen16(len(b))
		w.WriteBytes(b)
	}
	if password != nil {
		b := []byte(*password)
		w.WriteLen16(len(b))
		w.WriteBytes(b)
	}
	if disabled != nil {
		if *disabled {
			w.WriteBytes([]byte{1})
		} else {
			w.WriteBytes([]byte{0})
		}
	}
	return w.Bytes()
}

func DecodeUserUpdateReq(b []byte) (userKey string, id uint64, displayName *string, password *string, disabled *bool, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	uk, e := r.Read(ln)
	if e != nil {
		err = e
		return
	}
	userKey = string(uk)
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		id = v
	}
	bb, e := r.Read(1)
	if e != nil {
		err = e
		return
	}
	bm := bb[0]
	if bm&(1<<0) != 0 {
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		b, e := r.Read(ln)
		if e != nil {
			err = e
			return
		}
		s := string(b)
		displayName = &s
	}
	if bm&(1<<1) != 0 {
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		b, e := r.Read(ln)
		if e != nil {
			err = e
			return
		}
		s := string(b)
		password = &s
	}
	if bm&(1<<2) != 0 {
		b, e := r.Read(1)
		if e != nil {
			err = e
			return
		}
		d := b[0] != 0
		disabled = &d
	}
	return
}

// EncodeUserDeleteReq {user_key:str, id:u64}
func EncodeUserDeleteReq(userKey string, id uint64) []byte {
	w := NewWriter(64)
	uk := []byte(userKey)
	w.WriteLen16(len(uk))
	w.WriteBytes(uk)
	w.WriteU64(id)
	return w.Bytes()
}

func DecodeUserDeleteReq(b []byte) (userKey string, id uint64, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		return "", 0, e
	}
	uk, e := r.Read(ln)
	if e != nil {
		return "", 0, e
	}
	if v, e := r.ReadU64(); e != nil {
		return "", 0, e
	} else {
		return string(uk), v, nil
	}
}

// PermissionItem 权限节点条目
type PermissionItem struct{ Node string }

// EncodeUserPermListResp {request_id:u64, count:u32, items:[len16+utf8]}
func EncodeUserPermListResp(requestID uint64, items []PermissionItem) []byte {
	w := NewWriter(64 + len(items)*16)
	w.WriteU64(requestID)
	w.WriteU32(uint32(len(items)))
	for _, it := range items {
		b := []byte(it.Node)
		w.WriteLen16(len(b))
		w.WriteBytes(b)
	}
	return w.Bytes()
}

func DecodeUserPermListResp(b []byte) (requestID uint64, items []PermissionItem, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		return 0, nil, e
	} else {
		requestID = v
	}
	n32, e := r.ReadU32()
	if e != nil {
		return 0, nil, e
	}
	n := int(n32)
	items = make([]PermissionItem, 0, n)
	for i := 0; i < n; i++ {
		ln, e := r.ReadLen16()
		if e != nil {
			return 0, nil, e
		}
		b, e := r.Read(ln)
		if e != nil {
			return 0, nil, e
		}
		items = append(items, PermissionItem{Node: string(b)})
	}
	return
}

// EncodeUserPermListReq {user_key:str, user_id:u64}
func EncodeUserPermListReq(userKey string, userID uint64) []byte {
	w := NewWriter(64)
	uk := []byte(userKey)
	w.WriteLen16(len(uk))
	w.WriteBytes(uk)
	w.WriteU64(userID)
	return w.Bytes()
}

func DecodeUserPermListReq(b []byte) (userKey string, userID uint64, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		return "", 0, e
	}
	uk, e := r.Read(ln)
	if e != nil {
		return "", 0, e
	}
	if v, e := r.ReadU64(); e != nil {
		return "", 0, e
	} else {
		return string(uk), v, nil
	}
}

// EncodeUserPermAddReq {user_key:str, user_id:u64, node:str}
func EncodeUserPermAddReq(userKey string, userID uint64, node string) []byte {
	w := NewWriter(96)
	uk := []byte(userKey)
	w.WriteLen16(len(uk))
	w.WriteBytes(uk)
	w.WriteU64(userID)
	nb := []byte(node)
	w.WriteLen16(len(nb))
	w.WriteBytes(nb)
	return w.Bytes()
}

func DecodeUserPermAddReq(b []byte) (userKey string, userID uint64, node string, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		return "", 0, "", e
	}
	uk, e := r.Read(ln)
	if e != nil {
		return "", 0, "", e
	}
	if v, e := r.ReadU64(); e != nil {
		return "", 0, "", e
	} else {
		userID = v
	}
	ln, e = r.ReadLen16()
	if e != nil {
		return "", 0, "", e
	}
	nb, e := r.Read(ln)
	if e != nil {
		return "", 0, "", e
	}
	return string(uk), userID, string(nb), nil
}

// EncodeUserPermRemoveReq 同 Add，仅语义不同
func EncodeUserPermRemoveReq(userKey string, userID uint64, node string) []byte {
	return EncodeUserPermAddReq(userKey, userID, node)
}
func DecodeUserPermRemoveReq(b []byte) (string, uint64, string, error) {
	return DecodeUserPermAddReq(b)
}

// EncodeUserSelfUpdateReq {user_key:str, display_name:str}
func EncodeUserSelfUpdateReq(userKey string, displayName string) []byte {
	w := NewWriter(96)
	uk := []byte(userKey)
	w.WriteLen16(len(uk))
	w.WriteBytes(uk)
	dn := []byte(displayName)
	w.WriteLen16(len(dn))
	w.WriteBytes(dn)
	return w.Bytes()
}

func DecodeUserSelfUpdateReq(b []byte) (userKey string, displayName string, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		return "", "", e
	}
	uk, e := r.Read(ln)
	if e != nil {
		return "", "", e
	}
	ln, e = r.ReadLen16()
	if e != nil {
		return "", "", e
	}
	dn, e := r.Read(ln)
	if e != nil {
		return "", "", e
	}
	return string(uk), string(dn), nil
}

// EncodeUserSelfPasswordReq {user_key:str, old_password:str, new_password:str}
func EncodeUserSelfPasswordReq(userKey, oldPassword, newPassword string) []byte {
	w := NewWriter(128)
	uk := []byte(userKey)
	w.WriteLen16(len(uk))
	w.WriteBytes(uk)
	op := []byte(oldPassword)
	w.WriteLen16(len(op))
	w.WriteBytes(op)
	np := []byte(newPassword)
	w.WriteLen16(len(np))
	w.WriteBytes(np)
	return w.Bytes()
}

func DecodeUserSelfPasswordReq(b []byte) (userKey, oldPassword, newPassword string, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		return "", "", "", e
	}
	uk, e := r.Read(ln)
	if e != nil {
		return "", "", "", e
	}
	ln, e = r.ReadLen16()
	if e != nil {
		return "", "", "", e
	}
	op, e := r.Read(ln)
	if e != nil {
		return "", "", "", e
	}
	ln, e = r.ReadLen16()
	if e != nil {
		return "", "", "", e
	}
	np, e := r.Read(ln)
	if e != nil {
		return "", "", "", e
	}
	return string(uk), string(op), string(np), nil
}

// ========== User/Auth ==========
// UserLoginReq {username:str, password:str}
func EncodeUserLoginReq(username, password string) []byte {
	w := NewWriter(64)
	ub, pb := []byte(username), []byte(password)
	w.WriteLen16(len(ub))
	w.WriteBytes(ub)
	w.WriteLen16(len(pb))
	w.WriteBytes(pb)
	return w.Bytes()
}
func DecodeUserLoginReq(b []byte) (username, password string, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	u, e := r.Read(ln)
	if e != nil {
		err = e
		return
	}
	ln, e = r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	p, e := r.Read(ln)
	if e != nil {
		err = e
		return
	}
	return string(u), string(p), nil
}

// UserLoginResp {request_id:u64, token:str, key_id:u64, user_id:u64, username:str, display_name:str, perms:[str]}
func EncodeUserLoginResp(requestID, keyID, userID uint64, token, username, displayName string, perms []string) []byte {
	w := NewWriter(256)
	w.WriteU64(requestID)
	tb := []byte(token)
	w.WriteLen16(len(tb))
	w.WriteBytes(tb)
	w.WriteU64(keyID)
	w.WriteU64(userID)
	ub := []byte(username)
	w.WriteLen16(len(ub))
	w.WriteBytes(ub)
	db := []byte(displayName)
	w.WriteLen16(len(db))
	w.WriteBytes(db)
	w.WriteVarint(uint64(len(perms)))
	for _, p := range perms {
		pb := []byte(p)
		w.WriteLen16(len(pb))
		w.WriteBytes(pb)
	}
	return w.Bytes()
}
func DecodeUserLoginResp(b []byte) (requestID, keyID, userID uint64, token, username, displayName string, perms []string, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		requestID = v
	}
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if tb, e2 := r.Read(ln); e2 != nil {
		err = e2
		return
	} else {
		token = string(tb)
	}
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		keyID = v
	}
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		userID = v
	}
	ln, e = r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if ub, e2 := r.Read(ln); e2 != nil {
		err = e2
		return
	} else {
		username = string(ub)
	}
	ln, e = r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if db, e2 := r.Read(ln); e2 != nil {
		err = e2
		return
	} else {
		displayName = string(db)
	}
	cnt, e := r.ReadVarint()
	if e != nil {
		err = e
		return
	}
	perms = make([]string, 0, int(cnt))
	for i := 0; i < int(cnt); i++ {
		ln, e := r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		pb, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		perms = append(perms, string(pb))
	}
	return
}

// UserMeReq {user_key:str}
func EncodeUserMeReq(userKey string) []byte {
	w := NewWriter(32)
	b := []byte(userKey)
	w.WriteLen16(len(b))
	w.WriteBytes(b)
	return w.Bytes()
}
func DecodeUserMeReq(b []byte) (string, error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		return "", e
	}
	p, e := r.Read(ln)
	if e != nil {
		return "", e
	}
	return string(p), nil
}

// UserMeResp {request_id:u64, user_id:u64, username:str, display_name:str, perms:[str]}
func EncodeUserMeResp(requestID, userID uint64, username, displayName string, perms []string) []byte {
	w := NewWriter(128)
	w.WriteU64(requestID)
	w.WriteU64(userID)
	ub := []byte(username)
	w.WriteLen16(len(ub))
	w.WriteBytes(ub)
	db := []byte(displayName)
	w.WriteLen16(len(db))
	w.WriteBytes(db)
	w.WriteVarint(uint64(len(perms)))
	for _, p := range perms {
		pb := []byte(p)
		w.WriteLen16(len(pb))
		w.WriteBytes(pb)
	}
	return w.Bytes()
}
func DecodeUserMeResp(b []byte) (requestID, userID uint64, username, displayName string, perms []string, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		requestID = v
	}
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		userID = v
	}
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if ub, e2 := r.Read(ln); e2 != nil {
		err = e2
		return
	} else {
		username = string(ub)
	}
	ln, e = r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if db, e2 := r.Read(ln); e2 != nil {
		err = e2
		return
	} else {
		displayName = string(db)
	}
	cnt, e := r.ReadVarint()
	if e != nil {
		err = e
		return
	}
	perms = make([]string, 0, int(cnt))
	for i := 0; i < int(cnt); i++ {
		ln, e := r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		pb, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		perms = append(perms, string(pb))
	}
	return
}

// UserLogoutReq {user_key:str}
func EncodeUserLogoutReq(userKey string) []byte {
	w := NewWriter(32)
	b := []byte(userKey)
	w.WriteLen16(len(b))
	w.WriteBytes(b)
	return w.Bytes()
}
func DecodeUserLogoutReq(b []byte) (string, error) { return DecodeUserMeReq(b) }

// ========== System Log ==========
// SystemLogListReq {user_key:str, level?:str, source?:str, keyword?:str, start_at:i64, end_at:i64, page:i32, page_size:i32}
func EncodeSystemLogListReq(userKey, level, source, keyword string, startAt, endAt int64, page, pageSize int32) []byte {
	w := NewWriter(128)
	// bitmap: level(0), source(1), keyword(2)
	var bm byte
	if level != "" {
		bm |= 0x01
	}
	if source != "" {
		bm |= 0x02
	}
	if keyword != "" {
		bm |= 0x04
	}
	w.WriteBytes([]byte{bm})
	uk := []byte(userKey)
	w.WriteLen16(len(uk))
	w.WriteBytes(uk)
	if bm&0x01 != 0 {
		lb := []byte(level)
		w.WriteLen16(len(lb))
		w.WriteBytes(lb)
	}
	if bm&0x02 != 0 {
		sb := []byte(source)
		w.WriteLen16(len(sb))
		w.WriteBytes(sb)
	}
	if bm&0x04 != 0 {
		kb := []byte(keyword)
		w.WriteLen16(len(kb))
		w.WriteBytes(kb)
	}
	w.WriteI64(startAt)
	w.WriteI64(endAt)
	w.WriteI32(page)
	w.WriteI32(pageSize)
	return w.Bytes()
}

type SystemLogItem struct {
	Level, Source, Message, Details string
	At                              int64
}

// SystemLogListResp {request_id:u64, total:i64, page:i32, page_size:i32, logs:[{level,source,message,details,at}]}
func EncodeSystemLogListResp(requestID uint64, total int64, page, pageSize int32, logs []SystemLogItem) []byte {
	w := NewWriter(512)
	w.WriteU64(requestID)
	w.WriteI64(total)
	w.WriteI32(page)
	w.WriteI32(pageSize)
	w.WriteVarint(uint64(len(logs)))
	for _, lg := range logs {
		lb := []byte(lg.Level)
		w.WriteLen16(len(lb))
		w.WriteBytes(lb)
		sb := []byte(lg.Source)
		w.WriteLen16(len(sb))
		w.WriteBytes(sb)
		mb := []byte(lg.Message)
		w.WriteLen16(len(mb))
		w.WriteBytes(mb)
		db := []byte(lg.Details)
		w.WriteLen16(len(db))
		w.WriteBytes(db)
		w.WriteI64(lg.At)
	}
	return w.Bytes()
}
func DecodeSystemLogListReq(b []byte) (userKey, level, source, keyword string, startAt, endAt int64, page, pageSize int32, err error) {
	r := NewReader(b)
	bmRaw, e := r.Read(1)
	if e != nil {
		err = e
		return
	}
	bm := bmRaw[0]
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if uk, e2 := r.Read(ln); e2 != nil {
		err = e2
		return
	} else {
		userKey = string(uk)
	}
	if bm&0x01 != 0 {
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		p, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		level = string(p)
	}
	if bm&0x02 != 0 {
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		p, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		source = string(p)
	}
	if bm&0x04 != 0 {
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		p, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		keyword = string(p)
	}
	if v, e := r.ReadI64(); e != nil {
		err = e
		return
	} else {
		startAt = v
	}
	if v, e := r.ReadI64(); e != nil {
		err = e
		return
	} else {
		endAt = v
	}
	if v, e := r.ReadI32(); e != nil {
		err = e
		return
	} else {
		page = v
	}
	if v, e := r.ReadI32(); e != nil {
		err = e
		return
	} else {
		pageSize = v
	}
	return
}
func DecodeSystemLogListResp(b []byte) (requestID uint64, total int64, page, pageSize int32, logs []SystemLogItem, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		requestID = v
	}
	if v, e := r.ReadI64(); e != nil {
		err = e
		return
	} else {
		total = v
	}
	if v, e := r.ReadI32(); e != nil {
		err = e
		return
	} else {
		page = v
	}
	if v, e := r.ReadI32(); e != nil {
		err = e
		return
	} else {
		pageSize = v
	}
	cnt, e := r.ReadVarint()
	if e != nil {
		err = e
		return
	}
	logs = make([]SystemLogItem, 0, int(cnt))
	for i := 0; i < int(cnt); i++ {
		ln, e := r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		lb, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		sb, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		mb, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		db, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		at, e := r.ReadI64()
		if e != nil {
			err = e
			return
		}
		logs = append(logs, SystemLogItem{Level: string(lb), Source: string(sb), Message: string(mb), Details: string(db), At: at})
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
}

// EncodeQueryNodesReq {bitmap(1)=user_key(bit0), user_key?:str}
func EncodeQueryNodesReq(userKey string) []byte {
	w := NewWriter(32)
	var bm byte
	if userKey != "" {
		bm |= 0x01
	}
	w.WriteBytes([]byte{bm})
	if bm&0x01 != 0 {
		ub := []byte(userKey)
		w.WriteLen16(len(ub))
		w.WriteBytes(ub)
	}
	return w.Bytes()
}

// DecodeQueryNodesReq -> userKey (optional)
func DecodeQueryNodesReq(b []byte) (string, error) {
	r := NewReader(b)
	p, e := r.Read(1)
	if e != nil {
		return "", e
	}
	bm := p[0]
	if bm&0x01 != 0 {
		ln, e := r.ReadLen16()
		if e != nil {
			return "", e
		}
		ub, e := r.Read(ln)
		if e != nil {
			return "", e
		}
		return string(ub), nil
	}
	return "", nil
}

// helper: encode one DeviceItem with inner bitmap for optional fields
func encodeDeviceItem(w *Writer, d DeviceItem) {
	// bitmap: parentID(bit0), ownerUserID(bit1), lastSeen(bit2)
	var bm byte
	if d.ParentID != nil {
		bm |= 0x01
	}
	if d.OwnerUserID != nil {
		bm |= 0x02
	}
	if d.LastSeenSec != nil {
		bm |= 0x04
	}
	w.WriteBytes([]byte{bm})
	w.WriteU64(d.ID)
	w.WriteU64(d.DeviceUID)
	hb := []byte(d.HardwareID)
	w.WriteLen16(len(hb))
	w.WriteBytes(hb)
	rb := []byte(d.Role)
	w.WriteLen16(len(rb))
	w.WriteBytes(rb)
	nb := []byte(d.Name)
	w.WriteLen16(len(nb))
	w.WriteBytes(nb)
	if bm&0x01 != 0 {
		w.WriteU64(*d.ParentID)
	}
	if bm&0x02 != 0 {
		w.WriteU64(*d.OwnerUserID)
	}
	if bm&0x04 != 0 {
		w.WriteI64(*d.LastSeenSec)
	}
	w.WriteI64(d.CreatedAtSec)
	w.WriteI64(d.UpdatedAtSec)
}

func decodeDeviceItem(r *Reader) (DeviceItem, error) {
	var d DeviceItem
	p, e := r.Read(1)
	if e != nil {
		return d, e
	}
	bm := p[0]
	if v, e := r.ReadU64(); e != nil {
		return d, e
	} else {
		d.ID = v
	}
	if v, e := r.ReadU64(); e != nil {
		return d, e
	} else {
		d.DeviceUID = v
	}
	ln, e := r.ReadLen16()
	if e != nil {
		return d, e
	}
	if hb, e2 := r.Read(ln); e2 != nil {
		return d, e2
	} else {
		d.HardwareID = string(hb)
	}
	ln, e = r.ReadLen16()
	if e != nil {
		return d, e
	}
	if rb, e2 := r.Read(ln); e2 != nil {
		return d, e2
	} else {
		d.Role = string(rb)
	}
	ln, e = r.ReadLen16()
	if e != nil {
		return d, e
	}
	if nb, e2 := r.Read(ln); e2 != nil {
		return d, e2
	} else {
		d.Name = string(nb)
	}
	if bm&0x01 != 0 {
		if v, e := r.ReadU64(); e != nil {
			return d, e
		} else {
			d.ParentID = &v
		}
	}
	if bm&0x02 != 0 {
		if v, e := r.ReadU64(); e != nil {
			return d, e
		} else {
			d.OwnerUserID = &v
		}
	}
	if bm&0x04 != 0 {
		if v, e := r.ReadI64(); e != nil {
			return d, e
		} else {
			d.LastSeenSec = &v
		}
	}
	if v, e := r.ReadI64(); e != nil {
		return d, e
	} else {
		d.CreatedAtSec = v
	}
	if v, e := r.ReadI64(); e != nil {
		return d, e
	} else {
		d.UpdatedAtSec = v
	}
	return d, nil
}

// EncodeQueryNodesResp {request_id:u64, count:varint, devices:[DeviceItem]}
func EncodeQueryNodesResp(requestID uint64, devices []DeviceItem) []byte {
	w := NewWriter(512)
	w.WriteU64(requestID)
	w.WriteVarint(uint64(len(devices)))
	for _, d := range devices {
		encodeDeviceItem(w, d)
	}
	return w.Bytes()
}

func DecodeQueryNodesResp(b []byte) (requestID uint64, devices []DeviceItem, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		requestID = v
	}
	cnt, e := r.ReadVarint()
	if e != nil {
		err = e
		return
	}
	devices = make([]DeviceItem, 0, int(cnt))
	for i := 0; i < int(cnt); i++ {
		d, e := decodeDeviceItem(r)
		if e != nil {
			err = e
			return
		}
		devices = append(devices, d)
	}
	return
}

// ========== Devices: Create/Update/Delete ==========
// Create/Update: {bitmap(1)=user_key(bit0), user_key?:str, device:DeviceItem}
func EncodeCreateDeviceReq(userKey string, d DeviceItem) []byte {
	w := NewWriter(256)
	var bm byte
	if userKey != "" {
		bm |= 0x01
	}
	w.WriteBytes([]byte{bm})
	if bm&0x01 != 0 {
		ub := []byte(userKey)
		w.WriteLen16(len(ub))
		w.WriteBytes(ub)
	}
	encodeDeviceItem(w, d)
	return w.Bytes()
}
func DecodeCreateDeviceReq(b []byte) (userKey string, d DeviceItem, err error) {
	r := NewReader(b)
	p, e := r.Read(1)
	if e != nil {
		err = e
		return
	}
	bm := p[0]
	if bm&0x01 != 0 {
		ln, e := r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		ub, e := r.Read(ln)
		if e != nil {
			err = e
			return
		}
		userKey = string(ub)
	}
	if d, e = decodeDeviceItem(r); e != nil {
		err = e
		return
	}
	return
}
func EncodeUpdateDeviceReq(userKey string, d DeviceItem) []byte {
	return EncodeCreateDeviceReq(userKey, d)
}
func DecodeUpdateDeviceReq(b []byte) (string, DeviceItem, error) { return DecodeCreateDeviceReq(b) }

// Delete: {bitmap(1)=user_key(bit0), id:u64, user_key?:str}
func EncodeDeleteDeviceReq(id uint64, userKey string) []byte {
	w := NewWriter(32)
	var bm byte
	if userKey != "" {
		bm |= 0x01
	}
	w.WriteBytes([]byte{bm})
	w.WriteU64(id)
	if bm&0x01 != 0 {
		ub := []byte(userKey)
		w.WriteLen16(len(ub))
		w.WriteBytes(ub)
	}
	return w.Bytes()
}
func DecodeDeleteDeviceReq(b []byte) (id uint64, userKey string, err error) {
	r := NewReader(b)
	p, e := r.Read(1)
	if e != nil {
		err = e
		return
	}
	bm := p[0]
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		id = v
	}
	if bm&0x01 != 0 {
		ln, e := r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		ub, e := r.Read(ln)
		if e != nil {
			err = e
			return
		}
		userKey = string(ub)
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
	w := NewWriter(32)
	var bm byte
	if userKey != "" {
		bm |= 0x01
	}
	if deviceUID != nil {
		bm |= 0x02
	}
	w.WriteBytes([]byte{bm})
	if bm&0x01 != 0 {
		kb := []byte(userKey)
		w.WriteLen16(len(kb))
		w.WriteBytes(kb)
	}
	if bm&0x02 != 0 {
		w.WriteU64(*deviceUID)
	}
	return w.Bytes()
}
func DecodeVarListReq(b []byte) (userKey string, deviceUID *uint64, err error) {
	r := NewReader(b)
	p, e := r.Read(1)
	if e != nil {
		err = e
		return
	}
	bm := p[0]
	if bm&0x01 != 0 {
		ln, e := r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		kb, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		userKey = string(kb)
	}
	if bm&0x02 != 0 {
		if v, e := r.ReadU64(); e != nil {
			err = e
			return
		} else {
			deviceUID = &v
		}
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
	w := NewWriter(512)
	w.WriteU64(requestID)
	w.WriteVarint(uint64(len(items)))
	for _, it := range items {
		w.WriteU64(it.ID)
		w.WriteU64(it.OwnerDeviceID)
		w.WriteU64(it.OwnerDeviceUID)
		nb := []byte(it.Name)
		w.WriteLen16(len(nb))
		w.WriteBytes(nb)
		w.WriteLen16(len(it.Value))
		w.WriteBytes(it.Value)
		w.WriteI64(it.CreatedAtSec)
		w.WriteI64(it.UpdatedAtSec)
	}
	return w.Bytes()
}
func DecodeVarListResp(b []byte) (requestID uint64, items []VarListItem, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		requestID = v
	}
	cnt, e := r.ReadVarint()
	if e != nil {
		err = e
		return
	}
	items = make([]VarListItem, 0, int(cnt))
	for i := 0; i < int(cnt); i++ {
		var it VarListItem
		if v, e := r.ReadU64(); e != nil {
			err = e
			return
		} else {
			it.ID = v
		}
		if v, e := r.ReadU64(); e != nil {
			err = e
			return
		} else {
			it.OwnerDeviceID = v
		}
		if v, e := r.ReadU64(); e != nil {
			err = e
			return
		} else {
			it.OwnerDeviceUID = v
		}
		ln, e := r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		if nb, e2 := r.Read(ln); e2 != nil {
			err = e2
			return
		} else {
			it.Name = string(nb)
		}
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		if vb, e2 := r.Read(ln); e2 != nil {
			err = e2
			return
		} else {
			it.Value = append([]byte(nil), vb...)
		}
		if v, e := r.ReadI64(); e != nil {
			err = e
			return
		} else {
			it.CreatedAtSec = v
		}
		if v, e := r.ReadI64(); e != nil {
			err = e
			return
		} else {
			it.UpdatedAtSec = v
		}
		items = append(items, it)
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
	w := NewWriter(512)
	kb := []byte(userKey)
	w.WriteLen16(len(kb))
	w.WriteBytes(kb)
	w.WriteVarint(uint64(len(items)))
	for _, it := range items {
		w.WriteU64(it.DeviceUID)
		nb := []byte(it.Name)
		w.WriteLen16(len(nb))
		w.WriteBytes(nb)
		w.WriteLen16(len(it.Value))
		w.WriteBytes(it.Value)
	}
	return w.Bytes()
}
func DecodeVarUpdateReq(b []byte) (userKey string, items []VarUpdateItem, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if ln > 0 {
		ub, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		userKey = string(ub)
	}
	cnt, e := r.ReadVarint()
	if e != nil {
		err = e
		return
	}
	items = make([]VarUpdateItem, 0, int(cnt))
	for i := 0; i < int(cnt); i++ {
		var it VarUpdateItem
		if v, e := r.ReadU64(); e != nil {
			err = e
			return
		} else {
			it.DeviceUID = v
		}
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		nb, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		it.Name = string(nb)
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		vb, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		it.Value = append([]byte(nil), vb...)
		items = append(items, it)
	}
	return
}

type VarDeleteItem struct {
	DeviceUID uint64
	Name      string
}

// VarDeleteReq {user_key:str (len16, 0 表示缺省), count:varint, items:[{device_uid:u64, name:len16+utf8}]}
func EncodeVarDeleteReq(userKey string, items []VarDeleteItem) []byte {
	w := NewWriter(256)
	kb := []byte(userKey)
	w.WriteLen16(len(kb))
	w.WriteBytes(kb)
	w.WriteVarint(uint64(len(items)))
	for _, it := range items {
		w.WriteU64(it.DeviceUID)
		nb := []byte(it.Name)
		w.WriteLen16(len(nb))
		w.WriteBytes(nb)
	}
	return w.Bytes()
}
func DecodeVarDeleteReq(b []byte) (userKey string, items []VarDeleteItem, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if ln > 0 {
		ub, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		userKey = string(ub)
	}
	cnt, e := r.ReadVarint()
	if e != nil {
		err = e
		return
	}
	items = make([]VarDeleteItem, 0, int(cnt))
	for i := 0; i < int(cnt); i++ {
		var it VarDeleteItem
		if v, e := r.ReadU64(); e != nil {
			err = e
			return
		} else {
			it.DeviceUID = v
		}
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		nb, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		it.Name = string(nb)
		items = append(items, it)
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
//  bit0 OwnerUserID
//  bit1 BindSubjectType
//  bit2 BindSubjectID
//  bit3 ExpiresAtSec
//  bit4 MaxUses
//  bit5 RemainingUses
//  bit6 IssuedBy
//  bit7 Meta
func encodeKeyItem(w *Writer, k KeyItem) {
	var bm byte
	if k.OwnerUserID != nil {
		bm |= 0x01
	}
	if k.BindSubjectType != nil {
		bm |= 0x02
	}
	if k.BindSubjectID != nil {
		bm |= 0x04
	}
	if k.ExpiresAtSec != nil {
		bm |= 0x08
	}
	if k.MaxUses != nil {
		bm |= 0x10
	}
	if k.RemainingUses != nil {
		bm |= 0x20
	}
	if k.IssuedBy != nil {
		bm |= 0x40
	}
	if len(k.Meta) > 0 {
		bm |= 0x80
	}
	w.WriteBytes([]byte{bm})
	w.WriteU64(k.ID)
	sh := []byte(k.SecretHash)
	w.WriteLen16(len(sh))
	w.WriteBytes(sh)
	if bm&0x01 != 0 {
		w.WriteU64(*k.OwnerUserID)
	}
	if bm&0x02 != 0 {
		b := []byte(*k.BindSubjectType)
		w.WriteLen16(len(b))
		w.WriteBytes(b)
	}
	if bm&0x04 != 0 {
		w.WriteU64(*k.BindSubjectID)
	}
	if bm&0x08 != 0 {
		w.WriteI64(*k.ExpiresAtSec)
	}
	if bm&0x10 != 0 {
		w.WriteI32(*k.MaxUses)
	}
	if bm&0x20 != 0 {
		w.WriteI32(*k.RemainingUses)
	}
	// revoked always present as 1 byte
	if k.Revoked {
		w.WriteBytes([]byte{1})
	} else {
		w.WriteBytes([]byte{0})
	}
	if bm&0x40 != 0 {
		w.WriteU64(*k.IssuedBy)
	}
	w.WriteI64(k.IssuedAtSec)
	if bm&0x80 != 0 {
		w.WriteLen16(len(k.Meta))
		w.WriteBytes(k.Meta)
	}
}

func decodeKeyItem(r *Reader) (KeyItem, error) {
	var k KeyItem
	p, e := r.Read(1)
	if e != nil {
		return k, e
	}
	bm := p[0]
	if v, e := r.ReadU64(); e != nil {
		return k, e
	} else {
		k.ID = v
	}
	ln, e := r.ReadLen16()
	if e != nil {
		return k, e
	}
	if sh, e2 := r.Read(ln); e2 != nil {
		return k, e2
	} else {
		k.SecretHash = string(sh)
	}
	if bm&0x01 != 0 {
		if v, e := r.ReadU64(); e != nil {
			return k, e
		} else {
			k.OwnerUserID = &v
		}
	}
	if bm&0x02 != 0 {
		ln, e = r.ReadLen16()
		if e != nil {
			return k, e
		}
		if b, e2 := r.Read(ln); e2 != nil {
			return k, e2
		} else {
			s := string(b)
			k.BindSubjectType = &s
		}
	}
	if bm&0x04 != 0 {
		if v, e := r.ReadU64(); e != nil {
			return k, e
		} else {
			k.BindSubjectID = &v
		}
	}
	if bm&0x08 != 0 {
		if v, e := r.ReadI64(); e != nil {
			return k, e
		} else {
			k.ExpiresAtSec = &v
		}
	}
	if bm&0x10 != 0 {
		if v, e := r.ReadI32(); e != nil {
			return k, e
		} else {
			k.MaxUses = &v
		}
	}
	if bm&0x20 != 0 {
		if v, e := r.ReadI32(); e != nil {
			return k, e
		} else {
			k.RemainingUses = &v
		}
	}
	// revoked
	if b, e := r.Read(1); e != nil {
		return k, e
	} else {
		k.Revoked = b[0] != 0
	}
	if bm&0x40 != 0 {
		if v, e := r.ReadU64(); e != nil {
			return k, e
		} else {
			k.IssuedBy = &v
		}
	}
	if v, e := r.ReadI64(); e != nil {
		return k, e
	} else {
		k.IssuedAtSec = v
	}
	if bm&0x80 != 0 {
		ln, e = r.ReadLen16()
		if e != nil {
			return k, e
		}
		if m, e2 := r.Read(ln); e2 != nil {
			return k, e2
		} else {
			k.Meta = append([]byte(nil), m...)
		}
	}
	return k, nil
}

// KeyListReq: {user_key:str}
func EncodeKeyListReq(userKey string) []byte {
	w := NewWriter(32)
	b := []byte(userKey)
	w.WriteLen16(len(b))
	w.WriteBytes(b)
	return w.Bytes()
}
func DecodeKeyListReq(b []byte) (string, error) { return DecodeUserMeReq(b) }

// KeyListResp: {request_id:u64, count:varint, items:[KeyItem]}
func EncodeKeyListResp(requestID uint64, items []KeyItem) []byte {
	w := NewWriter(512)
	w.WriteU64(requestID)
	w.WriteVarint(uint64(len(items)))
	for _, it := range items {
		encodeKeyItem(w, it)
	}
	return w.Bytes()
}
func DecodeKeyListResp(b []byte) (requestID uint64, items []KeyItem, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		requestID = v
	}
	cnt, e := r.ReadVarint()
	if e != nil {
		err = e
		return
	}
	items = make([]KeyItem, 0, int(cnt))
	for i := 0; i < int(cnt); i++ {
		it, e := decodeKeyItem(r)
		if e != nil {
			err = e
			return
		}
		items = append(items, it)
	}
	return
}

// KeyCreateReq: {user_key:str, bitmap: bindType(bit0), bindId(bit1), expiresAtSec(bit2), maxUses(bit3), meta(bit4); fields..., nodes:[str]}
func EncodeKeyCreateReq(userKey string, bindType *string, bindID *uint64, expiresAtSec *int64, maxUses *int32, meta []byte, nodes []string) []byte {
	w := NewWriter(256)
	b := []byte(userKey)
	w.WriteLen16(len(b))
	w.WriteBytes(b)
	var bm byte
	if bindType != nil {
		bm |= 0x01
	}
	if bindID != nil {
		bm |= 0x02
	}
	if expiresAtSec != nil {
		bm |= 0x04
	}
	if maxUses != nil {
		bm |= 0x08
	}
	if len(meta) > 0 {
		bm |= 0x10
	}
	w.WriteBytes([]byte{bm})
	if bm&0x01 != 0 {
		bt := []byte(*bindType)
		w.WriteLen16(len(bt))
		w.WriteBytes(bt)
	}
	if bm&0x02 != 0 {
		w.WriteU64(*bindID)
	}
	if bm&0x04 != 0 {
		w.WriteI64(*expiresAtSec)
	}
	if bm&0x08 != 0 {
		w.WriteI32(*maxUses)
	}
	if bm&0x10 != 0 {
		w.WriteLen16(len(meta))
		w.WriteBytes(meta)
	}
	w.WriteVarint(uint64(len(nodes)))
	for _, n := range nodes {
		nb := []byte(n)
		w.WriteLen16(len(nb))
		w.WriteBytes(nb)
	}
	return w.Bytes()
}
func DecodeKeyCreateReq(b []byte) (userKey string, bindType *string, bindID *uint64, expiresAtSec *int64, maxUses *int32, meta []byte, nodes []string, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if ub, e2 := r.Read(ln); e2 != nil {
		err = e2
		return
	} else {
		userKey = string(ub)
	}
	p, e := r.Read(1)
	if e != nil {
		err = e
		return
	}
	bm := p[0]
	if bm&0x01 != 0 {
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		if bt, e2 := r.Read(ln); e2 != nil {
			err = e2
			return
		} else {
			s := string(bt)
			bindType = &s
		}
	}
	if bm&0x02 != 0 {
		if v, e := r.ReadU64(); e != nil {
			err = e
			return
		} else {
			bindID = &v
		}
	}
	if bm&0x04 != 0 {
		if v, e := r.ReadI64(); e != nil {
			err = e
			return
		} else {
			expiresAtSec = &v
		}
	}
	if bm&0x08 != 0 {
		if v, e := r.ReadI32(); e != nil {
			err = e
			return
		} else {
			maxUses = &v
		}
	}
	if bm&0x10 != 0 {
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		if m, e2 := r.Read(ln); e2 != nil {
			err = e2
			return
		} else {
			meta = append([]byte(nil), m...)
		}
	}
	cnt, e := r.ReadVarint()
	if e != nil {
		err = e
		return
	}
	nodes = make([]string, 0, int(cnt))
	for i := 0; i < int(cnt); i++ {
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		nb, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		nodes = append(nodes, string(nb))
	}
	return
}

// KeyCreateResp: {request_id:u64, secret:str, item:KeyItem, nodes:[str]}
func EncodeKeyCreateResp(requestID uint64, secret string, item KeyItem, nodes []string) []byte {
	w := NewWriter(512)
	w.WriteU64(requestID)
	sb := []byte(secret)
	w.WriteLen16(len(sb))
	w.WriteBytes(sb)
	encodeKeyItem(w, item)
	w.WriteVarint(uint64(len(nodes)))
	for _, n := range nodes {
		nb := []byte(n)
		w.WriteLen16(len(nb))
		w.WriteBytes(nb)
	}
	return w.Bytes()
}
func DecodeKeyCreateResp(b []byte) (requestID uint64, secret string, item KeyItem, nodes []string, err error) {
	r := NewReader(b)
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		requestID = v
	}
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if sb, e2 := r.Read(ln); e2 != nil {
		err = e2
		return
	} else {
		secret = string(sb)
	}
	if it, e := decodeKeyItem(r); e != nil {
		err = e
		return
	} else {
		item = it
	}
	cnt, e := r.ReadVarint()
	if e != nil {
		err = e
		return
	}
	nodes = make([]string, 0, int(cnt))
	for i := 0; i < int(cnt); i++ {
		ln, e = r.ReadLen16()
		if e != nil {
			err = e
			return
		}
		nb, e2 := r.Read(ln)
		if e2 != nil {
			err = e2
			return
		}
		nodes = append(nodes, string(nb))
	}
	return
}

// KeyUpdateReq: {user_key:str, item:KeyItem}
func EncodeKeyUpdateReq(userKey string, item KeyItem) []byte {
	w := NewWriter(512)
	b := []byte(userKey)
	w.WriteLen16(len(b))
	w.WriteBytes(b)
	encodeKeyItem(w, item)
	return w.Bytes()
}
func DecodeKeyUpdateReq(b []byte) (userKey string, item KeyItem, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if ub, e2 := r.Read(ln); e2 != nil {
		err = e2
		return
	} else {
		userKey = string(ub)
	}
	if it, e := decodeKeyItem(r); e != nil {
		err = e
		return
	} else {
		item = it
	}
	return
}

// KeyDeleteReq: {user_key:str, id:u64}
func EncodeKeyDeleteReq(userKey string, id uint64) []byte {
	w := NewWriter(32)
	b := []byte(userKey)
	w.WriteLen16(len(b))
	w.WriteBytes(b)
	w.WriteU64(id)
	return w.Bytes()
}
func DecodeKeyDeleteReq(b []byte) (userKey string, id uint64, err error) {
	r := NewReader(b)
	ln, e := r.ReadLen16()
	if e != nil {
		err = e
		return
	}
	if ub, e2 := r.Read(ln); e2 != nil {
		err = e2
		return
	} else {
		userKey = string(ub)
	}
	if v, e := r.ReadU64(); e != nil {
		err = e
		return
	} else {
		id = v
	}
	return
}

// KeyDevicesReq: {user_key:str}
func EncodeKeyDevicesReq(userKey string) []byte    { return EncodeKeyListReq(userKey) }
func DecodeKeyDevicesReq(b []byte) (string, error) { return DecodeUserMeReq(b) }

// KeyDevicesResp: {request_id:u64, count:varint, devices:[DeviceItem]}
func EncodeKeyDevicesResp(requestID uint64, devices []DeviceItem) []byte {
	return EncodeQueryNodesResp(requestID, devices)
}
func DecodeKeyDevicesResp(b []byte) (uint64, []DeviceItem, error) { return DecodeQueryNodesResp(b) }
