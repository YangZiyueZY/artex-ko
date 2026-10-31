// Package config loads runtime configuration from a JSON file, with environment
// variables taking precedence. Currently it carries the PostgreSQL connection.
package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Database is the PostgreSQL connection config. Either set DSN directly, or set
// the component fields and a DSN is assembled from them.
type Database struct {
	DSN      string `json:"dsn"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

// Config 是磁盘配置文件的结构。
type Config struct {
	Database Database `json:"database"`
	SkillDir string   `json:"skill_dir"`
	// Language 是启动时的默认用户可见输出语言（默认 "zh"，可选 "en"/"ko"/"es"/"ja"）。
	// 为空则用 DefaultLanguage。运行期在设置表（UI 设置页）里选定的值优先生效。
	Language string `json:"language"`
}

// SupportedLanguages 列出本项目可切换的用户可见输出语言。智能体输出、界面文案与后端消息都以它为键。
var SupportedLanguages = []string{"en", "ko", "zh", "es", "ja"}

// DefaultLanguage 是没有任何配置时的出厂用户可见语言。本项目默认简体中文，运行时可
// 通过语言设置（以及 ARTEX_LANG）切换为韩语、英语、西班牙语或日语。
const DefaultLanguage = "zh"

// NormalizeLanguage lower-cases and validates a language code, returning "" when
// the code is empty or unsupported (so callers can fall through to the next source).
func NormalizeLanguage(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	for _, c := range SupportedLanguages {
		if c == code {
			return c
		}
	}
	return ""
}

// CurrentLanguage 由服务端接线后返回运行期（设置表）选定的用户可见输出语言，并叠加启动默认值。
// 后端展示字符串（枚举标签等）统一经由 ActiveLanguage 读取，它是唯一来源。
// 未接线（单元测试/独立运行）时 ActiveLanguage 回落到 "zh"。
var CurrentLanguage func() string

// ActiveLanguage 解析后端展示字符串所用语言。优先级：运行期钩子（服务端接线到设置表）
// 高于未接线时的 "zh" 兜底。生产环境总会接线 CurrentLanguage，因此新部署经该钩子解析到
// 配置默认值（DefaultLanguage = "zh"）；"zh" 兜底只在钩子缺席（单元测试/独立运行）时生效。
func ActiveLanguage() string {
	if CurrentLanguage != nil {
		if v := NormalizeLanguage(CurrentLanguage()); v != "" {
			return v
		}
	}
	return "zh"
}

// Language 解析默认用户可见输出语言，优先级：
//
//	环境变量 ARTEX_LANG  >  配置文件（language）  >  DefaultLanguage
//
// 这里只是启动默认值。用户在界面里选定的运行期值保存在设置表中并优先生效——
// 服务端把两者分层后接线给智能体与消息目录。
func Language() string {
	if v := NormalizeLanguage(os.Getenv("ARTEX_LANG")); v != "" {
		return v
	}
	if v := NormalizeLanguage(Load().Language); v != "" {
		return v
	}
	return DefaultLanguage
}

// BaseDir is the directory that anchors all runtime artifacts (config.json and
// the data/ store). It is the directory the running binary lives in, so a
// distributed executable keeps its files next to itself on any OS (Windows,
// Linux, …) regardless of the working directory it is launched from.
//
// When launched via `go run`, the binary is throwaway: it sits either in a temp
// build dir (cache miss → fresh link) OR straight inside the Go build cache
// (cache hit → run from $GOCACHE/.../...-d). We detect both and fall back to the
// current working directory so dev artifacts (data/, transcripts) and config
// resolve against the project dir, not the throwaway binary's location.
func BaseDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	dir := filepath.Dir(exe)
	if isGoRunDir(dir) {
		return "." // throwaway `go run` binary → use CWD
	}
	return dir
}

// isGoRunDir reports whether dir is where `go run` parked its executable: under
// the system temp dir (cache miss), or anywhere inside a Go build cache
// (…/go-build/…, the cache-hit case — NOT under os.TempDir(), which is why the
// old temp-only check failed intermittently). In both cases the binary is
// throwaway, so config/data must resolve against the CWD.
func isGoRunDir(dir string) bool {
	if tmp := os.TempDir(); tmp != "" {
		if rel, err := filepath.Rel(tmp, dir); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	for _, seg := range strings.Split(filepath.ToSlash(dir), "/") {
		if seg == "go-build" {
			return true
		}
	}
	return false
}

// Path returns the config file path. Resolution order:
//  1. env ARTEX_CONFIG (explicit override)
//  2. ./config.json in the current working directory (running from the project
//     dir — robust no matter where `go run` placed the temp/cached binary)
//  3. config.json next to the executable (a distributed binary keeps it beside)
//
// The first existing file wins. If none exist, the CWD path is returned so the
// "not found" message points at the project dir the user most likely expected.
func Path() string {
	if v := strings.TrimSpace(os.Getenv("ARTEX_CONFIG")); v != "" {
		return v
	}
	var candidates []string
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "config.json"))
	}
	candidates = append(candidates, filepath.Join(BaseDir(), "config.json"))
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return candidates[0]
}

// Load reads and parses the config file. A missing/unreadable file yields a zero
// Config (so callers fall back to defaults) rather than an error.
func Load() Config {
	var c Config
	b, err := os.ReadFile(Path())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(b, &c)
	return c
}

// SkillDir returns the skill root directory with precedence:
//
//	env ARTEX_SKILL_DIR  >  config file (skill_dir)  >  BaseDir()/skills
//
// The directory is created if it does not exist.
func SkillDir() string {
	var d string
	if v := strings.TrimSpace(os.Getenv("ARTEX_SKILL_DIR")); v != "" {
		d = v
	} else if v := strings.TrimSpace(Load().SkillDir); v != "" {
		d = v
	} else {
		d = filepath.Join(BaseDir(), "skills")
	}
	_ = os.MkdirAll(d, 0o755)
	return d
}

// PostgresDSN resolves the connection string with precedence:
//
//	env ARTEX_PG_DSN  >  config file (database.dsn, or assembled from fields)
//
// There is NO built-in fallback: when neither source supplies a database config,
// it returns an error naming the config path it inspected, so startup fails loudly
// instead of silently connecting to a wrong default. source describes where the
// DSN came from (for startup logging).
func PostgresDSN() (dsn, source string, err error) {
	if v := strings.TrimSpace(os.Getenv("ARTEX_PG_DSN")); v != "" {
		return v, trCfg("环境变量 ARTEX_PG_DSN"), nil
	}
	db := Load().Database
	if d := strings.TrimSpace(db.DSN); d != "" {
		return d, fmt.Sprintf(trCfg("配置文件 %s (database.dsn)"), Path()), nil
	}
	if db.Host != "" || db.DBName != "" || db.User != "" {
		return db.buildDSN(), fmt.Sprintf(trCfg("配置文件 %s (database 字段)"), Path()), nil
	}
	return "", "", fmt.Errorf(trCfg("找不到数据库配置：环境变量 ARTEX_PG_DSN 未设置，配置文件 %s 中也没有 database（dsn 或 host/user/dbname）配置。请创建配置文件或设置环境变量后重试"), Path())
}

func (d Database) buildDSN() string {
	host := d.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := d.Port
	if port == 0 {
		port = 5432
	}
	ssl := d.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	u := url.URL{
		Scheme: "postgres",
		Host:   host + ":" + strconv.Itoa(port),
		Path:   "/" + d.DBName,
	}
	if d.User != "" {
		if d.Password != "" {
			u.User = url.UserPassword(d.User, d.Password)
		} else {
			u.User = url.User(d.User)
		}
	}
	u.RawQuery = url.Values{"sslmode": {ssl}}.Encode()
	return u.String()
}

// String is a redacted view of the resolved DSN (password masked) for logging.
func Redact(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	if u.User != nil {
		if _, hasPw := u.User.Password(); hasPw {
			u.User = url.UserPassword(u.User.Username(), "****")
		}
	}
	return fmt.Sprintf("%s", u.String())
}
