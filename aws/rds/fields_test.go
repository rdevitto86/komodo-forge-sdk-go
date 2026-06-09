package rds

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
)

// ── Unit Tests ───────────────────────────────────────────────────────────────

func TestToField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		want    types.Field
		wantErr bool
	}{
		{
			name:  "nil becomes IsNull",
			input: nil,
			want:  &types.FieldMemberIsNull{Value: true},
		},
		{
			name:  "string",
			input: "hello",
			want:  &types.FieldMemberStringValue{Value: "hello"},
		},
		{
			name:  "empty string",
			input: "",
			want:  &types.FieldMemberStringValue{Value: ""},
		},
		{
			name:  "int",
			input: int(42),
			want:  &types.FieldMemberLongValue{Value: 42},
		},
		{
			name:  "int32",
			input: int32(100),
			want:  &types.FieldMemberLongValue{Value: 100},
		},
		{
			name:  "int64",
			input: int64(999),
			want:  &types.FieldMemberLongValue{Value: 999},
		},
		{
			name:  "float32",
			input: float32(1.5),
			want:  &types.FieldMemberDoubleValue{Value: float64(float32(1.5))},
		},
		{
			name:  "float64",
			input: float64(3.14),
			want:  &types.FieldMemberDoubleValue{Value: 3.14},
		},
		{
			name:  "bool true",
			input: true,
			want:  &types.FieldMemberBooleanValue{Value: true},
		},
		{
			name:  "bool false",
			input: false,
			want:  &types.FieldMemberBooleanValue{Value: false},
		},
		{
			name:  "[]byte",
			input: []byte{0x01, 0x02, 0x03},
			want:  &types.FieldMemberBlobValue{Value: []byte{0x01, 0x02, 0x03}},
		},
		{
			name:    "struct returns error",
			input:   struct{ X int }{X: 1},
			wantErr: true,
		},
		{
			name:    "map returns error",
			input:   map[string]int{"a": 1},
			wantErr: true,
		},
		{
			name:    "slice of ints returns error",
			input:   []int{1, 2, 3},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := toField(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Compare via type assertion since Field is an interface.
			if got == nil {
				t.Fatalf("got nil Field")
			}
			switch want := tc.want.(type) {
			case *types.FieldMemberIsNull:
				v, ok := got.(*types.FieldMemberIsNull)
				if !ok || v.Value != want.Value {
					t.Fatalf("got %T(%v), want %T(%v)", got, got, tc.want, tc.want)
				}
			case *types.FieldMemberStringValue:
				v, ok := got.(*types.FieldMemberStringValue)
				if !ok || v.Value != want.Value {
					t.Fatalf("got %T(%v), want %T(%v)", got, got, tc.want, tc.want)
				}
			case *types.FieldMemberLongValue:
				v, ok := got.(*types.FieldMemberLongValue)
				if !ok || v.Value != want.Value {
					t.Fatalf("got %T(%v), want %T(%v)", got, got, tc.want, tc.want)
				}
			case *types.FieldMemberDoubleValue:
				v, ok := got.(*types.FieldMemberDoubleValue)
				if !ok || v.Value != want.Value {
					t.Fatalf("got %T(%v), want %T(%v)", got, got, tc.want, tc.want)
				}
			case *types.FieldMemberBooleanValue:
				v, ok := got.(*types.FieldMemberBooleanValue)
				if !ok || v.Value != want.Value {
					t.Fatalf("got %T(%v), want %T(%v)", got, got, tc.want, tc.want)
				}
			case *types.FieldMemberBlobValue:
				v, ok := got.(*types.FieldMemberBlobValue)
				if !ok {
					t.Fatalf("got %T, want *FieldMemberBlobValue", got)
				}
				if string(v.Value) != string(want.Value) {
					t.Fatalf("blob mismatch: got %v, want %v", v.Value, want.Value)
				}
			default:
				t.Fatalf("unhandled want type %T", tc.want)
			}
		})
	}
}

func TestFromField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   types.Field
		want    any
		wantErr bool
	}{
		{
			name:  "StringValue",
			input: &types.FieldMemberStringValue{Value: "world"},
			want:  "world",
		},
		{
			name:  "LongValue",
			input: &types.FieldMemberLongValue{Value: 42},
			want:  int64(42),
		},
		{
			name:  "DoubleValue",
			input: &types.FieldMemberDoubleValue{Value: 2.718},
			want:  float64(2.718),
		},
		{
			name:  "BooleanValue true",
			input: &types.FieldMemberBooleanValue{Value: true},
			want:  true,
		},
		{
			name:  "BooleanValue false",
			input: &types.FieldMemberBooleanValue{Value: false},
			want:  false,
		},
		{
			name:  "BlobValue",
			input: &types.FieldMemberBlobValue{Value: []byte{0xDE, 0xAD}},
			want:  []byte{0xDE, 0xAD},
		},
		{
			name:  "IsNull returns nil",
			input: &types.FieldMemberIsNull{Value: true},
			want:  nil,
		},
		{
			name:    "ArrayValue returns error",
			input:   &types.FieldMemberArrayValue{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := fromField(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Use string comparison for []byte, direct == for scalars.
			switch want := tc.want.(type) {
			case []byte:
				v, ok := got.([]byte)
				if !ok {
					t.Fatalf("got %T, want []byte", got)
				}
				if string(v) != string(want) {
					t.Fatalf("blob mismatch: got %v, want %v", v, want)
				}
			default:
				if got != tc.want {
					t.Fatalf("got %v (%T), want %v (%T)", got, got, tc.want, tc.want)
				}
			}
		})
	}
}
