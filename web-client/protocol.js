/**
 * MyFlowHub Binary Protocol v1 Implementation in JavaScript
 * 
 * This file provides functions to encode and decode binary messages
 * for the MyFlowHub system, matching the Go implementation.
 */

// Constants for Type IDs, mirrored from Go implementation
const Type = {
    OK_RESP: 0,
    ERR_RESP: 1,
    MSG_SEND: 10,
    CREATE_DEVICE_REQ: 21,
    PARENT_AUTH_REQ: 130,
    PARENT_AUTH_RESP: 131,
    VAR_UPDATE_REQ: 162,
    VAR_DELETE_REQ: 163,
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
function encodeFrame(typeID, msgID, source, target, payload) {
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
function decodeFrame(frame) {
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