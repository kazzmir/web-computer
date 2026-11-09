package py

import "reflect"

// Identical compares native-backed object identity without applying __eq__ or
// attempting Go interface equality on a map/slice (which would panic).
func Identical(a, b Object) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	left, right := reflect.ValueOf(a), reflect.ValueOf(b)
	if left.Type() != right.Type() {
		return false
	}
	switch left.Kind() {
	case reflect.Map:
		return left.UnsafePointer() == right.UnsafePointer()
	case reflect.Slice:
		return left.Len() == right.Len() && left.UnsafePointer() == right.UnsafePointer()
	default:
		if left.Comparable() {
			return left.Equal(right)
		}
		return false
	}
}
