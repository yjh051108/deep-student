// WebDAV 后端：用标准 HTTP 方法（PUT/GET/DELETE/PROPFIND/MKCOL）实现 Backend。

package cloudstorage

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// webDAV WebDAV 后端。
type webDAV struct {
	baseURL string
	user    string
	pass    string
	client  *http.Client
}

func newWebDAV(cfg Config) *webDAV {
	return &webDAV{
		baseURL: strings.TrimSuffix(cfg.WebDAVURL, "/"),
		user:    cfg.WebDAVUser,
		pass:    cfg.WebDAVPass,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// urlFor 把 key 拼成完整 URL（支持带路径的 base，如 /remote.php/dav/files/u）。
func (d *webDAV) urlFor(key string) string {
	return d.baseURL + "/" + strings.TrimPrefix(key, "/")
}

func (d *webDAV) setAuth(req *http.Request) {
	if d.user != "" {
		req.SetBasicAuth(d.user, d.pass)
	}
}

// Put 写入对象（自动创建父目录）。
func (d *webDAV) Put(ctx context.Context, key string, body io.Reader, size int64) error {
	if err := d.ensureDir(ctx, key); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, d.urlFor(key), body)
	if err != nil {
		return err
	}
	d.setAuth(req)
	req.Header.Set("Content-Type", "application/octet-stream")
	if size >= 0 {
		req.ContentLength = size
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("webdav put: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webdav put: %s", resp.Status)
	}
	return nil
}

// Get 读取对象。
func (d *webDAV) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.urlFor(key), nil)
	if err != nil {
		return nil, 0, err
	}
	d.setAuth(req)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("webdav get: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, 0, errNotFound(key)
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("webdav get: %s", resp.Status)
	}
	return resp.Body, resp.ContentLength, nil
}

// Delete 删除对象。
func (d *webDAV) Delete(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, d.urlFor(key), nil)
	if err != nil {
		return err
	}
	d.setAuth(req)
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("webdav delete: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("webdav delete: %s", resp.Status)
	}
	return nil
}

// List 用 PROPFIND 列出 prefix 下的对象。
func (d *webDAV) List(ctx context.Context, prefix string) ([]Object, error) {
	u := d.urlFor(prefix)
	if prefix == "" {
		u = d.baseURL
	}
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", u, nil)
	if err != nil {
		return nil, err
	}
	d.setAuth(req)
	req.Header.Set("Depth", "1")
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webdav list: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return nil, fmt.Errorf("webdav list: %s", resp.Status)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	var ms multiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, fmt.Errorf("webdav list parse: %w", err)
	}
	var out []Object
	for _, respEl := range ms.Responses {
		if !strings.Contains(respEl.Href, "/"+strings.TrimPrefix(prefix, "/")) {
			continue
		}
		var obj Object
		obj.Key = respEl.Href
		for _, prop := range respEl.Propstat {
			if prop.Prop.GetContentLength != nil {
				obj.Size = *prop.Prop.GetContentLength
			}
			if prop.Prop.GetLastModified != nil {
				if t, err := http.ParseTime(*prop.Prop.GetLastModified); err == nil {
					obj.Mod = t
				}
			}
		}
		// 跳过目录自身（以 / 结尾）
		if strings.HasSuffix(obj.Key, "/") {
			continue
		}
		out = append(out, obj)
	}
	return out, nil
}

// Stat 返回对象元数据。
func (d *webDAV) Stat(ctx context.Context, key string) (Object, bool, error) {
	objs, err := d.List(ctx, key)
	if err != nil {
		return Object{}, false, err
	}
	for _, o := range objs {
		if strings.HasSuffix(o.Key, "/"+strings.TrimPrefix(key, "/")) || o.Key == key {
			return o, true, nil
		}
	}
	return Object{}, false, nil
}

// ensureDir 递归创建父目录（MKCOL）。
func (d *webDAV) ensureDir(ctx context.Context, key string) error {
	parts := strings.Split(strings.Trim(key, "/"), "/")
	if len(parts) <= 1 {
		return nil
	}
	cur := ""
	for i := 0; i < len(parts)-1; i++ {
		cur = keyJoin(cur, parts[i])
		req, err := http.NewRequestWithContext(ctx, "MKCOL", d.urlFor(cur), nil)
		if err != nil {
			return err
		}
		d.setAuth(req)
		resp, err := d.client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		// 201 创建 / 405 已存在均接受
		if resp.StatusCode != 201 && resp.StatusCode != 405 && resp.StatusCode != 200 {
			return fmt.Errorf("webdav mkcol %s: %s", cur, resp.Status)
		}
	}
	return nil
}

// errNotFound 对象不存在错误。
func errNotFound(key string) error { return &NotFoundError{Key: key} }

// NotFoundError 对象不存在。
type NotFoundError struct{ Key string }

func (e *NotFoundError) Error() string { return "cloudstorage: not found: " + e.Key }

// IsNotFound 判断是否为不存在错误。
func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// ---- XML 结构（PROPFIND 响应） ----

type multiStatus struct {
	Responses []propfindResponse `xml:"response"`
}

type propfindResponse struct {
	Href     string       `xml:"href"`
	Propstat []propstatEl `xml:"propstat"`
}

type propstatEl struct {
	Prop multistatusProps `xml:"prop"`
}

type multistatusProps struct {
	GetContentLength *int64  `xml:"getcontentlength"`
	GetLastModified  *string `xml:"getlastmodified"`
}

