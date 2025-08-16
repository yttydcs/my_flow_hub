package controller

import (
	"time"

	"myflowhub/pkg/database"
	binproto "myflowhub/pkg/protocol/binproto"
	"myflowhub/server/internal/hub"
)

// ========== Auth ==========
type AuthBin struct{ C *AuthController }

func (a *AuthBin) ManagerAuth(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	token, err := binproto.DecodeManagerAuthReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	deviceUID, role, err := a.C.AuthenticateManagerToken(token)
	if err != nil {
		sendErr(s, c, h, 401, "unauthorized")
		return
	}
	c.DeviceID = deviceUID
	s.Clients[c.DeviceID] = c
	pl := binproto.EncodeManagerAuthResp(h.MsgID, deviceUID, role)
	sendFrame(s, c, h, binproto.TypeManagerAuthResp, pl)
}

func (a *AuthBin) UserLogin(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	username, password, err := binproto.DecodeUserLoginReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	keyID, userID, secret, uname, displayName, perms, err := a.C.Login(username, password)
	if err != nil {
		sendErr(s, c, h, 401, "invalid credentials")
		return
	}
	pl := binproto.EncodeUserLoginResp(h.MsgID, keyID, userID, secret, uname, displayName, perms)
	sendFrame(s, c, h, binproto.TypeUserLoginResp, pl)
}

func (a *AuthBin) UserMe(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, err := binproto.DecodeUserMeReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	uid, uname, displayName, perms, err := a.C.Me(userKey)
	if err != nil {
		sendErr(s, c, h, 401, "invalid key")
		return
	}
	pl := binproto.EncodeUserMeResp(h.MsgID, uid, uname, displayName, perms)
	sendFrame(s, c, h, binproto.TypeUserMeResp, pl)
}

func (a *AuthBin) UserLogout(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, err := binproto.DecodeUserLogoutReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	if e := a.C.Logout(userKey); e != nil {
		sendErr(s, c, h, 400, "logout failed")
		return
	}
	sendOK(s, c, h, 0, "ok")
}

// ========== Devices ==========
type DeviceBin struct{ C *DeviceController }

func (d *DeviceBin) QueryNodes(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, err := binproto.DecodeQueryNodesReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	list, err := d.C.QueryVisibleDevices(userKey, c.DeviceID)
	if err != nil {
		sendErr(s, c, h, 401, "unauthorized")
		return
	}
	items := make([]binproto.DeviceItem, 0, len(list))
	for _, dv := range list {
		var parentID, ownerID *uint64
		if dv.ParentID != nil {
			parentID = dv.ParentID
		}
		if dv.OwnerUserID != nil {
			ownerID = dv.OwnerUserID
		}
		var last *int64
		if dv.LastSeen != nil {
			v := dv.LastSeen.Unix()
			last = &v
		}
		items = append(items, binproto.DeviceItem{ID: dv.ID, DeviceUID: dv.DeviceUID, HardwareID: dv.HardwareID, Role: string(dv.Role), Name: dv.Name, ParentID: parentID, OwnerUserID: ownerID, LastSeenSec: last, CreatedAtSec: dv.CreatedAt.Unix(), UpdatedAtSec: dv.UpdatedAt.Unix()})
	}
	pl := binproto.EncodeQueryNodesResp(h.MsgID, items)
	sendFrame(s, c, h, binproto.TypeQueryNodesResp, pl)
}

func (d *DeviceBin) Create(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	uk, item, err := binproto.DecodeCreateDeviceReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	dev := &database.Device{HardwareID: item.HardwareID, Role: database.DeviceRole(item.Role), Name: item.Name}
	if item.ParentID != nil {
		dev.ParentID = item.ParentID
	}
	if item.OwnerUserID != nil {
		dev.OwnerUserID = item.OwnerUserID
	}
	if e := d.C.CreateDevice(uk, *dev, c.DeviceID); e != nil {
		sendErr(s, c, h, 403, "permission denied")
		return
	}
	sendOK(s, c, h, 0, "ok")
}

