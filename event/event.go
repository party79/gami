//Package event decoder
//This Build Type of Event received
package event

import (
	"fmt"
	"log"
	"reflect"
	"strconv"

	"github.com/party79/gami"
)

// eventTrap used internal for trap events and cast
var eventTrap = make(map[string]reflect.Type)

func RegisterEvent(x interface{}, name string) {
	if _, ok := eventTrap[name]; ok {
		// TODO: Some day, make this a panic.
		log.Printf("event: duplicate event trap registered: %s", name)
		return
	}
	t := reflect.TypeOf(x)
	eventTrap[name] = t
}

//New build a new event Type if not return the AMIEvent
func New(event *gami.AMIEvent) interface{} {
	if mt, ok := eventTrap[event.ID]; ok {
		elem := reflect.New(mt).Elem()
		if b, ok := elem.Interface().(Builder); ok {
			if elem.Kind() == reflect.Ptr && elem.IsNil() {
				elem.Set(reflect.New(mt.Elem()))
				b = elem.Interface().(Builder)
			}
			b.BuildEvent(event)
			return b
		}
		return Build(event, mt)
	}
	return *event
}

type Builder interface {
	BuildEvent(event *gami.AMIEvent)
}

func Build(event *gami.AMIEvent, mt reflect.Type) interface{} {
	ret := reflect.New(mt).Elem()
	value := ret
	if ret.Kind() == reflect.Ptr {
		if ret.IsNil() {
			ret.Set(reflect.New(mt.Elem()))
		}
		value = ret.Elem()
	}
	typ := value.Type()
	for ix := 0; ix < value.NumField(); ix++ {
		field := value.Field(ix)
		tfield := typ.Field(ix)

		name := tfield.Name
		if tag := tfield.Tag.Get("AMI"); tag != "" {
			name = tag
		}
		if name == "-" {
			continue
		}

		if name == "Privilege" {
			field.Set(reflect.ValueOf(event.Privilege))
			continue
		}
		switch field.Kind() {
		case reflect.String:
			field.SetString(event.Params[name])
		case reflect.Int64:
			vint, _ := strconv.Atoi(event.Params[name])
			field.SetInt(int64(vint))
		default:
			fmt.Print(ix, name, ":", field, "\n")
		}

	}
	return ret.Interface()
}
