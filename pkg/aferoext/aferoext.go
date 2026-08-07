// Package aferoext 是 afero 的最小封装（为后续文件操作统一接口占位）。
package aferoext

import (
	"github.com/spf13/afero"
)

// FS 暴露 afero.Fs。
type FS = afero.Fs

// NewMem 返回内存文件系统（用于测试）。
func NewMem() afero.Fs { return afero.NewMemMapFs() }

// NewOS 返回操作系统文件系统。
func NewOS() afero.Fs { return afero.NewOsFs() }
