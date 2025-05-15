package niltoempty

import (
	"fmt"
	"reflect"
	"strings"
)

// Initialize traverses any addressable entity and replaces all nil maps and slices
// with empty map and slices respectively.
//
// Because input object have to be addressable in order to make changes Initialize
// panics when non-addressable object is passed as argument.
//
// Because pointer to element is usually used for modeling optional fields
// nil pointers to the map or slices are left untouched.
func Initialize(obj interface{}) interface{} {
	fmt.Printf("Initialize called with type: %T (concrete type: %v)\n", obj, reflect.TypeOf(obj))
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		fmt.Printf("ERROR: Expected pointer, got %v\n", v.Kind())
		panic("niltoempty: expected pointer")
	}
	fmt.Printf("Starting initialization of type: %v\n", v.Type())

	initializeNils(v, map[uintptr]bool{}, "", 0)
	fmt.Printf("Finished initialization of type: %v\n", v.Type())

	return obj
}

func indent(depth int) string {
	return strings.Repeat("  ", depth)
}

func getFieldPath(currentPath string, field string) string {
	if currentPath == "" {
		return field
	}
	return currentPath + "." + field
}

func initializeNils(v reflect.Value, visited map[uintptr]bool, path string, depth int) {
	prefix := indent(depth)

	if checkVisited(v, visited, depth) {
		fmt.Printf("%sAlready visited value at path '%s' of type: %v\n", prefix, path, v.Type())
		return
	}

	if !v.IsValid() {
		fmt.Printf("%sWARNING: Invalid reflect.Value encountered at path '%s'\n", prefix, path)
		return
	}

	fmt.Printf("%sProcessing value at path '%s' of kind: %v, type: %v\n", prefix, path, v.Kind(), v.Type())

	switch v.Kind() {
	case reflect.Pointer:
		fmt.Printf("%sHandling pointer at path '%s' type: %v -> %v\n", prefix, path, v.Type(), v.Type().Elem())
		if !v.IsNil() {
			fmt.Printf("%sFollowing non-nil pointer at path '%s' to: %v\n", prefix, path, v.Elem().Type())
			initializeNils(v.Elem(), visited, path, depth+1)
		} else {
			fmt.Printf("%sSkipping nil pointer at path '%s'\n", prefix, path)
		}
	case reflect.Slice:
		fmt.Printf("%sHandling slice at path '%s' type: %v (element type: %v)\n", prefix, path, v.Type(), v.Type().Elem())
		if v.IsNil() {
			fmt.Printf("%sInitializing nil slice at path '%s' of type: %v\n", prefix, path, v.Type())
			v.Set(reflect.MakeSlice(v.Type(), 0, 0))
			break
		}

		fmt.Printf("%sProcessing %d slice elements at path '%s'\n", prefix, v.Len(), path)
		for i := 0; i < v.Len(); i++ {
			item := v.Index(i)
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			fmt.Printf("%sProcessing slice element at path '%s' of type: %v\n", prefix, itemPath, item.Type())
			initializeNils(item, visited, itemPath, depth+1)
		}

	case reflect.Map:
		fmt.Printf("%sHandling map at path '%s' type: %v (key: %v, value: %v)\n", prefix, path, v.Type(), v.Type().Key(), v.Type().Elem())
		if v.IsNil() {
			fmt.Printf("%sInitializing nil map at path '%s' of type: %v\n", prefix, path, v.Type())
			v.Set(reflect.MakeMap(v.Type()))
			break
		}

		fmt.Printf("%sProcessing map with %d entries at path '%s'\n", prefix, v.Len(), path)
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()
			keyStr := fmt.Sprintf("%v", key.Interface())
			mapItemPath := fmt.Sprintf("%s[%v]", path, keyStr)
			fmt.Printf("%sProcessing map entry at path '%s' with key type: %v\n", prefix, mapItemPath, key.Type())

			if !val.IsValid() {
				fmt.Printf("%sWARNING: Invalid map value at path '%s' for key: %v\n", prefix, mapItemPath, key)
				continue
			}

			elemType := val.Type()
			fmt.Printf("%sCreating addressable copy at path '%s' of map value type: %v\n", prefix, mapItemPath, elemType)
			subv := reflect.New(elemType).Elem()
			subv.Set(val)
			initializeNils(subv, visited, mapItemPath, depth+1)
			fmt.Printf("%sSetting processed value back into map at path '%s' for key: %v\n", prefix, mapItemPath, key)
			v.SetMapIndex(iter.Key(), subv)
		}

	case reflect.Interface:
		fmt.Printf("%sHandling interface at path '%s' type: %v\n", prefix, path, v.Type())
		if v.IsNil() {
			fmt.Printf("%sSkipping nil interface at path '%s'\n", prefix, path)
			break
		}

		valueUnderInterface := reflect.ValueOf(v.Interface())
		elemType := valueUnderInterface.Type()
		fmt.Printf("%sCreating addressable copy at path '%s' of interface value type: %v\n", prefix, path, elemType)
		subv := reflect.New(elemType).Elem()
		subv.Set(valueUnderInterface)

		initializeNils(subv, visited, path, depth+1)
		fmt.Printf("%sSetting processed value back into interface at path '%s'\n", prefix, path)
		v.Set(subv)

	case reflect.Array:
		fmt.Printf("%sHandling array at path '%s' type: %v with length %d (element type: %v)\n", prefix, path, v.Type(), v.Len(), v.Type().Elem())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			arrayItemPath := fmt.Sprintf("%s[%d]", path, i)
			fmt.Printf("%sProcessing array element at path '%s' of type: %v\n", prefix, arrayItemPath, elem.Type())
			initializeNils(elem, visited, arrayItemPath, depth+1)
		}

	case reflect.Struct:
		fmt.Printf("%sHandling struct at path '%s' type: %v with %d fields\n", prefix, path, v.Type(), v.NumField())
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldType := v.Type().Field(i)
			fieldPath := getFieldPath(path, fieldType.Name)
			fmt.Printf("%sProcessing struct field at path '%s' name: '%s' type: %v tags: '%v'\n",
				prefix, fieldPath, fieldType.Name, field.Type(), fieldType.Tag)
			initializeNils(field, visited, fieldPath, depth+1)
		}
	default:
		fmt.Printf("%sSkipping unsupported kind at path '%s': %v\n", prefix, path, v.Kind())
	}
}

func checkVisited(v reflect.Value, visited map[uintptr]bool, depth int) bool {
	prefix := indent(depth)
	if !v.IsValid() {
		return false
	}

	kind := v.Kind()
	if kind == reflect.Map || kind == reflect.Ptr || kind == reflect.Slice {
		if v.IsNil() {
			return false
		}
		p := v.Pointer()
		wasVisited := visited[p]
		if wasVisited {
			fmt.Printf("%sFound already visited pointer: %v\n", prefix, p)
		} else {
			fmt.Printf("%sMarking pointer as visited: %v\n", prefix, p)
		}
		visited[p] = true
		return wasVisited
	}
	return false
}
