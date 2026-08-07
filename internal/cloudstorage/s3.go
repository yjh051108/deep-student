// S3 后端：AWS Signature V4 最小实现（纯标准库，无外部 SDK）。
// 支持 PUT/GET/DELETE/ListObjectsV2；兼容 R2 / OSS / MinIO 等 S3 兼容端点。

package cloudstorage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// s3Backend S3 后端。
type s3Backend struct {
	endpoint string // 如 https://s3.amazonaws.com 或 http://127.0.0.1:9000
	region   string
	bucket   string
	access   string
	secret   string
	client   *http.Client
}

func newS3Backend(cfg Config) *s3Backend {
	return &s3Backend{
		endpoint: strings.TrimSuffix(cfg.S3Endpoint, "/"),
		region:   cfg.S3Region,
		bucket:   cfg.S3Bucket,
		access:   cfg.S3AccessKey,
		secret:   cfg.S3SecretKey,
		client:   &http.Client{Timeout: 120 * time.Second},
	}
}

func (b *s3Backend) urlFor(key string) string {
	return b.endpoint + "/" + b.bucket + "/" + strings.TrimPrefix(key, "/")
}

func (b *s3Backend) regionOrDefault() string {
	if b.region == "" {
		return "us-east-1"
	}
	return b.region
}

// sign 对请求做 SigV4 签名。
func (b *s3Backend) sign(req *http.Request, payloadHash string, t time.Time) {
	amzDate := t.UTC().Format("20060102T150405Z")
	dateStamp := t.UTC().Format("20060102")
	region := b.regionOrDefault()

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	// canonical request
	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalQuery := canonicalQueryString(req.URL.Query())

	// 收集 headers：host + x-amz-* 排序
	headerNames := []string{"host"}
	headerVals := map[string]string{"host": req.URL.Host}
	for k, v := range req.Header {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "x-amz-") {
			headerNames = append(headerNames, lk)
			headerVals[lk] = strings.Join(v, ",")
		}
	}
	sort.Strings(headerNames)
	var hdrParts []string
	for _, n := range headerNames {
		hdrParts = append(hdrParts, n+":"+strings.TrimSpace(headerVals[n]))
	}
	canonicalHeaders := strings.Join(hdrParts, "\n") + "\n"
	signedHeaders := strings.Join(headerNames, ";")

	canonicalRequest := strings.Join([]string{
		req.Method, canonicalURI, canonicalQuery, canonicalHeaders, signedHeaders, payloadHash,
	}, "\n")

	credentialScope := dateStamp + "/" + region + "/s3/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256", amzDate, credentialScope,
		hexSHA256([]byte(canonicalRequest)),
	}, "\n")

	kDate := hmacSHA256([]byte("AWS4"+b.secret), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, "s3")
	kSigning := hmacSHA256(kService, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))

	auth := "AWS4-HMAC-SHA256 Credential=" + b.access + "/" + credentialScope +
		", SignedHeaders=" + signedHeaders + ", Signature=" + signature
	req.Header.Set("Authorization", auth)
}

func canonicalQueryString(q url.Values) string {
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		vals := q[k]
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

func hexSHA256(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key []byte, data string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(data))
	return m.Sum(nil)
}

// Put 写入对象。
func (b *s3Backend) Put(ctx context.Context, key string, body io.Reader, size int64) error {
	// 先读入内存计算 hash（备份文件通常不大；大文件场景后续可加流式分片）
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, b.urlFor(key), strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if size >= 0 {
		req.ContentLength = size
	}
	b.sign(req, hexSHA256(data), time.Now())
	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("s3 put: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body2, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 put: %s: %s", resp.Status, truncateS3(string(body2), 200))
	}
	return nil
}

// Get 读取对象。
func (b *s3Backend) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.urlFor(key), nil)
	if err != nil {
		return nil, 0, err
	}
	b.sign(req, hexSHA256(nil), time.Now())
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("s3 get: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, 0, errNotFound(key)
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("s3 get: %s", resp.Status)
	}
	return resp.Body, resp.ContentLength, nil
}

// Delete 删除对象。
func (b *s3Backend) Delete(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, b.urlFor(key), nil)
	if err != nil {
		return err
	}
	b.sign(req, hexSHA256(nil), time.Now())
	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("s3 delete: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("s3 delete: %s", resp.Status)
	}
	return nil
}

// List 列出 prefix 下的对象（ListObjectsV2）。
func (b *s3Backend) List(ctx context.Context, prefix string) ([]Object, error) {
	u := b.endpoint + "/" + b.bucket
	q := url.Values{}
	if prefix != "" {
		q.Set("list-type", "2")
		q.Set("prefix", strings.TrimPrefix(prefix, "/"))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	b.sign(req, hexSHA256(nil), time.Now())
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 list: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("s3 list: %s", resp.Status)
	}
	var result listBucketResult
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("s3 list parse: %w", err)
	}
	var out []Object
	for _, c := range result.Contents {
		var mod time.Time
		if c.LastModified != "" {
			if t, err := time.Parse(time.RFC3339, c.LastModified); err == nil {
				mod = t
			}
		}
		out = append(out, Object{
			Key:  c.Key,
			Size: c.Size,
			ETag: c.ETag,
			Mod:  mod,
		})
	}
	return out, nil
}

// Stat 返回对象元数据（HEAD）。
func (b *s3Backend) Stat(ctx context.Context, key string) (Object, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, b.urlFor(key), nil)
	if err != nil {
		return Object{}, false, err
	}
	b.sign(req, hexSHA256(nil), time.Now())
	resp, err := b.client.Do(req)
	if err != nil {
		return Object{}, false, fmt.Errorf("s3 head: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Object{}, false, nil
	}
	if resp.StatusCode >= 300 {
		return Object{}, false, fmt.Errorf("s3 head: %s", resp.Status)
	}
	obj := Object{Key: key}
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		fmt.Sscanf(cl, "%d", &obj.Size)
	}
	obj.ETag = resp.Header.Get("ETag")
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		if t, err := http.ParseTime(lm); err == nil {
			obj.Mod = t
		}
	}
	return obj, true, nil
}

// listBucketResult ListObjectsV2 响应。
type listBucketResult struct {
	Contents []struct {
		Key          string `xml:"Key"`
		Size         int64  `xml:"Size"`
		ETag         string `xml:"ETag"`
		LastModified string `xml:"LastModified"`
	} `xml:"Contents"`
}

func truncateS3(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
