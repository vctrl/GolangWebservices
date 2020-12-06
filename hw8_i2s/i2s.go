package main

import (
	"reflect"
)

func main() {

}

type TypeMismatchError struct {
	message string
}

func (e TypeMismatchError) Error() string {
	return e.message
}

func NewTypeMismatchError() *TypeMismatchError {
	return &TypeMismatchError{message: "some not very informative message"}
}

func i2s(data interface{}, out interface{}) error {
	if reflect.TypeOf(out).Kind() != reflect.Ptr {
		return NewTypeMismatchError()
	}
	err := goUnder(data, reflect.ValueOf(out).Elem())
	if err != nil {
		return err
	}

	return nil
}

func goUnder(data interface{}, field reflect.Value) error {
	v := reflect.ValueOf(data)
	inputKind := reflect.TypeOf(field.Interface()).Kind()

	switch reflect.TypeOf(data).Kind() {
	case reflect.Slice:
		if inputKind != reflect.Slice {
			return NewTypeMismatchError()
		}

		slice := reflect.MakeSlice(field.Type(), v.Len(), v.Len())
		field.Set(slice)
		for i := 0; i < v.Len(); i++ {
			err := goUnder(v.Index(i).Interface(), slice.Index(i))
			if err != nil {
				return err
			}
		}
	case reflect.Map:
		if inputKind != reflect.Struct {
			return NewTypeMismatchError()
		}

		for _, key := range v.MapKeys() {
			err := goUnder(v.MapIndex(key).Interface(), field.FieldByName(key.String()))
			if err != nil {
				return err
			}
		}

	case reflect.Float64:
		if inputKind != reflect.Int {
			return NewTypeMismatchError()
		}

		field.SetInt(int64(data.(float64)))
	case reflect.String:
		if inputKind != reflect.String {
			return NewTypeMismatchError()
		}

		field.SetString(data.(string))
	case reflect.Bool:
		if inputKind != reflect.Bool {
			return NewTypeMismatchError()
		}

		field.SetBool(data.(bool))
	}

	return nil
}
