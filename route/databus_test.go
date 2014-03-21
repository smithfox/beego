package route

import (
	"reflect"
	"testing"
)

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

type Embstruct struct {
	Dep2 int
}
type TestStruct struct {
	Dep1 string
	*Embstruct
	//Dep2 int
	Dep3 int
}

func Test_PtrStruct(t *testing.T) {
	td := &TestStruct{}
	td.Dep1 = "Hello"
	td.Embstruct = &Embstruct{}
	td.Dep2 = 76
	td.Dep3 = 87
	bus := WrapperBus(td)
	d1 := bus.Get("Dep1")
	d2 := bus.Get("Dep2")
	d3 := bus.Get("Dep3")
	expect(t, d1, "Hello")
	expect(t, d2, 76)
	expect(t, d3, 87)

	bus.Set("Dep1", "World")
	bus.Set("Dep2", 77)
	bus.Set("Dep3", 85)

	expect(t, td.Dep1, "World")
	expect(t, td.Dep2, 77)
	expect(t, td.Dep3, 85)

	bus.Set("Embstruct", nil)
	if td.Embstruct != nil {
		t.Errorf("!!not is nil\n")
	}

	if !bus.Has("Dep1") {
		t.Errorf("!!has aaaa\n")
	}

	if bus.Has("aaaaa") {
		t.Errorf("!!not has aaaa\n")
	}
	//expect(t, td.Embstruct, nil)
}
