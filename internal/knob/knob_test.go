package knob

import "testing"

func TestMediaEventFiltering(t *testing.T) {
	for _, tc := range []struct {
		kind, code uint16
		value      int32
		want       Event
		ok         bool
	}{
		{1, 115, 1, Event{Step: 1}, true},
		{1, 114, 1, Event{Step: -1}, true},
		{1, 115, 2, Event{Step: 1}, true},
		{1, 113, 1, Event{Toggle: true}, true},
		{1, 113, 2, Event{}, false},
		{1, 115, 0, Event{}, false},
		{1, 30, 1, Event{}, false}, // ordinary typing is ignored
		{2, 115, 1, Event{}, false},
	} {
		got, ok := decode(tc.kind, tc.code, tc.value)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("%+v: got %+v, %v", tc, got, ok)
		}
	}
}
