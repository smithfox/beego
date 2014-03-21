package route

import (
	// "fmt"
	"reflect"
)

var gInjectFieldCache map[reflect.Type][]string = make(map[reflect.Type][]string)

type Databus interface {
	Type(name string) reflect.Type
	Value(name string) reflect.Value
	Has(name string) bool
	Get(name string) interface{}
	Set(name string, v interface{})
	Fields() []string
}

type databus struct {
	wv reflect.Value
	wt reflect.Type
}

func cacheInjectFields(t reflect.Type) {
	if _, ok := gInjectFieldCache[t]; ok {
		return
	}

	fs := []string{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Tag.Get("bmgo") == "databus" {
			fs = append(fs, f.Name)
		}
	}

	if len(fs) > 0 {
		gInjectFieldCache[t] = fs
	}
}

func WrapperBusValue(v reflect.Value) Databus {
	wv := v
	wt := reflect.TypeOf(v)

	if wv.Kind() == reflect.Ptr {
		wv = wv.Elem()
		wt = wv.Type()
	} else {
		//fmt.Printf("WrapperBusValue,v.kind=%s\n", wv.Kind().String())
		panic("WrapperBus v must Pointer")
	}
	cacheInjectFields(wt)
	return &databus{wv: wv, wt: wt}
}

func WrapperBus(v interface{}) Databus {
	wv := reflect.ValueOf(v)
	wt := reflect.TypeOf(v)

	if wv.Kind() == reflect.Ptr {
		wv = wv.Elem()
		wt = wv.Type()
		return &databus{wv: wv, wt: wt}
	} else {
		panic("WrapperBus v must Pointer")
	}
}

func (bus *databus) Type(name string) reflect.Type {
	f := bus.wv.FieldByName(name)
	return f.Type()
}

func (bus *databus) Value(name string) reflect.Value {
	f := bus.wv.FieldByName(name)
	return f
}

func (bus *databus) Has(name string) bool {
	f := bus.wv.FieldByName(name)
	return f.Kind() != reflect.Invalid
}

func (bus *databus) Get(name string) interface{} {
	f := bus.wv.FieldByName(name)
	if f.IsValid() {
		return f.Interface()
	} else {
		return nil
	}
}

func (bus *databus) Set(name string, v interface{}) {
	f := bus.wv.FieldByName(name)
	///vv := reflect.ValueOf(v)
	if v == nil {
		f.Set(reflect.Zero(f.Type()))
		//f.Set(nil)
	} else {
		f.Set(reflect.ValueOf(v))
	}
}

func (bus *databus) Fields() []string {
	s, _ := gInjectFieldCache[bus.wt]
	return s
}
