package converter

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

func ConvertToGrpcMap(data map[string]any) (map[string]*structpb.Value, error) {
	root, err := convertToValueGrpcMap(data)
	if err != nil {
		return nil, err
	}
	structVal := root.GetStructValue()
	if structVal == nil {
		return nil, fmt.Errorf("root value is not a struct")
	}
	return structVal.Fields, nil
}

func convertToValueGrpcMap(v any) (*structpb.Value, error) {
	switch val := v.(type) {
	case map[string]any:
		fields := make(map[string]*structpb.Value, len(val))
		for k, v2 := range val {
			pbVal, err := convertToValueGrpcMap(v2)
			if err != nil {
				return nil, err
			}
			fields[k] = pbVal
		}
		return structpb.NewStructValue(&structpb.Struct{
			Fields: fields,
		}), nil
	case []any:
		values := make([]*structpb.Value, len(val))
		for i, elem := range val {
			pbVal, err := convertToValueGrpcMap(elem)
			if err != nil {
				return nil, err
			}
			values[i] = pbVal
		}
		return structpb.NewListValue(&structpb.ListValue{
			Values: values,
		}), nil
	default:
		return structpb.NewValue(val)
	}
}
