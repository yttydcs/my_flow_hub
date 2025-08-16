package binproto

import "errors"

// EncodeFrame builds header(38B)+payload.
func EncodeFrame(h HeaderV1, payload []byte) ([]byte, error) {
	hb, err := h.Encode(nil)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(hb)+len(payload))
	out = append(out, hb...)
	out = append(out, payload...)
	return out, nil
}

// DecodeFrame splits header and payload view without copy.
func DecodeFrame(b []byte) (HeaderV1, []byte, error) {
	if len(b) < HeaderSizeV1 {
		return HeaderV1{}, nil, errors.New("short frame")
	}
	var h HeaderV1
	if err := h.Decode(b[:HeaderSizeV1]); err != nil {
		return HeaderV1{}, nil, err
	}
	return h, b[HeaderSizeV1:], nil
}
