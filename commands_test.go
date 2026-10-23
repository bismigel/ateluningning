package main

import (
	"testing"
	"time"
)

func TestRegistryLookupNameAndAlias(t *testing.T) {
	r := NewRegistry()
	r.Register(
		&Command{Name: "hello"},
		&Command{Name: "askluningning", Aliases: []string{"ask"}},
	)

	if c, ok := r.Lookup("hello"); !ok || c.Name != "hello" {
		t.Fatalf("Lookup(hello) = %v, %v", c, ok)
	}
	if c, ok := r.Lookup("ask"); !ok || c.Name != "askluningning" {
		t.Fatalf("Lookup(ask) = %v, %v", c, ok)
	}
	if _, ok := r.Lookup("nope"); ok {
		t.Fatalf("Lookup(nope) should not exist")
	}
}

func TestRegistryPreservesOrder(t *testing.T) {
	r := NewRegistry()
	r.Register(&Command{Name: "a"}, &Command{Name: "b"}, &Command{Name: "c"})
	all := r.All()
	if len(all) != 3 || all[0].Name != "a" || all[2].Name != "c" {
		t.Fatalf("unexpected order: %+v", all)
	}
}

func TestLockableList(t *testing.T) {
	r := NewRegistry()
	r.Register(
		&Command{Name: "hello"},
		&Command{Name: "horoscope", Lockable: true},
		&Command{Name: "askluningning", Lockable: true},
	)
	got := r.LockableList()
	want := "`!horoscope`, or `!askluningning`"
	if got != want {
		t.Fatalf("LockableList() = %q, want %q", got, want)
	}
}

func TestHumanJoin(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a, or b"},
		{[]string{"a", "b", "c"}, "a, b, or c"},
	}
	for _, tc := range cases {
		if got := humanJoin(tc.in); got != tc.want {
			t.Fatalf("humanJoin(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	if got := formatDuration(0); got != "0 seconds" {
		t.Fatalf("formatDuration(0) = %q", got)
	}
	d := time.Duration(24*3600+2*3600+3*60+4) * time.Second
	if got := formatDuration(d); got != "1 days, 2 hours, 3 minutes, 4 seconds" {
		t.Fatalf("formatDuration = %q", got)
	}
}

func TestCapitalizeFirstLetter(t *testing.T) {
	if got := CapitalizeFirstLetter("leo"); got != "Leo" {
		t.Fatalf("CapitalizeFirstLetter(leo) = %q", got)
	}
	if got := CapitalizeFirstLetter(""); got != "" {
		t.Fatalf("CapitalizeFirstLetter(empty) = %q", got)
	}
}
