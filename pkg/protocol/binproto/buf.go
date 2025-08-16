package binproto

import (
	"encoding/binary"
	"errors"
)

type Writer struct{ b []byte }

type Reader struct {
	b   []byte
	off int
}

func NewWriter(capacity int) *Writer { return &Writer{b: make([]byte, 0, capacity)} }
func (w *Writer) Bytes() []byte      { return w.b }

func (w *Writer) WriteU16(v uint16) {
	tmp := make([]byte, 2)
	binary.LittleEndian.PutUint16(tmp, v)
	w.b = append(w.b, tmp...)
}
func (w *Writer) WriteU32(v uint32) {
	tmp := make([]byte, 4)
	binary.LittleEndian.PutUint32(tmp, v)
	w.b = append(w.b, tmp...)
}
func (w *Writer) WriteU64(v uint64) {
	tmp := make([]byte, 8)
	binary.LittleEndian.PutUint64(tmp, v)
	w.b = append(w.b, tmp...)
}
func (w *Writer) WriteI32(v int32)    { w.WriteU32(uint32(v)) }
func (w *Writer) WriteI64(v int64)    { w.WriteU64(uint64(v)) }
func (w *Writer) WriteBytes(p []byte) { w.b = append(w.b, p...) }

func (w *Writer) WriteLen16(n int) {
	if n < 0xFFFF {
		w.WriteU16(uint16(n))
	} else {
		w.WriteU16(0xFFFF)
		w.WriteU32(uint32(n))
	}
}

func (w *Writer) WriteVarint(u uint64) {
	for u >= 0x80 {
		w.b = append(w.b, byte(u)|0x80)
		u >>= 7
	}
	w.b = append(w.b, byte(u))
}

func NewReader(b []byte) *Reader { return &Reader{b: b} }

func (r *Reader) Read(n int) ([]byte, error) {
	if r.off+n > len(r.b) {
		return nil, errors.New("short buffer")
	}
	p := r.b[r.off : r.off+n]
	r.off += n
	return p, nil
}

func (r *Reader) ReadU16() (uint16, error) {
	p, e := r.Read(2)
	if e != nil {
		return 0, e
	}
	return binary.LittleEndian.Uint16(p), nil
}
func (r *Reader) ReadU32() (uint32, error) {
	p, e := r.Read(4)
	if e != nil {
		return 0, e
	}
	return binary.LittleEndian.Uint32(p), nil
}
func (r *Reader) ReadU64() (uint64, error) {
	p, e := r.Read(8)
	if e != nil {
		return 0, e
	}
	return binary.LittleEndian.Uint64(p), nil
}
func (r *Reader) ReadI32() (int32, error) { u, e := r.ReadU32(); return int32(u), e }
func (r *Reader) ReadI64() (int64, error) { u, e := r.ReadU64(); return int64(u), e }

func (r *Reader) ReadLen16() (int, error) {
	u16, e := r.ReadU16()
	if e != nil {
		return 0, e
	}
	if u16 < 0xFFFF {
		return int(u16), nil
	}
	u32, e2 := r.ReadU32()
	if e2 != nil {
		return 0, e2
	}
	return int(u32), nil
}

func (r *Reader) ReadVarint() (uint64, error) {
	var v uint64
	var shift uint
	for {
		if r.off >= len(r.b) {
			return 0, errors.New("short buffer")
		}
		b := r.b[r.off]
		r.off++
		v |= uint64(b&0x7F) << shift
		if b < 0x80 {
			break
		}
		shift += 7
		if shift > 63 {
			return 0, errors.New("varint overflow")
		}
	}
	return v, nil
}
