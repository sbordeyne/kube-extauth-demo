package main

import (
	"reflect"
	"testing"
)

func TestParseClaim(t *testing.T) {
	tests := []struct {
		name    string
		rawKey  string
		values  []string
		wantKey string
		wantVal any
		wantErr bool
	}{
		{"infer string", "sub", []string{"alice"}, "sub", "alice", false},
		{"infer int", "uid", []string{"42"}, "uid", int64(42), false},
		{"infer float", "ratio", []string{"1.5"}, "ratio", 1.5, false},
		{"infer bool true", "admin", []string{"true"}, "admin", true, false},
		{"infer bool false", "admin", []string{"false"}, "admin", false, false},
		{"one not treated as bool", "n", []string{"1"}, "n", int64(1), false},
		{"repeated key array", "scopes", []string{"read", "write"}, "scopes", []any{"read", "write"}, false},
		{"array mixed inference", "vals", []string{"1", "true", "x"}, "vals", []any{int64(1), true, "x"}, false},
		{"hint string keeps digits", "zip:string", []string{"01000"}, "zip", "01000", false},
		{"hint int", "uid:int", []string{"7"}, "uid", int64(7), false},
		{"hint float", "r:float", []string{"2"}, "r", float64(2), false},
		{"hint bool accepts 1", "b:bool", []string{"1"}, "b", true, false},
		{"hint json object", "meta:json", []string{`{"tier":1}`}, "meta", map[string]any{"tier": float64(1)}, false},
		{"hint json array", "list:json", []string{`[1,2]`}, "list", []any{float64(1), float64(2)}, false},
		{"hint applies to array", "ids:int", []string{"1", "2"}, "ids", []any{int64(1), int64(2)}, false},
		{"unknown suffix is part of key", "urn:custom", []string{"v"}, "urn:custom", "v", false},
		{"bad int hint errors", "uid:int", []string{"abc"}, "", nil, true},
		{"bad json hint errors", "meta:json", []string{"{"}, "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, val, err := parseClaim(tt.rawKey, tt.values)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got val=%v", val)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
			if !reflect.DeepEqual(val, tt.wantVal) {
				t.Errorf("val = %#v, want %#v", val, tt.wantVal)
			}
		})
	}
}
