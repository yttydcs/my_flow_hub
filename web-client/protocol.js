/**
 * MyFlowHub Binary Protocol v1 - Browser Helpers (ESM)
 * - Frame header (38B LE) encode/decode
 * - TypeID constants
 * - Protobuf encode/decode via protobufjs (bundled reflection subset)
 */

// --- Type IDs (subset, aligned with Go binproto) ---
export const Type = {
    OK_RESP: 0,
    ERR_RESP: 1,
    MSG_SEND: 10,
    // Devices
    QUERY_NODES_REQ: 20,
    QUERY_NODES_RESP: 120,
    CREATE_DEVICE_REQ: 21,
    // Auth (manager/user)
    MANAGER_AUTH_REQ: 100,
    MANAGER_AUTH_RESP: 101,
    USER_LOGIN_REQ: 110,
    USER_LOGIN_RESP: 111,
    USER_ME_REQ: 112,
    USER_ME_RESP: 113,
    USER_LOGOUT_REQ: 114,
    USER_LOGOUT_RESP: 115,
    // System logs
    SYSTEMLOG_LIST_REQ: 150,
    SYSTEMLOG_LIST_RESP: 151,
    // Parent link auth
    PARENT_AUTH_REQ: 130,
    PARENT_AUTH_RESP: 131,
    // Keys (subset)
    KEY_DEVICES_REQ: 176,
    KEY_DEVICES_RESP: 177,
}

// --- Binary Writer/Reader ---
class Writer {
    constructor(initialSize = 256) {
        this.buffer = new ArrayBuffer(initialSize)
        this.dv = new DataView(this.buffer)
        this.offset = 0
    }
    ensure(n) {
        if (this.buffer.byteLength < this.offset + n) {
            const newSize = Math.max(this.buffer.byteLength * 2, this.offset + n)
            const buf = new ArrayBuffer(newSize)
            new Uint8Array(buf).set(new Uint8Array(this.buffer))
            this.buffer = buf
            this.dv = new DataView(this.buffer)
        }
    }
    writeBytes(arr) {
        this.ensure(arr.length)
        new Uint8Array(this.buffer).set(arr, this.offset)
        this.offset += arr.length
    }
    writeU16(v) {
        this.ensure(2)
        this.dv.setUint16(this.offset, v, true)
        this.offset += 2
    }
    writeU64(v) {
        // Robust LE write compatible with browsers: split into two 32-bit words
        // Accept number|string|bigint
        let big = typeof v === 'bigint' ? v : BigInt(v)
        // Normalize to unsigned 64-bit space
        if (big < 0n) big = (1n << 64n) + big
        const lo = Number(big & 0xffffffffn)
        const hi = Number((big >> 32n) & 0xffffffffn)
        this.ensure(8)
        this.dv.setUint32(this.offset + 0, lo, true)
        this.dv.setUint32(this.offset + 4, hi, true)
        this.offset += 8
    }
    writeI64(v) {
        // Two's complement encoding for signed 64-bit
        let big = typeof v === 'bigint' ? v : BigInt(v)
        if (big < 0n) big = (1n << 64n) + big
        const lo = Number(big & 0xffffffffn)
        const hi = Number((big >> 32n) & 0xffffffffn)
        this.ensure(8)
        this.dv.setUint32(this.offset + 0, lo, true)
        this.dv.setUint32(this.offset + 4, hi, true)
        this.offset += 8
    }
    getBytes() {
        return this.buffer.slice(0, this.offset)
    }
}

class Reader {
    constructor(buffer) {
        this.buffer = buffer
        this.dv = new DataView(buffer)
        this.offset = 0
    }
    readBytes(n) {
        if (this.offset + n > this.buffer.byteLength) throw new Error('oob')
        const out = this.buffer.slice(this.offset, this.offset + n)
        this.offset += n
        return new Uint8Array(out)
    }
    readU16() {
        if (this.offset + 2 > this.buffer.byteLength) throw new Error('oob')
        const v = this.dv.getUint16(this.offset, true)
        this.offset += 2
        return v
    }
    readU64() {
        if (this.offset + 8 > this.buffer.byteLength) throw new Error('oob')
        const v = this.dv.getBigUint64(this.offset, true)
        this.offset += 8
        return v
    }
    readI64() {
        if (this.offset + 8 > this.buffer.byteLength) throw new Error('oob')
        const v = this.dv.getBigInt64(this.offset, true)
        this.offset += 8
        return v
    }
}

// --- Frame helpers ---
function writeU16LE(buf, off, v) {
    buf[off + 0] = v & 0xff
    buf[off + 1] = (v >>> 8) & 0xff
}
function writeU64LE(buf, off, v) {
    let x = typeof v === 'bigint' ? v : BigInt(v)
    if (x < 0n) x = (1n << 64n) + x
    for (let i = 0; i < 8; i++) {
        buf[off + i] = Number((x >> BigInt(8 * i)) & 0xffn)
    }
}
function writeI64LE(buf, off, v) {
    let x = typeof v === 'bigint' ? v : BigInt(v)
    if (x < 0n) x = (1n << 64n) + x
    for (let i = 0; i < 8; i++) {
        buf[off + i] = Number((x >> BigInt(8 * i)) & 0xffn)
    }
}

