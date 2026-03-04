package tasks

import (
	"testing"
)

func TestSetPayloadFormat(t *testing.T) {
	t.Parallel()

	t.Run("green accepts json and proto", func(t *testing.T) {
		t.Parallel()
		helperWithPayloadFormat(t, PayloadFormatJSON)
		helperWithPayloadFormat(t, PayloadFormatProto)
	})

	t.Run("red rejects invalid value", func(t *testing.T) {
		t.Parallel()
		helperWithPayloadFormat(t, PayloadFormatJSON)
		if err := SetPayloadFormat("xml"); err == nil {
			t.Fatal("SetPayloadFormat(xml) expected error, got nil")
		}
	})
}

func TestMarshalUnmarshalPayload_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	helperWithPayloadFormat(t, PayloadFormatJSON)

	in := EmailPayload{To: "a@b.com", Subject: "s", Body: "b"}
	data, err := marshalPayload(in)
	if err != nil {
		t.Fatalf("marshalPayload() error: %v", err)
	}

	var out EmailPayload
	if err := unmarshalPayload(data, &out); err != nil {
		t.Fatalf("unmarshalPayload() error: %v", err)
	}
	if out != in {
		t.Fatalf("roundtrip mismatch: got %+v want %+v", out, in)
	}
}

func TestMarshalPayload_ProtoUnsupportedType(t *testing.T) {
	t.Parallel()
	helperWithPayloadFormat(t, PayloadFormatProto)

	_, err := marshalPayload(struct{ X int }{X: 1})
	if err == nil {
		t.Fatal("marshalPayload() expected error for unsupported type, got nil")
	}
}
