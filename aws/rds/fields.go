package rds

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
)

func toField(v any) (types.Field, error) {
	if v == nil {
		return &types.FieldMemberIsNull{Value: true}, nil
	}
	switch val := v.(type) {
	case string:
		return &types.FieldMemberStringValue{Value: val}, nil
	case int:
		return &types.FieldMemberLongValue{Value: int64(val)}, nil
	case int32:
		return &types.FieldMemberLongValue{Value: int64(val)}, nil
	case int64:
		return &types.FieldMemberLongValue{Value: val}, nil
	case float32:
		return &types.FieldMemberDoubleValue{Value: float64(val)}, nil
	case float64:
		return &types.FieldMemberDoubleValue{Value: val}, nil
	case bool:
		return &types.FieldMemberBooleanValue{Value: val}, nil
	case []byte:
		return &types.FieldMemberBlobValue{Value: val}, nil
	default:
		return nil, fmt.Errorf("unsupported parameter type %T: use string, int, int32, int64, float32, float64, bool, []byte, or nil", v)
	}
}

// Converts an RDS Data API Field union back to a Go value. Returns an error for array fields.
func fromField(f types.Field) (any, error) {
	switch v := f.(type) {
	case *types.FieldMemberStringValue:
		return v.Value, nil
	case *types.FieldMemberLongValue:
		return v.Value, nil
	case *types.FieldMemberDoubleValue:
		return v.Value, nil
	case *types.FieldMemberBooleanValue:
		return v.Value, nil
	case *types.FieldMemberBlobValue:
		return v.Value, nil
	case *types.FieldMemberIsNull:
		return nil, nil
	case *types.FieldMemberArrayValue:
		return nil, fmt.Errorf("array fields are not supported in v1: use scalar types only")
	default:
		return nil, fmt.Errorf("unknown field type %T", f)
	}
}