func (d *DeviceBin) Update(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	uk, item, err := binproto.DecodeUpdateDeviceReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	dev := database.Device{ID: item.ID}
	if item.HardwareID != "" {
		dev.HardwareID = item.HardwareID
	}
	if item.Role != "" {
		dev.Role = database.DeviceRole(item.Role)
	}
	if item.Name != "" {
		dev.Name = item.Name
	}
	if item.ParentID != nil {
		dev.ParentID = item.ParentID
	}
	if item.OwnerUserID != nil {
		dev.OwnerUserID = item.OwnerUserID
	}
	if e := d.C.UpdateDevice(uk, dev, c.DeviceID); e != nil {
		sendErr(s, c, h, 403, "permission denied")
		return
	}
	sendOK(s, c, h, 0, "ok")
}

func (d *DeviceBin) Delete(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	id, uk, err := binproto.DecodeDeleteDeviceReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	if e := d.C.DeleteDevice(uk, id, c.DeviceID); e != nil {
		sendErr(s, c, h, 403, "permission denied")
		return
	}
	sendOK(s, c, h, 0, "ok")
}

// ========== Variables ==========
type VariableBin struct{ C *VariableController }

func (v *VariableBin) Update(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, items, err := binproto.DecodeVarUpdateReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	conv := make([]VarKV, 0, len(items))
	for _, it := range items {
		conv = append(conv, VarKV{DeviceUID: it.DeviceUID, Name: it.Name, Value: it.Value})
	}
	_, _ = v.C.Update(userKey, conv, c.DeviceID)
	sendOK(s, c, h, 0, "ok")
}

func (v *VariableBin) Delete(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, items, err := binproto.DecodeVarDeleteReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	conv := make([]VarKey, 0, len(items))
	for _, it := range items {
		conv = append(conv, VarKey{DeviceUID: it.DeviceUID, Name: it.Name})
	}
	_, _ = v.C.Delete(userKey, conv, c.DeviceID)
	sendOK(s, c, h, 0, "ok")
}

func (v *VariableBin) List(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, deviceUIDPtr, err := binproto.DecodeVarListReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	list, e := v.C.List(userKey, deviceUIDPtr, c.DeviceID)
	if e != nil {
		sendErr(s, c, h, 403, "permission denied")
		return
	}
	items := make([]binproto.VarListItem, 0, len(list))
	for _, it := range list {
		// enrich OwnerDeviceUID by querying device service
		ownerUID := uint64(0)
		if owner, derr := v.C.deviceService.GetDeviceByID(it.OwnerDeviceID); derr == nil && owner != nil {
			ownerUID = owner.DeviceUID
		}
		items = append(items, binproto.VarListItem{ID: it.ID, OwnerDeviceID: it.OwnerDeviceID, OwnerDeviceUID: ownerUID, Name: it.VariableName, Value: []byte(it.Value), CreatedAtSec: it.CreatedAt.Unix(), UpdatedAtSec: it.UpdatedAt.Unix()})
	}
	pl := binproto.EncodeVarListResp(h.MsgID, items)
	sendFrame(s, c, h, binproto.TypeVarListResp, pl)
}

// ========== Keys ==========
type KeyBin struct{ C *KeyController }

func (k *KeyBin) List(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, err := binproto.DecodeKeyListReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	list, err := k.C.List(userKey)
	if err != nil {
		sendErr(s, c, h, 401, "unauthorized")
		return
	}
	items := make([]binproto.KeyItem, 0, len(list))
	for _, kk := range list {
		var owner, bid *uint64
		var bt *string
		if kk.OwnerUserID != nil {
			owner = kk.OwnerUserID
		}
		if kk.BindSubjectType != nil {
			bt = kk.BindSubjectType
		}
		if kk.BindSubjectID != nil {
			bid = kk.BindSubjectID
		}
		var exp *int64
		if kk.ExpiresAt != nil {
			v := kk.ExpiresAt.Unix()
			exp = &v
		}
		var max, rem *int32
		if kk.MaxUses != nil {
			v := int32(*kk.MaxUses)
			max = &v
		}
		if kk.RemainingUses != nil {
			v := int32(*kk.RemainingUses)
			rem = &v
		}
		var issuedBy *uint64
		if kk.IssuedBy != nil {
			issuedBy = kk.IssuedBy
		}
		items = append(items, binproto.KeyItem{ID: kk.ID, OwnerUserID: owner, BindSubjectType: bt, BindSubjectID: bid, SecretHash: kk.SecretHash, ExpiresAtSec: exp, MaxUses: max, RemainingUses: rem, Revoked: kk.Revoked, IssuedBy: issuedBy, IssuedAtSec: kk.IssuedAt.Unix(), Meta: []byte(kk.Meta)})
	}
	pl := binproto.EncodeKeyListResp(h.MsgID, items)
	sendFrame(s, c, h, binproto.TypeKeyListResp, pl)
}

