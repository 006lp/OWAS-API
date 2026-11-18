package protostuff

import (
	"encoding/binary"
	"encoding/hex"
	"math"
	"testing"
)

func TestVarintRoundTrip(t *testing.T) {
	values := []int64{0, 1, 127, 128, 300, 31617863, -1, -42, math.MaxInt64, math.MinInt64}
	for _, v := range values {
		enc := AppendVarint(nil, uint64(v))
		got, pos, err := DecodeVarint(enc, 0)
		if err != nil {
			t.Fatalf("DecodeVarint(%d) failed: %v", v, err)
		}
		if pos != len(enc) {
			t.Fatalf("expected pos=%d, got %d", len(enc), pos)
		}
		if int64(got) != v {
			t.Fatalf("expected %d, got %d", v, int64(got))
		}
	}
}

func TestVarintErrors(t *testing.T) {
	if _, _, err := DecodeVarint([]byte{0x80}, 0); err == nil {
		t.Fatal("expected error for truncated varint")
	}
	overlong := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}
	if _, _, err := DecodeVarint(overlong, 0); err == nil {
		t.Fatal("expected error for overlong varint")
	}
}

func TestEncoderGoldenVectors(t *testing.T) {
	// Golden vectors verified against Python oppo_watch_store/wire.py.
	t.Run("UpgradeWrapReq", func(t *testing.T) {
		item := NewEncoder().
			AppendString(1, "com.tencent.watchwechat").
			AppendString(2, "1").
			Bytes()
		got := hex.EncodeToString(NewEncoder().AppendMessage(1, item).Bytes())
		want := "0a1c0a17636f6d2e74656e63656e742e7761746368776563686174120131"
		if got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	})

	t.Run("OrderPayParam", func(t *testing.T) {
		got := hex.EncodeToString(NewEncoder().
			AppendInt64(1, 31617863).
			AppendInt32(2, 600).
			AppendString(3, "TestApp").
			AppendString(4, "com.test").
			AppendString(5, "tok123").
			Bytes())
		want := "08c7e6890f10d8041a07546573744170702208636f6d2e746573742a06746f6b313233"
		if got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	})

	t.Run("OrderParam", func(t *testing.T) {
		got := hex.EncodeToString(NewEncoder().
			AppendString(1, "ord-1").
			AppendString(2, "u-1").
			AppendInt64(3, 31617863).
			Bytes())
		want := "0a056f72642d311203752d3118c7e6890f"
		if got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	})
}

func TestDecodeMapEntry(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		entry := NewEncoder().
			AppendString(1, "source").
			AppendString(2, "home").
			Bytes()
		k, v, err := DecodeMapEntry(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if k != "source" || v != "home" {
			t.Fatalf("got (%q, %#v), want (source, home)", k, v)
		}
	})

	t.Run("boxed int32 dynamic value", func(t *testing.T) {
		// Protostuff ObjectSchema tag 5 = int32 (wire tag byte = (5<<3)|0 = 0x28, wait: 0x28 is '(' printable!
		// Test tag 1 = bool (wire tag byte = 0x08, control char -> routes to DecodeDynamicValue).
		boxedBool := NewEncoder().AppendBool(1, true).Bytes()
		entry := NewEncoder().
			AppendString(1, "enabled").
			AppendBytes(2, boxedBool).
			Bytes()
		k, v, err := DecodeMapEntry(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if k != "enabled" || v != true {
			t.Fatalf("got (%q, %#v), want (enabled, true)", k, v)
		}
	})

	t.Run("boxed float64 dynamic value", func(t *testing.T) {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(3.5))
		raw := append(AppendVarint(nil, (8<<3)|uint64(Wire64Bit)), buf[:]...)
		v, err := DecodeDynamicValue(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 3.5 {
			t.Fatalf("got %#v, want 3.5", v)
		}
	})
}

func TestParseRawFieldsGroupAndPoly(t *testing.T) {
	// Build a message with tag 127 poly class + a start/end group (tag 4).
	raw := NewEncoder().
		AppendString(PolyClassTag, "com.heytap.appstore.watch.domain.v2.NavCardDto").
		AppendInt32(1, 1001).
		Bytes()
	raw = AppendVarint(raw, (4<<3)|uint64(WireStartGroup))
	raw = append(raw, NewEncoder().AppendInt32(1, 7).Bytes()...)
	raw = AppendVarint(raw, (4<<3)|uint64(WireEndGroup))

	fields, err := ParseRawFields(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cls := ExtractPolyClassName(fields); cls != "com.heytap.appstore.watch.domain.v2.NavCardDto" {
		t.Fatalf("unexpected poly class: %q", cls)
	}
	if len(fields) != 3 || fields[2].WireType != WireStartGroup || len(fields[2].Group) != 1 {
		t.Fatalf("unexpected parsed fields: %+v", fields)
	}
}
