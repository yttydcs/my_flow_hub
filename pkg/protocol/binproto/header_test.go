package binproto

import (
	"bytes"
	"testing"
)

func TestHeaderCodec(t *testing.T) {
	h := HeaderV1{TypeID: 1, MsgID: 123, Source: 10, Target: 20, Timestamp: 99}
	b, err := h.Encode(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != HeaderSizeV1 {
		t.Fatalf("size=%d", len(b))
	}
	var h2 HeaderV1
	if err := h2.Decode(b); err != nil {
		t.Fatal(err)
	}
	if h2 != h {
		t.Fatalf("mismatch: %+v vs %+v", h2, h)
	}
}

func TestOKErrCodec(t *testing.T) {
	ok := EncodeOKResp(7, 200, []byte("hello"))
	rid, code, msg, err := DecodeOKResp(ok)
	if err != nil {
		t.Fatal(err)
	}
	if rid != 7 || code != 200 || !bytes.Equal(msg, []byte("hello")) {
		t.Fatalf("bad decode: %d %d %q", rid, code, string(msg))
	}
	er := EncodeErrResp(9, 500, []byte("oops"))
	rid2, code2, msg2, err := DecodeErrResp(er)
	if err != nil {
		t.Fatal(err)
	}
	if rid2 != 9 || code2 != 500 || string(msg2) != "oops" {
		t.Fatalf("bad decode err: %d %d %q", rid2, code2, string(msg2))
	}
}
