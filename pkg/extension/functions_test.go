package extension_test

import (
	"slices"
	"testing"

	"github.com/quvly/Qasmith/pkg/extension"
)

func TestOpaqueSnapshotsAndTrustedRegistration(t *testing.T) {
	a, err := extension.NewOpaqueType("VectorMetadata", func(v []int) ([]int, error) { return slices.Clone(v), nil })
	if err != nil {
		t.Fatal(err)
	}
	b, err := extension.NewOpaqueType("VectorMetadata", func(v []int) ([]int, error) { return slices.Clone(v), nil })
	if err != nil {
		t.Fatal(err)
	}
	r := &extension.Registry{}
	if err := r.RegisterType(a.Descriptor()); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterType(b.Descriptor()); err == nil {
		t.Fatal("duplicate type accepted")
	}
	source := []int{1, 2}
	value, err := a.Pack(source)
	if err != nil {
		t.Fatal(err)
	}
	source[0] = 9
	got, err := a.Unpack(value)
	if err != nil || !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("invalid opaque snapshot %v, %v", got, err)
	}
	got[1] = 9
	snapshot, err := a.Unpack(value)
	if err != nil || !slices.Equal(snapshot, []int{1, 2}) {
		t.Fatalf("mutated opaque value %v, %v", snapshot, err)
	}
	if _, err := b.Unpack(value); err == nil {
		t.Fatal("foreign registration accepted")
	}

	fn := extension.Function{Name: "value", Return: a.Name(),
		Evaluate: func(extension.StaticContext, []extension.Value) (extension.Value, error) { return value, nil }}
	if err := r.RegisterModule("std/test/value", fn); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterModule("std/test/value", fn); err == nil {
		t.Fatal("duplicate trusted module accepted")
	}
	copy := r.Clone()
	if copy.LookupType(a.Name()) != a.Descriptor() || !copy.HasModule("std/test/value") {
		t.Fatal("clone lost registrations")
	}
	if err := r.RegisterModule("host:untrusted/path", fn); err == nil {
		t.Fatal("invalid module path accepted")
	}
}
