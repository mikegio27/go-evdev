package evdev

import "testing"

func TestRemapOptions(t *testing.T) {
	o := remapOptions{name: "go-evdev remapper"}
	for _, opt := range []RemapOption{
		WithName("swapper"),
		WithExtraKeys(KEY_C, KEY_LEFTCTRL),
		WithExtraCapabilities(Capabilities{Rels: []EvCode{REL_X}}),
	} {
		opt(&o)
	}

	if o.name != "swapper" {
		t.Errorf("name = %q, want swapper", o.name)
	}
	if len(o.extra.Keys) != 2 || o.extra.Keys[0] != KEY_C || o.extra.Keys[1] != KEY_LEFTCTRL {
		t.Errorf("extra keys = %v, want [KEY_C KEY_LEFTCTRL]", o.extra.Keys)
	}
	if len(o.extra.Rels) != 1 || o.extra.Rels[0] != REL_X {
		t.Errorf("extra rels = %v, want [REL_X]", o.extra.Rels)
	}
}

func TestMergeCaps(t *testing.T) {
	a := Capabilities{Keys: []EvCode{KEY_A}, Props: []InputProp{INPUT_PROP_POINTER}}
	b := Capabilities{Keys: []EvCode{KEY_B}, Rels: []EvCode{REL_X}}
	m := mergeCaps(a, b)

	if len(m.Keys) != 2 || m.Keys[0] != KEY_A || m.Keys[1] != KEY_B {
		t.Errorf("merged keys = %v, want [KEY_A KEY_B]", m.Keys)
	}
	if len(m.Rels) != 1 || m.Rels[0] != REL_X {
		t.Errorf("merged rels = %v, want [REL_X]", m.Rels)
	}
	if len(m.Props) != 1 || m.Props[0] != INPUT_PROP_POINTER {
		t.Errorf("merged props = %v, want [INPUT_PROP_POINTER]", m.Props)
	}
}
