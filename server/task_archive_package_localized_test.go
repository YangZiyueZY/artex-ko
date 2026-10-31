package server

import (
	"os"
	"path/filepath"
	"testing"
)

// 回归防护测试：task_archive_package.go 的归档包删除准备错误文案必须保持简体中文。
// 中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。
// 若有人把这些字面量改回韩文，本测试即失败。
//
// 调用图判定（见 errArchiveStagedAndOriginalCoexist 常量注释）：该错误由后台归档工作器
// runOneTaskArchiveJob（task_archives.go:102）经 FailTaskArchiveJob 写入 task_archives.error
// 字段，并由 tasks/page.tsx 的 TaskArchivesPanel 以 {archive.error} 直接显示给用户
// （actool 与 planner 回读均为 0）。同一传播路径上的兄弟错误（archiveTask、
// validateArchivePath）也已汉化，本次翻译与它们保持一致。
//
// stageTaskArchivePackageDelete 是不触及 DB 的纯文件函数，因此在没有 DB 的机器上，
// 可以同时造出原始文件与删除临时文件（.deleting-N），驱动真实的冲突分支。
func TestTaskArchiveStagedAndOriginalCoexistErrorLocalized(t *testing.T) {
	dataDir := t.TempDir()
	archivePath := taskArchivePath(dataDir, 21, "42")
	if err := os.MkdirAll(filepath.Dir(archivePath), archiveDirMode); err != nil {
		t.Fatal(err)
	}
	// 构造“原始归档包与删除临时文件同时存在”的冲突状态。
	if err := os.WriteFile(archivePath, []byte("original"), archiveFileMode); err != nil {
		t.Fatal(err)
	}
	staged := archivePath + ".deleting-21"
	if err := os.WriteFile(staged, []byte("archive"), archiveFileMode); err != nil {
		t.Fatal(err)
	}
	_, moved, err := stageTaskArchivePackageDelete(archivePath, 21)
	if err == nil {
		t.Fatalf("原始文件与删除临时文件共存时必须返回错误 (moved=%v)", moved)
	}
	if moved {
		t.Fatal("共存的矛盾状态下 moved 不能为 true")
	}
	assertChineseError(t, "staged_and_original_coexist", err.Error())
}
