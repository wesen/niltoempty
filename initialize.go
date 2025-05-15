package niltoempty

import (
	"fmt"
	"reflect"
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
	fmt.Printf("Initialize called with type: %T\n", obj)
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		fmt.Printf("ERROR: Expected pointer, got %v\n", v.Kind())
		panic("niltoempty: expected pointer")
	}
	fmt.Printf("Starting initialization of type: %v\n", v.Type())

	initializeNils(v, map[uintptr]bool{})
	fmt.Printf("Finished initialization of type: %v\n", v.Type())

	return obj
}

func initializeNils(v reflect.Value, visited map[uintptr]bool) {
	if checkVisited(v, visited) {
		fmt.Printf("Already visited value of type: %v\n", v.Type())
		return
	}

	if !v.IsValid() {
		fmt.Printf("WARNING: Invalid reflect.Value encountered\n")
		return
	}

	fmt.Printf("Processing value of kind: %v, type: %v\n", v.Kind(), v.Type())

	switch v.Kind() {
	case reflect.Pointer:
		fmt.Printf("Handling pointer type: %v\n", v.Type())
		if !v.IsNil() {
			fmt.Printf("Following non-nil pointer to: %v\n", v.Elem().Type())
			initializeNils(v.Elem(), visited)
		} else {
			fmt.Printf("Skipping nil pointer\n")
		}
	case reflect.Slice:
		fmt.Printf("Handling slice type: %v\n", v.Type())
		if v.IsNil() {
			fmt.Printf("Initializing nil slice of type: %v\n", v.Type())
			v.Set(reflect.MakeSlice(v.Type(), 0, 0))
			break
		}

		fmt.Printf("Processing %d slice elements\n", v.Len())
		for i := 0; i < v.Len(); i++ {
			item := v.Index(i)
			fmt.Printf("Processing slice element %d of type: %v\n", i, item.Type())
			initializeNils(item, visited)
		}

	case reflect.Map:
		fmt.Printf("Handling map type: %v\n", v.Type())
		if v.IsNil() {
			fmt.Printf("Initializing nil map of type: %v\n", v.Type())
			v.Set(reflect.MakeMap(v.Type()))
			break
		}

		fmt.Printf("Processing map with %d entries\n", v.Len())
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()
			fmt.Printf("Processing map entry with key: %v\n", key)

			if !val.IsValid() {
				fmt.Printf("WARNING: Invalid map value for key: %v\n", key)
				continue
			}

			elemType := val.Type()
			fmt.Printf("Creating addressable copy of map value type: %v\n", elemType)
			subv := reflect.New(elemType).Elem()
			subv.Set(val)
			initializeNils(subv, visited)
			fmt.Printf("Setting processed value back into map for key: %v\n", key)
			v.SetMapIndex(iter.Key(), subv)
		}

	case reflect.Interface:
		fmt.Printf("Handling interface type: %v\n", v.Type())
		if v.IsNil() {
			fmt.Printf("Skipping nil interface\n")
			break
		}

		valueUnderInterface := reflect.ValueOf(v.Interface())
		elemType := valueUnderInterface.Type()
		fmt.Printf("Creating addressable copy of interface value type: %v\n", elemType)
		subv := reflect.New(elemType).Elem()
		subv.Set(valueUnderInterface)

		initializeNils(subv, visited)
		fmt.Printf("Setting processed value back into interface\n")
		v.Set(subv)

	case reflect.Array:
		fmt.Printf("Handling array type: %v with length %d\n", v.Type(), v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			fmt.Printf("Processing array element %d of type: %v\n", i, elem.Type())
			initializeNils(elem, visited)
		}

	case reflect.Struct:
		fmt.Printf("Handling struct type: %v with %d fields\n", v.Type(), v.NumField())
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fmt.Printf("Processing struct field %d (%s) of type: %v\n", i, v.Type().Field(i).Name, field.Type())
			initializeNils(field, visited)
		}
	default:
		fmt.Printf("Skipping unsupported kind: %v\n", v.Kind())
	}
}

func checkVisited(v reflect.Value, visited map[uintptr]bool) bool {
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
			fmt.Printf("Found already visited pointer: %v\n", p)
		} else {
			fmt.Printf("Marking pointer as visited: %v\n", p)
		}
		visited[p] = true
		return wasVisited
	}
	return false
}
