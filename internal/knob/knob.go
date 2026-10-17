// Package knob reads only the volume/mute controls of a dedicated media device.
package knob

type Event struct {
	Step   int
	Toggle bool
}

// Releases and mute autorepeat must not change the selected color/target.
func decode(kind, code uint16, value int32) (Event, bool) {
	if kind != 1 || (value != 1 && value != 2) {
		return Event{}, false
	}
	switch code {
	case 115: // KEY_VOLUMEUP
		return Event{Step: 1}, true
	case 114: // KEY_VOLUMEDOWN
		return Event{Step: -1}, true
	case 113: // KEY_MUTE
		return Event{Toggle: true}, value == 1
	}
	return Event{}, false
}
