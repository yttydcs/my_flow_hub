package binproto

import "testing"

func TestFrameRoundtrip(t *testing.T) {
	h := HeaderV1{TypeID: TypeOKResp, MsgID: 42, Source: 1, Target: 2, Timestamp: 123456}
	payload := EncodeOKResp(99, 200, []byte("hi"))
	f, err := EncodeFrame(h, payload)
	if err != nil {
		t.Fatal(err)
	}
	hh, pl, err := DecodeFrame(f)
	if err != nil {
		t.Fatal(err)
	}
	if hh != h {
		t.Fatalf("header mismatch")
	}
	rid, code, msg, err := DecodeOKResp(pl)
	if err != nil {
		t.Fatal(err)
	}
	if rid != 99 || code != 200 || string(msg) != "hi" {
		t.Fatalf("payload mismatch: %d %d %q", rid, code, string(msg))
	}
}
