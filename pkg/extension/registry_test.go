package extension

import (
	"testing"
)

func TestRegistry(t *testing.T) {
	var registry Registry
	valid := Intrinsic{
		Namespace: "test", Name: "combine",
		Signature: Signature{Parameters: []Type{Qubit, Qubit}, Return: Void, Distinct: [][2]int{{0, 1}}},
		Lower:     func(LoweringContext, []Value) (Value, error) { return Value{Type: Void}, nil },
	}
	for _, mutate := range []func(*Intrinsic){
		func(i *Intrinsic) { i.Namespace = "bad namespace" },
		func(i *Intrinsic) { i.Name = "fn" },
		func(i *Intrinsic) { i.Lower = nil },
		func(i *Intrinsic) { i.Signature.Return = Qubit },
		func(i *Intrinsic) { i.Signature.Return = Int },
		func(i *Intrinsic) { i.Signature.Parameters[0] = Void },
		func(i *Intrinsic) { i.Signature.Distinct[0] = [2]int{0, 2} },
		func(i *Intrinsic) { i.Signature.Distinct[0] = [2]int{0, 0} },
	} {
		invalid := clone(valid)
		mutate(&invalid)
		if err := registry.Register(invalid); err == nil {
			t.Fatalf("accepted invalid intrinsic %+v", invalid)
		}
	}
	if err := registry.Register(valid); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(valid); err == nil {
		t.Fatal("accepted duplicate")
	}
	if _, ok := registry.Lookup("test", valid.Name); ok {
		t.Fatal("resolved non-host namespace")
	}
	got, ok := registry.Lookup("host:test", valid.Name)
	if !ok || got.Signature.Distinct[0] != [2]int{0, 1} {
		t.Fatalf("lookup got %+v", got)
	}
	got.Signature.Distinct[0] = [2]int{1, 0}
	again, _ := registry.Lookup("host:test", valid.Name)
	if again.Signature.Distinct[0] != [2]int{0, 1} {
		t.Fatal("mutable signature escaped registry")
	}
}
