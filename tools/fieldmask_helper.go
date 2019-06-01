package tools

import (
	"google.golang.org/genproto/protobuf/field_mask"
	"reflect"
	"strings"
	"fmt"
	"github.com/pkg/errors"
)

// FieldFilter is an interface used by the copying function to filter fields that are needed to be copied.
type FieldFilter interface {
	// Filter should return a corresponding FieldFilter for the given fieldName and
	Filter(fieldName string) (FieldFilter, bool)
}

// Mask is a tree-based implementation of a FieldFilter.
type Mask map[string]FieldFilter

// Compile time interface check.
var _ FieldFilter = Mask{}

// Filter returns true for those fieldNames that exist in the underlying map.
// Field names that start with "XXX_" are ignored as unexported.
func (m Mask) Filter(fieldName string) (FieldFilter, bool) {
	if len(m) == 0 {
		// If the mask is empty choose all the exported fields.
		return Mask{}, !strings.HasPrefix(fieldName, "XXX_")
	}
	subFilter, ok := m[fieldName]
	if !ok {
		subFilter = Mask{}
	}
	return subFilter, ok
}

func mapToString(m map[string]FieldFilter) string {
	if len(m) == 0 {
		return ""
	}
	var result []string
	for fieldName, maskNode := range m {
		r := fieldName
		var sub string
		if stringer, ok := maskNode.(fmt.Stringer); ok {
			sub = stringer.String()
		} else {
			sub = fmt.Sprint(maskNode)
		}
		if sub != "" {
			r += "{" + sub + "}"
		}
		result = append(result, r)
	}
	return strings.Join(result, ",")
}

func (m Mask) String() string {
	return mapToString(m)
}

// MaskInverse is an inversed version of a Mask (will copy all the fields except those mentioned in the mask).
type MaskInverse Mask

// Filter returns true for those fieldNames that do NOT exist in the underlying map.
// Field names that start with "XXX_" are ignored as unexported.
func (m MaskInverse) Filter(fieldName string) (FieldFilter, bool) {
	subFilter, ok := m[fieldName]
	if !ok {
		return MaskInverse{}, !strings.HasPrefix(fieldName, "XXX_")
	}
	return subFilter, subFilter != nil
}

func (m MaskInverse) String() string {
	return mapToString(m)
}

// MaskFromProtoFieldMask creates a Mask from the given FieldMask.
func MaskFromProtoFieldMask(fm *field_mask.FieldMask, naming func(string) string) (Mask, error) {
	root := make(Mask)
	for _, path := range fm.GetPaths() {
		mask := root
		for _, fieldName := range strings.Split(path, ".") {
			if fieldName == "" {
				return nil, errors.Errorf("invalid fieldName FieldFilter format: \"%s\"", path)
			}
			newFieldName := naming(fieldName)
			subNode, ok := mask[newFieldName]
			if !ok {
				mask[newFieldName] = make(Mask)
				subNode = mask[newFieldName]
			}
			mask = subNode.(Mask)
		}
	}
	return root, nil
}

// MaskFromString creates a `Mask` from a string `s`.
// `s` is supposed to be a valid string representation of a FieldFilter like "a,b,c{d,e{f,g}},d".
// This is the same string format as in FieldFilter.String(). This function should only be used in tests as it does not
// validate the given string and is only convenient to easily create DefaultMasks.
func MaskFromString(s string) Mask {
	mask, _ := maskFromRunes([]rune(s))
	return mask
}

func maskFromRunes(runes []rune) (Mask, int) {
	mask := make(Mask)
	var fieldName []string
	runes = append(runes, []rune(",")...)
	pos := 0
	for pos < len(runes) {
		char := fmt.Sprintf("%c", runes[pos])
		switch char {
		case " ", "\n", "\t":
			// Ignore white spaces.

		case ",", "{", "}":
			if len(fieldName) == 0 {
				switch char {
				case "}":
					return mask, pos
				case ",":
					pos += 1
					continue
				default:
					panic("invalid mask string format")
				}
			}

			var subMask FieldFilter
			if char == "{" {
				var jump int
				// Parse nested tree.
				subMask, jump = maskFromRunes(runes[pos+1:])
				pos += jump + 1
			} else {
				subMask = make(Mask)
			}
			f := strings.Join(fieldName, "")
			mask[f] = subMask
			// Reset FieldName.
			fieldName = []string{}

			if char == "}" {
				return mask, pos
			}

		default:
			fieldName = append(fieldName, char)
		}
		pos += 1
	}
	return mask, pos
}

