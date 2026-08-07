// cloudstorage 的存储后端层：Backend 接口 + WebDAV/S3 实现工厂。

package cloudstorage

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// Backend 统一对象存储接口。
type Backend interface {
	// Put 写入对象。
	Put(ctx context.Context, key string, body io.Reader, size int64) error
	// Get 读取对象。
	Get(ctx context.Context, key string) (io.ReadCloser, int64, error)
	// Delete 删除对象。
	Delete(ctx context.Context, key string) error
	// List 列出 prefix 下的对象。
	List(ctx context.Context, prefix string) ([]Object, error)
	// Stat 获取对象元数据；不存在返回 nil,false。
	Stat(ctx context.Context, key string) (Object, bool, error)
}

// NewBackend 根据配置构造后端。
func NewBackend(cfg Config) (Backend, error) {
	switch cfg.Provider {
	case ProviderWebDAV:
		return newWebDAV(cfg), nil
	case ProviderS3:
		return newS3Backend(cfg), nil
	default:
		return nil, fmt.Errorf("cloudstorage: unsupported provider %q", cfg.Provider)
	}
}

// keyJoin 拼接远端路径（斜杠分隔）。
func keyJoin(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, "/")
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "/")
}
