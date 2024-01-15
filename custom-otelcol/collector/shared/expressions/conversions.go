package expressions

import (
	"fmt"
	"reflect"
)

func flattenSliceStr(arr []any) []string {
	result := []string{}
	for _, item := range arr {
		rt := reflect.TypeOf(item)
		switch rt.Kind() {
		case reflect.String:
			result = append(result, item.(string))
		case reflect.Slice:
			flattened := flattenSliceStr(item.([]any))
			result = append(result, flattened...)
		}
	}
	return result
}

func flattenSliceAny(arr []any) []any {
	fmt.Printf("Flattening %v\n", arr)
	result := []any{}
	for _, item := range arr {
		fmt.Printf("Processing %v\n", item)
		rt := reflect.TypeOf(item)
		fmt.Printf("Processing %v of type %v\n", item, rt.Kind())
		switch rt.Kind() {
		case reflect.String:
			result = append(result, item.(string))
		case reflect.Slice:
			flattened := flattenSliceAny(item.([]any))
			result = append(result, flattened...)
		}
		fmt.Printf("Processed %v\n", result)
	}
	return result
}

func removeDuplicateValues(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
