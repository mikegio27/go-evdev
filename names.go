package evdev

import "fmt"

// String returns the EV_* name for the type (e.g. "EV_KEY"), or a numeric
// fallback like "EV_?(0x1f)" for unknown types.
func (t EvType) String() string {
	if name, ok := evTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("EV_?(0x%x)", uint16(t))
}

// String returns the name for the bus type (e.g. "BUS_USB").
func (b BusType) String() string {
	if name, ok := busNames[b]; ok {
		return name
	}
	return fmt.Sprintf("BUS_?(0x%x)", uint16(b))
}

// String returns the name for the device property (e.g. "INPUT_PROP_POINTER"),
// or a numeric fallback like "INPUT_PROP_?(0x10)" for unknown properties.
func (p InputProp) String() string {
	if name, ok := propNames[p]; ok {
		return name
	}
	return fmt.Sprintf("INPUT_PROP_?(0x%x)", uint16(p))
}

// CodeName returns the constant name for a code within the namespace of the
// given event type (e.g. CodeName(EV_KEY, KEY_A) == "KEY_A"). Unknown codes
// yield a numeric fallback like "KEY_?(0x1ff)".
func CodeName(t EvType, c EvCode) string {
	if codes, ok := evCodeNames[t]; ok {
		if name, ok := codes[c]; ok {
			return name
		}
	}
	return fmt.Sprintf("%s_?(0x%x)", codePrefixForType(t), uint16(c))
}

// EvTypeByName resolves an EV_* name to its EvType (e.g. "EV_KEY" -> EV_KEY).
func EvTypeByName(name string) (EvType, bool) {
	t, ok := evTypeByName[name]
	return t, ok
}

// EvCodeByName resolves a code or alias name to its EvCode (e.g. "KEY_A",
// "BTN_LEFT"). The name space is shared across types; callers that need the
// type should use the corresponding EV_* type.
func EvCodeByName(name string) (EvCode, bool) {
	c, ok := evCodeByName[name]
	return c, ok
}

// codePrefixForType returns the conventional code-name prefix for an event
// type, used to build fallbacks for unknown codes.
func codePrefixForType(t EvType) string {
	switch t {
	case EV_SYN:
		return "SYN"
	case EV_KEY:
		return "KEY"
	case EV_REL:
		return "REL"
	case EV_ABS:
		return "ABS"
	case EV_MSC:
		return "MSC"
	case EV_SW:
		return "SW"
	case EV_LED:
		return "LED"
	case EV_SND:
		return "SND"
	case EV_REP:
		return "REP"
	default:
		return "CODE"
	}
}
