package event

import (
	"reflect"
	"testing"
)

//Util para hacer pruebas de eventos
func testEvent(t *testing.T, fixture map[string]string, evtype interface{}) {
	value := reflect.ValueOf(evtype)
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			t.Fatal("Data is nil")
		}
		value = value.Elem()
	}
	typ := value.Type()
	for k, v := range fixture {
		field, tfield := findField(value, typ, k, v)
		if !field.IsValid() || field.String() == "" {
			t.Fatalf("Not Cast Field: %q", v)
		}

		if tfield.Tag.Get("AMI") != k {
			t.Fatalf("Not Cast AMI Field: %q from %q", k, v)
		}
	}
}

func findField(value reflect.Value, typ reflect.Type, k string, v string) (reflect.Value, reflect.StructField) {
	field := value.FieldByName(v)
	if field.IsValid() {
		tfield, _ := typ.FieldByName(v)
		return field, tfield
	}
	for ix := 0; ix < value.NumField(); ix++ {
		field := value.Field(ix)
		tfield := typ.Field(ix)
		if tfield.Tag.Get("AMI") == k && field.String() == v {
			return field, tfield
		}
	}
	return field, reflect.StructField{}
}
