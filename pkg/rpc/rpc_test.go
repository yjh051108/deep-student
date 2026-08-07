package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestRegisterFuncAndCall(t *testing.T) {
	r := NewRouter()
	type args struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	if err := r.RegisterFunc("sum", func(ctx context.Context, in args) (int, error) {
		return in.A + in.B, nil
	}); err != nil {
		t.Fatal(err)
	}
	out, err := r.Call(context.Background(), "sum", json.RawMessage(`{"a":2,"b":3}`))
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := out.(int); !ok || v != 5 {
		t.Fatalf("out=%v", out)
	}
}

func TestCommandNotFound(t *testing.T) {
	r := NewRouter()
	_, err := r.Call(context.Background(), "x", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, err) && err.Error() == "" {
		t.Fatal("bad error")
	}
}
