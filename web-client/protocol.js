/**
 * MyFlowHub Binary Protocol v1 Implementation in JavaScript
 * 
 * This file provides helpers to encode/decode binary frames and
 * decode Protobuf payloads in the browser for MyFlowHub.
 */

// Constants for Type IDs, mirrored from Go implementation
const Type = {
    OK_RESP: 0,
    ERR_RESP: 1,
    MSG_SEND: 10,
    QUERY_NODES_REQ: 20,
    QUERY_NODES_RESP: 120,
    CREATE_DEVICE_REQ: 21,
    PARENT_AUTH_REQ: 130,
    PARENT_AUTH_RESP: 131,
    USER_LOGIN_REQ: 110,
    USER_LOGIN_RESP: 111,
    USER_ME_REQ: 112,
    USER_ME_RESP: 113,
    USER_LOGOUT_REQ: 114,
    USER_LOGOUT_RESP: 115,
    SYSTEMLOG_LIST_REQ: 150,
    SYSTEMLOG_LIST_RESP: 151,
    KEY_DEVICES_REQ: 176,
    KEY_DEVICES_RESP: 177,
    VAR_UPDATE_REQ: 162, // legacy placeholder (not used after migration)
    VAR_DELETE_REQ: 163, // legacy placeholder (not used after migration)
    // Add other TypeIDs as needed
};

/**
 * A helper class to write data into an ArrayBuffer.
 * Handles Little Endian encoding.
 */
class Writer {
    constructor(initialSize = 256) {
        this.buffer = new ArrayBuffer(initialSize);
        this.dataView = new DataView(this.buffer);
        this.offset = 0;
    }

    ensureCapacity(needed) {
        if (this.buffer.byteLength < this.offset + needed) {
            const newSize = Math.max(this.buffer.byteLength * 2, this.offset + needed);
            const newBuffer = new ArrayBuffer(newSize);
            new Uint8Array(newBuffer).set(new Uint8Array(this.buffer));
            this.buffer = newBuffer;
            this.dataView = new DataView(this.buffer);
        }
    }

    writeBytes(bytes) {
        this.ensureCapacity(bytes.length);
        new Uint8Array(this.buffer).set(bytes, this.offset);
        this.offset += bytes.length;
    }

    writeU16(value) {
        this.ensureCapacity(2);
        this.dataView.setUint16(this.offset, value, true); // true for little-endian
        this.offset += 2;
    }
    
    writeU64(value) {
        this.ensureCapacity(8);
        this.dataView.setBigUint64(this.offset, BigInt(value), true);
        this.offset += 8;
    }

    writeI64(value) {
        this.ensureCapacity(8);
        this.dataView.setBigInt64(this.offset, BigInt(value), true);
        this.offset += 8;
    }

    writeString(str) {
        const bytes = new TextEncoder().encode(str);
        this.writeU16(bytes.length);
        this.writeBytes(bytes);
    }

    getBytes() {
        return this.buffer.slice(0, this.offset);
    }
}

/**
 * A helper class to read data from an ArrayBuffer.
 * Handles Little Endian decoding.
 */
class Reader {
    constructor(buffer) {
        this.buffer = buffer;
        this.dataView = new DataView(buffer);
        this.offset = 0;
    }

    readBytes(length) {
        if (this.offset + length > this.buffer.byteLength) {
            throw new Error("Read out of bounds");
        }
        const bytes = this.buffer.slice(this.offset, this.offset + length);
        this.offset += length;
        return new Uint8Array(bytes);
    }

    readU16() {
        if (this.offset + 2 > this.buffer.byteLength) {
            throw new Error("Read out of bounds");
        }
        const value = this.dataView.getUint16(this.offset, true);
        this.offset += 2;
        return value;
    }
    
    readU64() {
        if (this.offset + 8 > this.buffer.byteLength) {
            throw new Error("Read out of bounds");
        }
        const value = this.dataView.getBigUint64(this.offset, true);
        this.offset += 8;
        return value;
    }

    readString() {
        const length = this.readU16();
        const bytes = this.readBytes(length);
        return new TextDecoder().decode(bytes);
    }
}

/**
 * Encodes a full WebSocket frame (Header + Payload).
 * @param {number} typeID - The message Type ID.
 * @param {number|BigInt} msgID - The message ID.
 * @param {number|BigInt} source - The source device UID.
 * @param {number|BigInt} target - The target device UID.
 * @param {ArrayBuffer} payload - The message payload.
 * @returns {ArrayBuffer} The encoded frame.
 */
export function encodeFrame(typeID, msgID, source, target, payload) {
    const writer = new Writer(38 + payload.byteLength);
    
    // Header (38 bytes)
    writer.writeU16(typeID);
    writer.writeBytes(new Uint8Array(4)); // Reserved
    writer.writeU64(msgID);
    writer.writeU64(source);
    writer.writeU64(target);
    writer.writeI64(Date.now());

    // Payload
    writer.writeBytes(new Uint8Array(payload));

    return writer.getBytes();
}

