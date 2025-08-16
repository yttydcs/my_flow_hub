package binproto

import (
	"encoding/binary"
	"errors"
)

// HeaderV1 is the fixed 38-byte header of v1.
// Layout (little endian):
//
//	TypeID[2] uint16
//	Reserved[4] uint32 = 0
//	MsgID [8] uint64
//	Source[8] uint64
//	Target[8] uint64
//	Timestamp[8] int64 (ms)
//
// Total: 38 bytes
type HeaderV1 struct {
	TypeID    uint16
	MsgID     uint64
	Source    uint64
	Target    uint64
	Timestamp int64
}

const HeaderSizeV1 = 38

func (h *HeaderV1) Encode(dst []byte) ([]byte, error) {
	if dst == nil {
		dst = make([]byte, HeaderSizeV1)
	} else if len(dst) < HeaderSizeV1 {
		return nil, errors.New("buffer too small for header")
	}
	binary.LittleEndian.PutUint16(dst[0:2], h.TypeID)
	// reserved 4 bytes set to zero
	for i := 2; i < 6; i++ {
		dst[i] = 0
	}
	binary.LittleEndian.PutUint64(dst[6:14], h.MsgID)
	binary.LittleEndian.PutUint64(dst[14:22], h.Source)
	binary.LittleEndian.PutUint64(dst[22:30], h.Target)
	binary.LittleEndian.PutUint64(dst[30:38], uint64(h.Timestamp))
	return dst[:HeaderSizeV1], nil
}

func (h *HeaderV1) Decode(src []byte) error {
	if len(src) < HeaderSizeV1 {
		return errors.New("buffer too small for header")
	}
	h.TypeID = binary.LittleEndian.Uint16(src[0:2])
	// skip reserved [2:6]
	h.MsgID = binary.LittleEndian.Uint64(src[6:14])
	h.Source = binary.LittleEndian.Uint64(src[14:22])
	h.Target = binary.LittleEndian.Uint64(src[22:30])
	h.Timestamp = int64(binary.LittleEndian.Uint64(src[30:38]))
	return nil
}
