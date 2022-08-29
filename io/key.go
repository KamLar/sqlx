package io

func NormalizeKey(key interface{}) interface{} {
	if key == nil {
		return nil
	}

	switch actual := key.(type) {
	case *int64:
		if actual == nil {
			return nil
		}
		return int(*actual)
	case *int32:
		if actual == nil {
			return nil
		}
		return int(*actual)

	case *int16:
		if actual == nil {
			return nil
		}
		return int(*actual)

	case int32:
		return int(actual)
	case int64:
		return int(actual)
	case int16:
		return int(actual)
	case *int:
		if actual == nil {
			return nil
		}
		return int(*actual)
	case *string:
		if actual == nil {
			return nil
		}
		return *actual
	}
	return key
}