export function encodeFrame(typeID, msgID, source, target, payload) {
    const payloadLen = payload ? (payload.byteLength ?? payload.length ?? 0) : 0
    const out = new Uint8Array(38 + payloadLen)
    // typeID
    writeU16LE(out, 0, Number(typeID & 0xffff))
    // reserved 4 bytes already zero by default
    // msgID/source/target/timestamp
    writeU64LE(out, 6, msgID)
    writeU64LE(out, 14, source)
    writeU64LE(out, 22, target)
    writeI64LE(out, 30, BigInt(Date.now()))
    if (payloadLen) {
        const u8 = payload instanceof Uint8Array ? payload : new Uint8Array(payload)
        out.set(u8, 38)
    }
    return out.buffer
}

export function decodeFrame(frame) {
    const r = new Reader(frame)
    const header = {
        typeID: r.readU16(),
        reserved: r.readBytes(4),
        msgID: r.readU64(),
        source: r.readU64(),
        target: r.readU64(),
        timestamp: r.readI64(),
    }
    const payload = frame.slice(r.offset)
    return { header, payload }
}

export function nextMsgID() {
    const n = BigInt(Date.now())
    const rnd = BigInt(Math.floor(Math.random() * 0xff))
    return (n << 8n) ^ rnd
}

// --- Protobuf (protobufjs) reflection subset ---
const MyFlowHubDescriptor = {
    nested: {
        myflowhub: {
            nested: {
                v1: {
                    nested: {
                        ManagerAuthReq: {
                            fields: { token: { type: 'string', id: 1 } },
                        },
                        ManagerAuthResp: {
                            fields: {
                                request_id: { type: 'uint64', id: 1 },
                                device_uid: { type: 'uint64', id: 2 },
                                role: { type: 'string', id: 3 },
                            },
                        },
                        OKResp: {
                            fields: {
                                request_id: { type: 'uint64', id: 1 },
                                code: { type: 'int32', id: 2 },
                                message: { type: 'bytes', id: 3 },
                            },
                        },
                        ErrResp: {
                            fields: {
                                request_id: { type: 'uint64', id: 1 },
                                code: { type: 'int32', id: 2 },
                                message: { type: 'bytes', id: 3 },
                            },
                        },
                        ParentAuthReq: {
                            fields: {
                                version: { type: 'uint32', id: 1 },
                                ts_ms: { type: 'int64', id: 2 },
                                nonce: { type: 'bytes', id: 3 },
                                hardware_id: { type: 'string', id: 4 },
                                caps: { type: 'string', id: 5 },
                                sig: { type: 'bytes', id: 6 },
                            },
                        },
                        ParentAuthResp: {
                            fields: {
                                request_id: { type: 'uint64', id: 1 },
                                device_uid: { type: 'uint64', id: 2 },
                                session_id: { type: 'bytes', id: 3 },
                                heartbeat_sec: { type: 'uint32', id: 4 },
                                perms: { rule: 'repeated', type: 'string', id: 5 },
                                exp: { type: 'int64', id: 6 },
                                sig: { type: 'bytes', id: 7 },
                            },
                        },
                    },
                },
            },
        },
    },
}

let _root = null
function getRoot() {
    if (_root) return _root
    const protobuf = window.protobuf
    if (!protobuf) return null
    _root = protobuf.Root.fromJSON(MyFlowHubDescriptor)
    return _root
}

const typeIdToFQN = new Map([
    [Type.OK_RESP, 'myflowhub.v1.OKResp'],
    [Type.ERR_RESP, 'myflowhub.v1.ErrResp'],
    [Type.MANAGER_AUTH_RESP, 'myflowhub.v1.ManagerAuthResp'],
    [Type.PARENT_AUTH_RESP, 'myflowhub.v1.ParentAuthResp'],
])

export function decodeProtobufPayload(typeID, payloadArrayBuffer) {
    const root = getRoot()
    if (!root) return null
    const fqn = typeIdToFQN.get(typeID)
    if (!fqn) return null
    try {
        const TypeCtor = root.lookupType(fqn)
        const msg = TypeCtor.decode(new Uint8Array(payloadArrayBuffer))
        return TypeCtor.toObject(msg, { longs: String, bytes: String })
    } catch (e) {
        console.warn('protobuf decode failed', e)
        return null
    }
}

function encodeProtobuf(fqn, obj) {
    const root = getRoot()
    if (!root) throw new Error('protobufjs not loaded')
    const TypeCtor = root.lookupType(fqn)
    const err = TypeCtor.verify(obj)
    if (err) throw new Error('protobuf verify failed: ' + err)
    const msg = TypeCtor.create(obj)
    const buf = TypeCtor.encode(msg).finish()
    // Ensure we return an ArrayBuffer containing exactly the encoded bytes
    return buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength)
}

// Ensure a value is Uint8Array for protobuf `bytes` fields
function ensureBytes(v) {
    if (v == null) return v
    if (v instanceof Uint8Array) return v
    if (ArrayBuffer.isView(v)) return new Uint8Array(v.buffer, v.byteOffset, v.byteLength)
    if (v instanceof ArrayBuffer) return new Uint8Array(v)
    if (typeof v === 'string') return new TextEncoder().encode(v)
    throw new Error('invalid bytes value')
}

// --- High-level encoders ---
export function encodeManagerAuthReq(token) {
    return encodeProtobuf('myflowhub.v1.ManagerAuthReq', { token })
}

// ParentAuth helpers
export function encodeParentAuthReq({ version = 1, tsMs, nonce, hardwareId, caps, sig }) {
    return encodeProtobuf('myflowhub.v1.ParentAuthReq', {
        version,
        ts_ms: tsMs,
        nonce: ensureBytes(nonce),
        hardware_id: hardwareId,
        caps,
        sig: ensureBytes(sig),
    })
}