func (k *KeyBin) Create(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, bindType, bindID, expiresAtSec, maxUses, meta, nodes, err := binproto.DecodeKeyCreateReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	var expPtr *time.Time
	if expiresAtSec != nil {
		v := time.Unix(*expiresAtSec, 0)
		expPtr = &v
	}
	var maxPtr *int32
	if maxUses != nil {
		v := int32(*maxUses)
		maxPtr = &v
	}
	secret, key, nodesOut, e := k.C.Create(userKey, bindType, bindID, expPtr, maxPtr, meta, nodes)
	if e != nil {
		sendErr(s, c, h, 401, "unauthorized")
		return
	}
	var owner, bid *uint64
	var bt *string
	if key.OwnerUserID != nil {
		owner = key.OwnerUserID
	}
	if key.BindSubjectType != nil {
		bt = key.BindSubjectType
	}
	if key.BindSubjectID != nil {
		bid = key.BindSubjectID
	}
	var exp *int64
	if key.ExpiresAt != nil {
		v := key.ExpiresAt.Unix()
		exp = &v
	}
	var max, rem *int32
	if key.MaxUses != nil {
		v := int32(*key.MaxUses)
		max = &v
	}
	if key.RemainingUses != nil {
		v := int32(*key.RemainingUses)
		rem = &v
	}
	var issuedBy *uint64
	if key.IssuedBy != nil {
		issuedBy = key.IssuedBy
	}
	item := binproto.KeyItem{ID: key.ID, OwnerUserID: owner, BindSubjectType: bt, BindSubjectID: bid, SecretHash: key.SecretHash, ExpiresAtSec: exp, MaxUses: max, RemainingUses: rem, Revoked: key.Revoked, IssuedBy: issuedBy, IssuedAtSec: key.IssuedAt.Unix(), Meta: []byte(key.Meta)}
	pl := binproto.EncodeKeyCreateResp(h.MsgID, secret, item, nodesOut)
	sendFrame(s, c, h, binproto.TypeKeyCreateResp, pl)
}

func (k *KeyBin) Update(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, item, err := binproto.DecodeKeyUpdateReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	kdb := &database.Key{ID: item.ID, SecretHash: item.SecretHash, Revoked: item.Revoked}
	if item.OwnerUserID != nil {
		kdb.OwnerUserID = item.OwnerUserID
	}
	if item.BindSubjectType != nil {
		kdb.BindSubjectType = item.BindSubjectType
	}
	if item.BindSubjectID != nil {
		kdb.BindSubjectID = item.BindSubjectID
	}
	if item.ExpiresAtSec != nil {
		v := time.Unix(*item.ExpiresAtSec, 0)
		kdb.ExpiresAt = &v
	}
	if item.MaxUses != nil {
		v := int(*item.MaxUses)
		kdb.MaxUses = &v
	}
	if item.RemainingUses != nil {
		v := int(*item.RemainingUses)
		kdb.RemainingUses = &v
	}
	if item.IssuedBy != nil {
		kdb.IssuedBy = item.IssuedBy
	}
	if len(item.Meta) > 0 {
		kdb.Meta = item.Meta
	}
	if err := k.C.Update(userKey, kdb); err != nil {
		sendErr(s, c, h, 401, "unauthorized")
		return
	}
	sendOK(s, c, h, 0, "ok")
}

func (k *KeyBin) Delete(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, id, err := binproto.DecodeKeyDeleteReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	if err := k.C.Delete(userKey, id); err != nil {
		sendErr(s, c, h, 401, "unauthorized")
		return
	}
	sendOK(s, c, h, 0, "ok")
}

