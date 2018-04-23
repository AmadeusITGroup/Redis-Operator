package redis

import (
	"reflect"
	"sort"
	"testing"
)

func TestSlotRangeDecode(t *testing.T) {
	testTable := []struct {
		asString string
		slots    []Slot
		err      bool
	}{
		{"", nil, true},
		{"1-9000", BuildSlotSlice(1, 9000), false},
		{"1-1", []Slot{1}, false},
		{"-1-10", nil, true},
		{"foo", nil, true},
	}
	for _, tt := range testTable {
		result, _, _, err := DecodeSlotRange(tt.asString)
		if tt.err && (err == nil) {
			t.Errorf("expected error got no error")
			continue
		}
		if !tt.err && (err != nil) {
			t.Errorf("expected no error got error: %s", err)
			continue
		}
		if !reflect.DeepEqual(result, tt.slots) {
			if !(len(tt.slots) == 0 && len(result) == 0) {
				t.Errorf("expected result to be '%s', got '%s'", tt.slots, result)
			}
		}
	}
}

func TestMigratingSlotDecode(t *testing.T) {
	testTable := []struct {
		asString string
		migSlot  *MigratingSlot
		err      bool
	}{
		{"", nil, true},
		{"1-9000", nil, false},
		{"[fail->-anodeid]", nil, true},
		{"[42-<-anodeid]", nil, false},
		{"[42->-anodeid]", &MigratingSlot{SlotID: 42, ToNodeID: "anodeid"}, false},
	}
	for _, tt := range testTable {
		_, _, mig, err := DecodeSlotRange(tt.asString)
		if tt.err && (err == nil) {
			t.Errorf("expected error got no error")
		}
		if !tt.err && (err != nil) {
			t.Errorf("expected no error got error: %s", err)
		}
		if !reflect.DeepEqual(tt.migSlot, mig) {
			t.Errorf("expected '%s', got '%s'", tt.migSlot, mig)
		}
	}
}

func TestImporatingSlotDecode(t *testing.T) {
	testTable := []struct {
		asString   string
		importSlot *ImportingSlot
		err        bool
	}{
		{"", nil, true},
		{"1-9000", nil, false},
		{"[fail-<-anodeid]", nil, true},
		{"[42->-anodeid]", nil, false},
		{"[42-<-anodeid]", &ImportingSlot{SlotID: 42, FromNodeID: "anodeid"}, false},
	}
	for _, tt := range testTable {
		_, imp, _, err := DecodeSlotRange(tt.asString)
		if tt.err && (err == nil) {
			t.Errorf("expected error got no error")
		}
		if !tt.err && (err != nil) {
			t.Errorf("expected no error got error: %s", err)
		}
		if !reflect.DeepEqual(tt.importSlot, imp) {
			t.Errorf("expected '%s', got '%s'", tt.importSlot, imp)
		}
	}
}

func TestSlotContains(t *testing.T) {
	slice := []Slot{1, 2, 3}
	if !Contains(slice, 1) {
		t.Error("1 should be in {1, 2, 3}")
	}
	if Contains(slice, 4) {
		t.Error("4 is not in {1, 2, 3}")
	}
}

func TestSlotRangesFromSlots(t *testing.T) {
	testTable := []struct {
		sSlice  []Slot
		sRanges []SlotRange
	}{
		{[]Slot{8, 3, 10, 5, 6, 7, 2, 9, 4}, []SlotRange{{Min: 2, Max: 10}}},
		{[]Slot{2, 2, 3, 4, 5, 6, 7, 8, 9, 10, 4}, []SlotRange{{Min: 2, Max: 10}}}, //overlap
		{[]Slot{0}, []SlotRange{{Min: 0, Max: 0}}},                                 // one
		{nil, []SlotRange{}},                                                       // nil
		{[]Slot{0, 1, 2, 5, 6, 7, 345}, []SlotRange{{Min: 0, Max: 2}, {Min: 5, Max: 7}, {Min: 345, Max: 345}}}, // several ranges
	}

	for i, tt := range testTable {
		ranges := SlotRangesFromSlots(tt.sSlice)
		if !reflect.DeepEqual(ranges, tt.sRanges) {
			t.Errorf("[case %d]expected result to be '%s', got '%s'", i, tt.sRanges, ranges)
		}
	}
}

func TestRemoveSlots(t *testing.T) {
	testTable := []struct {
		sSlice1  []Slot
		sSlice2  []Slot
		expected []Slot
	}{
		{[]Slot{2, 3, 4, 5, 6, 7, 8, 9, 10}, []Slot{2, 10}, []Slot{3, 4, 5, 6, 7, 8, 9}},
		{[]Slot{2, 5}, []Slot{2, 2, 3}, []Slot{5}},
		{[]Slot{0, 1, 3, 4}, []Slot{0, 1, 3, 4}, []Slot{}},
		{[]Slot{}, []Slot{2, 10}, []Slot{}},
		{[]Slot{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, []Slot{5}, []Slot{0, 1, 2, 3, 4, 6, 7, 8, 9, 10}},
	}

	for i, tt := range testTable {
		newRange := RemoveSlots(tt.sSlice1, tt.sSlice2)
		if !reflect.DeepEqual(newRange, tt.expected) {
			t.Errorf("[case %d]expected result to be '%s', got '%s'", i, tt.expected, newRange)
		}
	}
}

func TestAddSlots(t *testing.T) {
	testTable := []struct {
		sSlice1  []Slot
		sSlice2  []Slot
		expected []Slot
	}{
		{[]Slot{2, 3, 4, 5, 6, 7, 8, 9, 10}, []Slot{1, 11, 13}, []Slot{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 13}},
		{[]Slot{2, 5}, []Slot{2, 2, 3}, []Slot{2, 3, 5}},
		{[]Slot{}, []Slot{0, 1, 2, 3, 4}, []Slot{0, 1, 2, 3, 4}},
		{[]Slot{}, []Slot{2, 10}, []Slot{2, 10}},
	}

	for i, tt := range testTable {
		newSlots := AddSlots(tt.sSlice1, tt.sSlice2)
		sort.Sort(SlotSlice(newSlots))
		if !reflect.DeepEqual(newSlots, tt.expected) {
			t.Errorf("[case %d]expected result to be '%s', got '%s'", i, tt.expected, newSlots)
		}
	}
}
