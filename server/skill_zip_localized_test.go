package server

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

// TestSkillZipErrorsLocalized 守护 skill_zip.go 的汉化：每条用户可见的技能上传错误响应
// （fsUploadSkill 经 writeErr(400, err.Error()) 直接返回，见 server_mgmt.go）都必须是
// 简体中文，且任何 zip 压缩方式显示名都不得残留韩文。把这些字面量改回韩文/英文会导致失败。
func TestSkillZipErrorsLocalized(t *testing.T) {
	// Message constants rendered with sample arguments.
	assertChineseError(t, "errSkillZipParse",
		fmt.Errorf(errSkillZipParse, fmt.Errorf("boom")).Error())
	assertChineseError(t, "errSkillZipEncrypted",
		fmt.Sprintf(errSkillZipEncrypted, "demo/SKILL.md"))
	assertChineseError(t, "errSkillZipUnsupported",
		fmt.Sprintf(errSkillZipUnsupported, "LZMA", 14, "demo/SKILL.md"))

	// 非 zip 上传必须给用户中文提示，而不是 stdlib 的原始错误。
	if _, err := newSkillZipReader([]byte("this is not a zip archive")); err == nil {
		t.Fatal("非 zip 输入必须返回错误")
	} else {
		assertChineseError(t, "newSkillZipReader", err.Error())
		if !strings.Contains(err.Error(), "压缩文件") {
			t.Fatalf("parse error = %q, want '压缩文件' 提示", err.Error())
		}
	}

	// 任何 zip 压缩方式显示名都不得残留韩文；已汉化的两个（AES、未知回落）必须是中文。
	for m, name := range zipMethodNames {
		for _, r := range name {
			if unicode.Is(unicode.Hangul, r) {
				t.Fatalf("zipMethodNames[%d] = %q 中仍残留韩文", m, name)
			}
		}
	}
	assertChineseError(t, "zipMethodName(AES)", zipMethodName(zipMethodAES))
	assertChineseError(t, "zipMethodName(unknown)", zipMethodName(0xffff))
}
