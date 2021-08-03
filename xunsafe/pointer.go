package xunsafe

import (
	"fmt"
	"reflect"
	"unsafe"
)

//Pointer represents a func returning field value pointer, it takes holder address
type Pointer func(structPtr uintptr) interface{}




//FieldPointer create Pointer function for supported field or error
func FieldPointer(structType reflect.Type, fieldPath *Field) (Pointer, error) {
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct but had: %T", reflect.New(structType))
	}
	field := structType.Field(fieldPath.Index)
	offset := field.Offset
	var result Pointer
	if fieldPath.Getter != nil {
		return func(structAddr uintptr) interface{} {
			return fieldPath.Getter(structAddr)
		}, nil
	}
	switch field.Type.Kind() {
	case reflect.Int:
		result = func(structAddr uintptr) interface{} {
			return (*int)(unsafe.Pointer(structAddr + offset))
		}
	case reflect.Uint:
		result = func(structAddr uintptr) interface{} {
			return (*uint)(unsafe.Pointer(structAddr + offset))
		}
	case reflect.Int64:
		result = func(structAddr uintptr) interface{} {
			return (*int64)(unsafe.Pointer(structAddr + offset))
		}
	case reflect.Int32:
		result = func(structAddr uintptr) interface{} {
			return (*int32)(unsafe.Pointer(structAddr + offset))
		}
	case reflect.Int16:
		result = func(structAddr uintptr) interface{} {
			return (*int16)(unsafe.Pointer(structAddr + offset))
		}
	case reflect.Int8:
		result = func(structAddr uintptr) interface{} {
			return (*int8)(unsafe.Pointer(structAddr + offset))
		}
	case reflect.Uint64:
		result = func(structAddr uintptr) interface{} {
			return (*uint64)(unsafe.Pointer(structAddr + offset))
		}
	case reflect.Uint32:
		result = func(structAddr uintptr) interface{} {
			return (*uint32)(unsafe.Pointer(structAddr + offset))
		}
	case reflect.Uint16:
		result = func(structAddr uintptr) interface{} {
			return (*uint16)(unsafe.Pointer(structAddr + offset))
		}
	case reflect.Uint8:
		result = func(structAddr uintptr) interface{} {
			return (*uint8)(unsafe.Pointer(structAddr + offset))
		}
	case reflect.String:
		result = func(structAddr uintptr) interface{} {
			return (*string)(unsafe.Pointer(structAddr + offset))
		}
	case reflect.Float64:
		result = func(structAddr uintptr) interface{} {
			return (*float64)(unsafe.Pointer(structAddr + offset))
		}

	case reflect.Float32:
		result = func(structAddr uintptr) interface{} {
			return (*float32)(unsafe.Pointer(structAddr + offset))
		}
	case reflect.Bool:
		result = func(structAddr uintptr) interface{} {
			return (*bool)(unsafe.Pointer(structAddr + offset))
		}

	case reflect.Struct:
		if fieldPath.Field == nil {
			return nil, fmt.Errorf("failed to get pointer on %v.%v, subPath was nil", structType.String(), field.Name)
		}
		fn, err := FieldPointer(field.Type, fieldPath.Field)
		if err != nil {
			return nil, fmt.Errorf("failed to get poiner on %v.%v due to %w", structType.String(), field.Name, err)
		}
		result = func(structAddr uintptr) interface{} {
			fieldValue := reflect.NewAt(field.Type, unsafe.Pointer(structAddr+offset))
			addr := fieldValue.Elem().UnsafeAddr()
			return fn(addr)
		}

	case reflect.Ptr:
		switch field.Type.Elem().Kind() {
		case reflect.Struct:
			if fieldPath.Field == nil {
				return nil, fmt.Errorf("failed to get pointer on %v.%v, subPath was nil", structType.String(), field.Name)
			}
			fn, err := FieldPointer(field.Type.Elem(), fieldPath.Field)
			if err != nil {
				return nil, fmt.Errorf("failed to get poiner on %v.%v due to %w", structType.String(), field.Name, err)
			}
			result = func(structAddr uintptr) interface{} {
				fieldValue := reflect.NewAt(field.Type, unsafe.Pointer(structAddr+offset))
				if fieldValue.Elem().IsNil() {
					ptr := reflect.New(fieldValue.Type().Elem().Elem())
					fieldValue.Elem().Set(ptr)
				}
				return fn(fieldValue.Elem().Elem().UnsafeAddr())
			}

		case reflect.Int:
			result = func(structAddr uintptr) interface{} {
				return (**int)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Uint:
			result = func(structAddr uintptr) interface{} {
				return (**uint)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Int64:
			result = func(structAddr uintptr) interface{} {
				return (**int64)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Int32:
			result = func(structAddr uintptr) interface{} {
				return (**int32)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Int16:
			result = func(structAddr uintptr) interface{} {
				return (**int16)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Int8:
			result = func(structAddr uintptr) interface{} {
				return (**int8)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Uint64:
			result = func(structAddr uintptr) interface{} {
				return (**uint64)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Uint32:
			result = func(structAddr uintptr) interface{} {
				return (**uint32)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Uint16:
			result = func(structAddr uintptr) interface{} {
				return (**uint16)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Uint8:
			result = func(structAddr uintptr) interface{} {
				return (**uint8)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.String:
			result = func(structAddr uintptr) interface{} {
				return (**string)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Float64:
			result = func(structAddr uintptr) interface{} {
				return (**float64)(unsafe.Pointer(structAddr + offset))
			}

		case reflect.Float32:
			result = func(structAddr uintptr) interface{} {
				return (**float32)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Bool:
			result = func(structAddr uintptr) interface{} {
				return (**bool)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Slice:
			switch field.Type.Elem().Elem().Kind() {
			case reflect.Int:
				result = func(structAddr uintptr) interface{} {
					return (**[]int)(unsafe.Pointer(structAddr + offset))
				}
			case reflect.Uint:
				result = func(structAddr uintptr) interface{} {
					return (**[]uint)(unsafe.Pointer(structAddr + offset))
				}
			case reflect.Int64:
				result = func(structAddr uintptr) interface{} {
					return (**[]int64)(unsafe.Pointer(structAddr + offset))
				}
			case reflect.Int32:
				result = func(structAddr uintptr) interface{} {
					return (**[]int32)(unsafe.Pointer(structAddr + offset))
				}
			case reflect.Int16:
				result = func(structAddr uintptr) interface{} {
					return (**[]int16)(unsafe.Pointer(structAddr + offset))
				}
			case reflect.Int8:
				result = func(structAddr uintptr) interface{} {
					return (**[]int8)(unsafe.Pointer(structAddr + offset))
				}
			case reflect.Uint64:
				result = func(structAddr uintptr) interface{} {
					return (**[]uint64)(unsafe.Pointer(structAddr + offset))
				}
			case reflect.Uint32:
				result = func(structAddr uintptr) interface{} {
					return (**[]uint32)(unsafe.Pointer(structAddr + offset))
				}
			case reflect.Uint16:
				result = func(structAddr uintptr) interface{} {
					return (**[]uint16)(unsafe.Pointer(structAddr + offset))
				}
			case reflect.Uint8:
				result = func(structAddr uintptr) interface{} {
					return (**[]uint8)(unsafe.Pointer(structAddr + offset))
				}
			case reflect.String:
				result = func(structAddr uintptr) interface{} {
					return (**[]string)(unsafe.Pointer(structAddr + offset))
				}
			case reflect.Float64:
				result = func(structAddr uintptr) interface{} {
					return (**[]float64)(unsafe.Pointer(structAddr + offset))
				}

			case reflect.Float32:
				result = func(structAddr uintptr) interface{} {
					return (**[]float32)(unsafe.Pointer(structAddr + offset))
				}
			case reflect.Bool:
				result = func(structAddr uintptr) interface{} {
					return (**[]bool)(unsafe.Pointer(structAddr + offset))
				}
			default:
				return raiseUnsupportedTypeError(structType, field)
			}

		default:
			return raiseUnsupportedTypeError(structType, field)
		}
	case reflect.Slice:
		switch field.Type.Elem().Kind() {
		case reflect.Int:
			result = func(structAddr uintptr) interface{} {
				return (*[]int)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Uint:
			result = func(structAddr uintptr) interface{} {
				return (*[]uint)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Int64:
			result = func(structAddr uintptr) interface{} {
				return (*[]int64)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Int32:
			result = func(structAddr uintptr) interface{} {
				return (*[]int32)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Int16:
			result = func(structAddr uintptr) interface{} {
				return (*[]int16)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Int8:
			result = func(structAddr uintptr) interface{} {
				return (*[]int8)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Uint64:
			result = func(structAddr uintptr) interface{} {
				return (*[]uint64)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Uint32:
			result = func(structAddr uintptr) interface{} {
				return (*[]uint32)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Uint16:
			result = func(structAddr uintptr) interface{} {
				return (*[]uint16)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Uint8:
			result = func(structAddr uintptr) interface{} {
				return (*[]uint8)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.String:
			result = func(structAddr uintptr) interface{} {
				return (*[]string)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Float64:
			result = func(structAddr uintptr) interface{} {
				return (*[]float64)(unsafe.Pointer(structAddr + offset))
			}

		case reflect.Float32:
			result = func(structAddr uintptr) interface{} {
				return (*[]float32)(unsafe.Pointer(structAddr + offset))
			}
		case reflect.Bool:
			result = func(structAddr uintptr) interface{} {
				return (*[]bool)(unsafe.Pointer(structAddr + offset))
			}
		default:
			return raiseUnsupportedTypeError(structType, field)
		}
	default:
		return raiseUnsupportedTypeError(structType, field)
	}
	return result, nil
}

func raiseUnsupportedTypeError(holder reflect.Type, field reflect.StructField) (Pointer, error) {
	return nil, fmt.Errorf("unsupported type: %v, at %T.%s", field.Type.Name(), reflect.New(holder).Interface(), field.Name)
}
