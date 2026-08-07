// Package rpc 提供统一 RPC 抽象（前端 → 后端命令）。
package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
)

// Handler RPC 处理函数。
type Handler func(ctx context.Context, args json.RawMessage) (any, error)

// Router 命令路由器。
type Router struct {
	mu       sync.RWMutex
	commands map[string]Handler
}

// NewRouter 创建路由器。
func NewRouter() *Router { return &Router{commands: map[string]Handler{}} }

// Register 注册一个命令。
func (r *Router) Register(name string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[name] = h
}

// RegisterFunc 用一个 func 注册（参数自动 JSON 解码）。
func (r *Router) RegisterFunc(name string, fn any) error {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return errors.New("rpc: not a func")
	}
	r.Register(name, func(ctx context.Context, args json.RawMessage) (any, error) {
		var out []reflect.Value
		if v.Type().NumIn() == 0 {
			out = v.Call(nil)
		} else {
			// first param must be context.Context
			argv := make([]reflect.Value, v.Type().NumIn())
			argv[0] = reflect.ValueOf(ctx)
			if len(args) > 0 && v.Type().NumIn() > 1 {
				t := v.Type().In(1)
				ptr := reflect.New(t)
				if err := json.Unmarshal(args, ptr.Interface()); err != nil {
					return nil, err
				}
				argv[1] = ptr.Elem()
			}
			out = v.Call(argv)
		}
		if len(out) == 1 {
			return out[0].Interface(), nil
		}
		if len(out) == 2 {
			errVal := out[1].Interface()
			if errVal == nil {
				return out[0].Interface(), nil
			}
			return out[0].Interface(), errVal.(error)
		}
		return nil, nil
	})
	return nil
}

// Call 调用。
func (r *Router) Call(ctx context.Context, name string, args json.RawMessage) (any, error) {
	r.mu.RLock()
	h, ok := r.commands[name]
	r.mu.RUnlock()
	if !ok {
		return nil, errors.New("rpc: command not found: " + name)
	}
	return h(ctx, args)
}

// List 列出已注册命令。
func (r *Router) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.commands))
	for k := range r.commands {
		out = append(out, k)
	}
	return out
}