/**
 * Decodes a WebSocket frame into Header and Payload.
 * @param {ArrayBuffer} frame - The raw frame from WebSocket.
 * @returns {{header: object, payload: ArrayBuffer}}
 */
export function decodeFrame(frame) {
    const reader = new Reader(frame);
    const header = {
        typeID: reader.readU16(),
        reserved: reader.readBytes(4),
        msgID: reader.readU64(),
        source: reader.readU64(),
        target: reader.readU64(),
                timestamp: reader.readU64(),
    };
    const payload = frame.slice(reader.offset);
    return { header, payload };
}

// =============================================================
// Protobuf payload decoding (browser) using protobufjs
// - Requires protobufjs to be loaded on the page (window.protobuf)
// - We embed a minimal reflection JSON for常见响应，足以在调试中直观查看
// =============================================================

// Minimal descriptor covering common response messages used by the client.
// This can be extended safely; authoritative schema is in pkg/protocol/pb/myflowhub.proto
const MyFlowHubDescriptor = {
    nested: {
        myflowhub: {
            nested: {
                v1: {
                    nested: {
                        OKResp: {
                            fields: {
                                request_id: { type: "uint64", id: 1 },
                                code: { type: "int32", id: 2 },
                                message: { type: "bytes", id: 3 }
                            }
                        },
                        ErrResp: {
                            fields: {
                                request_id: { type: "uint64", id: 1 },
                                code: { type: "int32", id: 2 },
                                message: { type: "bytes", id: 3 }
                            }
                        },
                        ManagerAuthResp: {
                            fields: {
                                request_id: { type: "uint64", id: 1 },
                                device_uid: { type: "uint64", id: 2 },
                                role: { type: "string", id: 3 }
                            }
                        },
                        DeviceItem: {
                            fields: {
                                id: { type: "uint64", id: 1 },
                                device_uid: { type: "uint64", id: 2 },
                                hardware_id: { type: "string", id: 3 },
                                role: { type: "string", id: 4 },
                                name: { type: "string", id: 5 },
                                parent_id: { type: "uint64", id: 6, options: { proto3_optional: true } },
                                owner_user_id: { type: "uint64", id: 7, options: { proto3_optional: true } },
                                last_seen_sec: { type: "int64", id: 8, options: { proto3_optional: true } },
                                created_at_sec: { type: "int64", id: 9 },
                                updated_at_sec: { type: "int64", id: 10 }
                            }
                        },
                        QueryNodesResp: {
                            fields: {
                                request_id: { type: "uint64", id: 1 },
                                devices: { rule: "repeated", type: "DeviceItem", id: 2 }
                            }
                        },
                        SystemLogItem: {
                            fields: {
                                level: { type: "string", id: 1 },
                                source: { type: "string", id: 2 },
                                message: { type: "string", id: 3 },
                                details: { type: "string", id: 4 },
                                at: { type: "int64", id: 5 }
                            }
                        },
                        SystemLogListResp: {
                            fields: {
                                request_id: { type: "uint64", id: 1 },
                                total: { type: "int64", id: 2 },
                                page: { type: "int32", id: 3 },
                                page_size: { type: "int32", id: 4 },
                                logs: { rule: "repeated", type: "SystemLogItem", id: 5 }
                            }
                        },
                        UserLoginResp: {
                            fields: {
                                request_id: { type: "uint64", id: 1 },
                                key_id: { type: "uint64", id: 2 },
                                user_id: { type: "uint64", id: 3 },
                                secret: { type: "string", id: 4 },
                                username: { type: "string", id: 5 },
                                display_name: { type: "string", id: 6 },
                                permissions: { rule: "repeated", type: "string", id: 7 }
                            }
                        },
                        UserMeResp: {
                            fields: {
                                request_id: { type: "uint64", id: 1 },
                                user_id: { type: "uint64", id: 2 },
                                username: { type: "string", id: 3 },
                                display_name: { type: "string", id: 4 },
                                permissions: { rule: "repeated", type: "string", id: 5 }
                            }
                        },
                        KeyDevicesResp: {
                            fields: {
                                request_id: { type: "uint64", id: 1 },
                                devices: { rule: "repeated", type: "DeviceItem", id: 2 }
                            }
                        },
                        ParentAuthResp: {
                            fields: {
                                request_id: { type: "uint64", id: 1 },
                                device_uid: { type: "uint64", id: 2 },
                                session_id: { type: "bytes", id: 3 },
                                heartbeat_sec: { type: "uint32", id: 4 },
                                perms: { rule: "repeated", type: "string", id: 5 },
                                exp: { type: "int64", id: 6 },
                                sig: { type: "bytes", id: 7 }
                            }
                        }
                    }
                }
            }
        }
    }
};