// StructToStruct copies `src` struct to `dst` struct using the given FieldFilter.
// Only the fields where FieldFilter returns true will be copied to `dst`.
// `src` and `dst` must be coherent in terms of the field names, but it is not required for them to be of the same type.
func StructToStruct(filter FieldFilter, src, dst interface{}) error {
	srcVal := indirect(reflect.ValueOf(src))
	srcType := srcVal.Type()
	for i := 0; i < srcVal.NumField(); i++ {
		f := srcVal.Field(i)
		fieldName := srcType.Field(i).Name
		subFilter, ok := filter.Filter(fieldName)
		if !ok {
			// Skip this field.
			continue
		}
		if !f.CanSet() {
			return errors.Errorf("Can't set a value on a field %s", fieldName)
		}

		srcField, err := getField(src, fieldName)
		if err != nil {
			return errors.Wrapf(err, "failed to get the field %s from %T", fieldName, src)
		}
		dstField, err := getField(dst, fieldName)
		if err != nil {
			return errors.Wrapf(err, "failed to get the field %s from %T", fieldName, dst)
		}

		dstFieldType := dstField.Type()

		switch dstFieldType.Kind() {
		case reflect.Interface:
			if srcField.IsNil() {
				dstField.Set(reflect.Zero(dstFieldType))
				continue
			}
			if !srcField.Type().Implements(dstFieldType) {
				return errors.Errorf("src %T does not implement dst %T",
					srcField.Interface(), dstField.Interface())
			}

			v := reflect.New(srcField.Elem().Elem().Type())
			if err := StructToStruct(subFilter, srcField.Interface(), v.Interface()); err != nil {
				return err
			}
			dstField.Set(v)

		case reflect.Ptr:
			if srcField.IsNil() {
				dstField.Set(reflect.Zero(dstFieldType))
				continue
			}
			v := reflect.New(dstFieldType.Elem())
			if err := StructToStruct(subFilter, srcField.Interface(), v.Interface()); err != nil {
				return err
			}
			dstField.Set(v)

		case reflect.Array, reflect.Slice:
			// Check if it is an array of values (non-pointers).
			if dstFieldType.Elem().Kind() != reflect.Ptr {
				// Handle this array/slice as a regular non-nested data structure: copy it entirely to dst.
				dstField.Set(srcField)
				continue
			}
			v := reflect.New(dstFieldType).Elem()
			// Iterate over items of the slice/array.
			for i := 0; i < srcField.Len(); i++ {
				subValue := srcField.Index(i)
				newDst := reflect.New(dstFieldType.Elem().Elem())
				if err := StructToStruct(subFilter, subValue.Interface(), newDst.Interface()); err != nil {
					return err
				}
				v.Set(reflect.Append(v, newDst))
			}
			dstField.Set(v)

		default:
			// For primitive data types just copy them entirely.
			dstField.Set(srcField)
		}
	}
	return nil
}

// StructToMap copies `src` struct to the `dst` map.
// Behavior is similar to `StructToStruct`.
func StructToMap(filter FieldFilter, src interface{}, dst map[string]interface{}) error {
	srcVal := indirect(reflect.ValueOf(src))
	srcType := srcVal.Type()
	for i := 0; i < srcVal.NumField(); i++ {
		f := srcVal.Field(i)
		fieldName := srcType.Field(i).Name
		subFilter, ok := filter.Filter(fieldName)
		if !ok {
			// Skip this field.
			continue
		}
		if !f.CanSet() {
			return errors.Errorf("Can't set a value on a field %s", fieldName)
		}
		srcField, err := getField(src, fieldName)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to get the field %s from %T", fieldName, src))
		}
		switch srcField.Kind() {
		case reflect.Ptr, reflect.Interface:
			if srcField.IsNil() {
				dst[fieldName] = nil
				continue
			}
			v := make(map[string]interface{})
			if err := StructToMap(subFilter, srcField.Interface(), v); err != nil {
				return err
			}
			dst[fieldName] = v

		case reflect.Array, reflect.Slice:
			// Check if it is an array of values (non-pointers).
			if srcField.Type().Elem().Kind() != reflect.Ptr {
				// Handle this array/slice as a regular non-nested data structure: copy it entirely to dst.
				if srcField.Len() > 0 {
					dst[fieldName] = srcField.Interface()
				} else {
					dst[fieldName] = []interface{}(nil)
				}
				continue
			}
			v := make([]map[string]interface{}, 0)
			// Iterate over items of the slice/array.
			for i := 0; i < srcField.Len(); i++ {
				subValue := srcField.Index(i)
				newDst := make(map[string]interface{})
				if err := StructToMap(subFilter, subValue.Interface(), newDst); err != nil {
					return err
				}
				v = append(v, newDst)
			}
			dst[fieldName] = v

		default:
			// Set a value on a map.
			dst[fieldName] = srcField.Interface()
		}
	}
	return nil
}

func getField(obj interface{}, name string) (reflect.Value, error) {
	objValue := reflectValue(obj)
	field := objValue.FieldByName(name)
	if !field.IsValid() {
		return reflect.ValueOf(nil), errors.Errorf("no such field: %s in obj %T", name, obj)
	}
	return field, nil
}

func reflectValue(obj interface{}) reflect.Value {
	var val reflect.Value

	if reflect.TypeOf(obj).Kind() == reflect.Ptr {
		val = reflect.ValueOf(obj).Elem()
	} else {
		val = reflect.ValueOf(obj)
	}

	return val
}

func indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}