func (k *KeyBin) Devices(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, err := binproto.DecodeKeyDevicesReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	list, e := k.C.VisibleDevices(userKey)
	if e != nil {
		sendErr(s, c, h, 401, "unauthorized")
		return
	}
	items := make([]binproto.DeviceItem, 0, len(list))
	for _, d := range list {
		var parentID, ownerID *uint64
		if d.ParentID != nil {
			parentID = d.ParentID
		}
		if d.OwnerUserID != nil {
			ownerID = d.OwnerUserID
		}
		var last *int64
		if d.LastSeen != nil {
			v := d.LastSeen.Unix()
			last = &v
		}
		items = append(items, binproto.DeviceItem{ID: d.ID, DeviceUID: d.DeviceUID, HardwareID: d.HardwareID, Role: string(d.Role), Name: d.Name, ParentID: parentID, OwnerUserID: ownerID, LastSeenSec: last, CreatedAtSec: d.CreatedAt.Unix(), UpdatedAtSec: d.UpdatedAt.Unix()})
	}
	pl := binproto.EncodeKeyDevicesResp(h.MsgID, items)
	sendFrame(s, c, h, binproto.TypeKeyDevicesResp, pl)
}

// ========== SystemLog ==========
type SystemLogBin struct{ C *SystemLogController }

func (slog *SystemLogBin) List(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, level, source, keyword, startAt, endAt, page, pageSize, err := binproto.DecodeSystemLogListReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	var startPtr, endPtr *int64
	if startAt > 0 {
		startPtr = &startAt
	}
	if endAt > 0 {
		endPtr = &endAt
	}
	out, e := slog.C.List(userKey, SystemLogListRequest{Level: level, Source: source, Keyword: keyword, StartAt: startPtr, EndAt: endPtr, Page: int(page), PageSize: int(pageSize)})
	if e != nil {
		sendErr(s, c, h, 500, "query failed")
		return
	}
	items := make([]binproto.SystemLogItem, 0, len(out.Items))
	for _, it := range out.Items {
		items = append(items, binproto.SystemLogItem{Level: it.Level, Source: it.Source, Message: it.Message, Details: string(it.Details), At: it.At.Unix()})
	}
	pl := binproto.EncodeSystemLogListResp(h.MsgID, out.Total, int32(out.Page), int32(out.Size), items)
	sendFrame(s, c, h, binproto.TypeSystemLogListResp, pl)
}

// ========== Users Management ==========
type UserBin struct{ Users *UserController }

func (ub *UserBin) List(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	// 要求管理员权限：通过用户密钥判断
	userKey, _ := binproto.DecodeUserMeReq(payload) // 复用 user_key 解码
	uid, ok := ub.Users.authz.ResolveUserIDFromKey(userKey)
	if !ok || !ub.Users.authz.HasUserPermission(uid, "admin.manage") {
		sendErr(s, c, h, 403, "permission denied")
		return
	}
	list, err := ub.Users.users.List()
	if err != nil {
		sendErr(s, c, h, 500, "list failed")
		return
	}
	items := make([]binproto.UserItem, 0, len(list))
	for _, u := range list {
		items = append(items, binproto.UserItem{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, Disabled: u.Disabled, CreatedAtSec: u.CreatedAt.Unix(), UpdatedAtSec: u.UpdatedAt.Unix()})
	}
	pl := binproto.EncodeUserListResp(h.MsgID, items)
	sendFrame(s, c, h, binproto.TypeUserListResp, pl)
}

func (ub *UserBin) Create(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, username, displayName, password, err := binproto.DecodeUserCreateReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	uid, ok := ub.Users.authz.ResolveUserIDFromKey(userKey)
	if !ok || !ub.Users.authz.HasUserPermission(uid, "admin.manage") {
		sendErr(s, c, h, 403, "permission denied")
		return
	}
	u, e := ub.Users.users.Create(username, displayName, password)
	if e != nil {
		sendErr(s, c, h, 500, "create failed")
		return
	}
	pl := binproto.EncodeUserCreateResp(h.MsgID, u.ID)
	sendFrame(s, c, h, binproto.TypeUserCreateResp, pl)
}

