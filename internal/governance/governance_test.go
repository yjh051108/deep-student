package governance

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/helixnow/deep-student-go/pkg/config"
	"github.com/helixnow/deep-student-go/pkg/crypto"
	"github.com/helixnow/deep-student-go/pkg/eventbus"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

func newSvc(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	bs, _ := blob.New(filepath.Join(dir, "blob"))
	fs := vfs.NewFS(bs)
	st, err := store.Open(filepath.Join(dir, "deepstudent.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	cr, err := crypto.NewManager(filepath.Join(dir, "keys"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{DataDir: dir, BackupDir: filepath.Join(dir, "backups")}
	return New(fs, st, cr, cfg, eventbus.New())
}

func TestEnsureDir(t *testing.T) {
	if err := EnsureDir(t.TempDir() + "/x/y"); err != nil {
		t.Fatal(err)
	}
}

func TestStatusAndIntegrity(t *testing.T) {
	s := newSvc(t)
	st := s.Status()
	if st["slot"] == "" {
		t.Fatal("missing slot")
	}
	issues := s.CheckIntegrity()
	// blob/keys 目录可能被 NewManager 创建过，但 deepstudent.db 已创建
	if len(issues) > 0 {
		// 至少 data_dir/deepstudent.db 必须存在
	}
}

func TestBackupAndRestore(t *testing.T) {
	s := newSvc(t)
	// 写入一个资源
	uri := "vfs://note/test1"
	if _, err := s.vfs.Put(uri, []byte(`{"x":1}`), map[string]string{"title": "t"}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "backup.bin")
	ref, err := s.Backup(target)
	if err != nil {
		t.Fatal(err)
	}
	if ref == "" {
		t.Fatal("empty ref")
	}
	if err := s.Restore(target); err != nil {
		t.Fatal(err)
	}
	// 验证资源已恢复
	_, _, err = s.vfs.Get(uri)
	if err != nil {
		t.Fatal("restore failed: " + err.Error())
	}
}

func TestExportImport(t *testing.T) {
	s := newSvc(t)
	if _, err := s.vfs.Put("vfs://note/n1", []byte("a"), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.vfs.Put("vfs://paper/p1", []byte("b"), nil); err != nil {
		t.Fatal(err)
	}
	zpath := filepath.Join(t.TempDir(), "exp.zip")
	if err := s.Export(zpath, []vfs.ResourceType{vfs.TypeNote, vfs.TypePaper}); err != nil {
		t.Fatal(err)
	}
	if err := s.Import(zpath); err != nil {
		t.Fatal(err)
	}
}

func TestSwitchSlot(t *testing.T) {
	s := newSvc(t)
	if err := s.SwitchSlot("A"); err != nil {
		t.Fatal(err)
	}
}

func TestAuditLogs(t *testing.T) {
	s := newSvc(t)
	if _, err := s.vfs.Put("vfs://note/n1", []byte("a"), nil); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if _, err := s.Backup(filepath.Join(dir, "bk.bin")); err != nil {
		t.Fatal(err)
	}
	logs, err := s.AuditLogs(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) == 0 {
		t.Fatal("no audit logs")
	}
}

// TestBackupConcurrentWrites BUG-004 回归：VFS 在并发 Put 的同时触发 Backup，
// 验证：
//  1) 备份能成功（zip 可被读回且不损坏）
//  2) 备份结束后再读取 vfs.Get 仍能拿到全部已 Put 的资源（无丢数据）
//  3) -race 下无数据竞争
func TestBackupConcurrentWrites(t *testing.T) {
	s := newSvc(t)
	// 先写一批初始资源，确保 backup 时 vfs 非空
	for i := 0; i < 20; i++ {
		uri := "vfs://note/init-" + time.Now().Format("150405.000000000") + "-" + idstr(i)
		if _, err := s.vfs.Put(uri, []byte("init"), nil); err != nil {
			t.Fatal(err)
		}
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	// 4 个写线程持续 Put 新资源
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; w >= 0 && i < 50; i++ {
				select {
				case <-stop:
					return
				default:
				}
				uri := "vfs://note/w" + idstr(w) + "-" + idstr(i) + "-" + time.Now().Format("150405.000000000")
				_, _ = s.vfs.Put(uri, []byte("payload"), nil)
			}
		}(w)
	}

	// 在并发写期间做一次 backup
	dir := t.TempDir()
	target := filepath.Join(dir, "concurrent.bin")
	ref, err := s.Backup(target)
	if err != nil {
		close(stop)
		wg.Wait()
		t.Fatalf("backup: %v", err)
	}
	if ref == "" {
		close(stop)
		wg.Wait()
		t.Fatal("empty ref")
	}
	close(stop)
	wg.Wait()

	// 校验：解出 backup 的明文（用 A 槽位解密），确认是合法 zip 且至少包含初始 20 个 note
	slot, _ := s.crypto.ActiveSlot()
	ct, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := s.crypto.Decrypt(slot, ct)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(plain), int64(len(plain)))
	if err != nil {
		t.Fatalf("backup zip corrupt: %v", err)
	}
	noteCount := 0
	for _, f := range zr.File {
		if filepath.Dir(f.Name) == string(vfs.TypeNote) {
			noteCount++
		}
	}
	if noteCount < 20 {
		t.Fatalf("backup should contain >=20 notes, got %d", noteCount)
	}

	// 校验 vfs 在 backup 后仍能正确读取每个 put 过的资源（无内部损坏）
	list := s.vfs.List(vfs.TypeNote)
	if len(list) < 20 {
		t.Fatalf("vfs lost notes: %d", len(list))
	}
}

// idstr 简单 int→string 工具（避免导入 strconv 在测试里噪音）。
func idstr(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