let _pbRoot = null;
function getRoot() {
    if (_pbRoot) return _pbRoot;
    const protobuf = window.protobuf;
    if (!protobuf) {
        console.warn('protobufjs not loaded; payloads will not be decoded');
        return null;
    }
    _pbRoot = protobuf.Root.fromJSON(MyFlowHubDescriptor);
    return _pbRoot;
}

const typeIdToFQN = new Map([
    [Type.OK_RESP, 'myflowhub.v1.OKResp'],
    [Type.ERR_RESP, 'myflowhub.v1.ErrResp'],
    [Type.MANAGER_AUTH_RESP || 101, 'myflowhub.v1.ManagerAuthResp'],
    [Type.QUERY_NODES_RESP, 'myflowhub.v1.QueryNodesResp'],
    [Type.SYSTEMLOG_LIST_RESP, 'myflowhub.v1.SystemLogListResp'],
    [Type.USER_LOGIN_RESP, 'myflowhub.v1.UserLoginResp'],
    [Type.USER_ME_RESP, 'myflowhub.v1.UserMeResp'],
    [Type.KEY_DEVICES_RESP, 'myflowhub.v1.KeyDevicesResp'],
    [Type.PARENT_AUTH_RESP, 'myflowhub.v1.ParentAuthResp'],
]);

export function decodeProtobufPayload(typeID, payloadArrayBuffer) {
    const root = getRoot();
    if (!root) return null;
    const fqn = typeIdToFQN.get(typeID);
    if (!fqn) return null;
    const TypeCtor = root.lookupType(fqn);
    try {
        const msg = TypeCtor.decode(new Uint8Array(payloadArrayBuffer));
        // Convert to plain object with defaults for readability
        return TypeCtor.toObject(msg, { longs: String, bytes: String });
    } catch (e) {
        console.warn('Failed to decode payload via protobuf', e);
        return null;
    }
}

export { Type };
/**
 * Encodes a DeviceItem structure.
 * @param {object} device - The device object.
 * @returns {ArrayBuffer}
 */
function encodeDeviceItem(device) {
    const writer = new Writer();
    // bitmap: parentID(bit0), ownerUserID(bit1), lastSeen(bit2)
    let bm = 0;
    if (device.ParentID) bm |= 0x01;
    
    writer.writeBytes(new Uint8Array([bm]));
    writer.writeU64(device.ID || 0);
    writer.writeU64(device.DeviceUID || 0);
    writer.writeString(device.HardwareID || '');
    writer.writeString(device.Role || 'device');
    writer.writeString(device.Name || device.HardwareID);
    if (device.ParentID) writer.writeU64(device.ParentID);
    // Other fields omitted for client-side creation
    writer.writeI64(device.CreatedAtSec || 0);
    writer.writeI64(device.UpdatedAtSec || 0);
    return writer.getBytes();
}

/**
 * Encodes a CreateDeviceReq payload.
 * @param {string} userKey - Optional user key.
 * @param {object} device - The device item to create.
 * @returns {ArrayBuffer}
 */
function encodeCreateDeviceReq(userKey, device) {
    const writer = new Writer();
    let bm = 0;
    if (userKey) bm |= 0x01;
    writer.writeBytes(new Uint8Array([bm]));
    if (userKey) writer.writeString(userKey);
    
    const deviceBytes = encodeDeviceItem(device);
    writer.writeBytes(new Uint8Array(deviceBytes));
    return writer.getBytes();
}

/**
 * Encodes a ParentAuthReq payload.
 * NOTE: HMAC signature is simplified for client-side PoC and is NOT secure.
 * A proper implementation would use a library like crypto-js.
 * @param {string} hardwareId - The device's hardware ID.
 * @returns {ArrayBuffer}
 */
function encodeParentAuthReq(hardwareId) {
    const writer = new Writer();
    writer.writeBytes(new Uint8Array()); // version
    writer.writeI64(Date.now()); // timestamp
    writer.writeBytes(crypto.getRandomValues(new Uint8Array(16))); // nonce
    writer.writeString(hardwareId);
    writer.writeString("device"); // caps
    writer.writeBytes(new Uint8Array(32)); // dummy signature
    return writer.getBytes();
}

/**
 * Decodes an OKResp payload.
 * @param {ArrayBuffer} payload
 * @returns {{requestID: BigInt, code: number, message: string}}
 */
function decodeOKResp(payload) {
    const r = new Reader(payload);
    const requestID = r.readU64();
    const code = r.dataView.getInt32(r.offset, true);
    r.offset += 4;
    const message = r.readString();
    return { requestID, code, message };
}

/**
 * Decodes a ParentAuthResp payload.
 * @param {ArrayBuffer} payload
 * @returns {object}
 */
function decodeParentAuthResp(payload) {
    const r = new Reader(payload);
    const requestID = r.readU64();
    const deviceUID = r.readU64();
    // ... skipping other fields for now
    return { requestID, deviceUID };
}