func (ub *UserBin) Update(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, id, displayName, password, disabled, err := binproto.DecodeUserUpdateReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	uid, ok := ub.Users.authz.ResolveUserIDFromKey(userKey)
	if !ok || !ub.Users.authz.HasUserPermission(uid, "admin.manage") {
		sendErr(s, c, h, 403, "permission denied")
		return
	}
	if e := ub.Users.users.Update(id, displayName, password, disabled); e != nil {
		sendErr(s, c, h, 500, "update failed")
		return
	}
	sendOK(s, c, h, 0, "ok")
}

func (ub *UserBin) Delete(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, id, err := binproto.DecodeUserDeleteReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	uid, ok := ub.Users.authz.ResolveUserIDFromKey(userKey)
	if !ok || !ub.Users.authz.HasUserPermission(uid, "admin.manage") {
		sendErr(s, c, h, 403, "permission denied")
		return
	}
	if e := ub.Users.users.Delete(id); e != nil {
		sendErr(s, c, h, 500, "delete failed")
		return
	}
	sendOK(s, c, h, 0, "ok")
}

func (ub *UserBin) PermList(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, targetID, err := binproto.DecodeUserPermListReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	uid, ok := ub.Users.authz.ResolveUserIDFromKey(userKey)
	if !ok || !ub.Users.authz.HasUserPermission(uid, "admin.manage") {
		sendErr(s, c, h, 403, "permission denied")
		return
	}
	ps, e := ub.Users.permsRepo.ListByUserID(targetID)
	if e != nil {
		sendErr(s, c, h, 500, "query failed")
		return
	}
	items := make([]binproto.PermissionItem, 0, len(ps))
	for _, p := range ps {
		items = append(items, binproto.PermissionItem{Node: p.Node})
	}
	pl := binproto.EncodeUserPermListResp(h.MsgID, items)
	sendFrame(s, c, h, binproto.TypeUserPermListResp, pl)
}

func (ub *UserBin) PermAdd(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, targetID, node, err := binproto.DecodeUserPermAddReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	uid, ok := ub.Users.authz.ResolveUserIDFromKey(userKey)
	if !ok || !ub.Users.authz.HasUserPermission(uid, "admin.manage") {
		sendErr(s, c, h, 403, "permission denied")
		return
	}
	if e := ub.Users.permsRepo.AddUserNode(targetID, node, &uid); e != nil {
		sendErr(s, c, h, 500, "add failed")
		return
	}
	sendOK(s, c, h, 0, "ok")
}

func (ub *UserBin) PermRemove(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, targetID, node, err := binproto.DecodeUserPermRemoveReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	uid, ok := ub.Users.authz.ResolveUserIDFromKey(userKey)
	if !ok || !ub.Users.authz.HasUserPermission(uid, "admin.manage") {
		sendErr(s, c, h, 403, "permission denied")
		return
	}
	if e := ub.Users.permsRepo.RemoveUserNode(targetID, node); e != nil {
		sendErr(s, c, h, 500, "remove failed")
		return
	}
	sendOK(s, c, h, 0, "ok")
}

func (ub *UserBin) SelfUpdate(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, displayName, err := binproto.DecodeUserSelfUpdateReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	uid, ok := ub.Users.authz.ResolveUserIDFromKey(userKey)
	if !ok {
		sendErr(s, c, h, 401, "unauthorized")
		return
	}
	if e := ub.Users.users.UpdateDisplayName(uid, displayName); e != nil {
		sendErr(s, c, h, 500, "update failed")
		return
	}
	sendOK(s, c, h, 0, "ok")
}

func (ub *UserBin) SelfPassword(s *hub.Server, c *hub.Client, h binproto.HeaderV1, payload []byte) {
	userKey, oldPassword, newPassword, err := binproto.DecodeUserSelfPasswordReq(payload)
	if err != nil {
		sendErr(s, c, h, 400, "bad request")
		return
	}
	uid, ok := ub.Users.authz.ResolveUserIDFromKey(userKey)
	if !ok {
		sendErr(s, c, h, 401, "unauthorized")
		return
	}
	if e := ub.Users.users.ChangePasswordWithVerify(uid, oldPassword, newPassword); e != nil {
		sendErr(s, c, h, 400, "change failed")
		return
	}
	sendOK(s, c, h, 0, "ok")
}
