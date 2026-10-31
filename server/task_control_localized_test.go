package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// 回归防护测试：task_control.go 的任务与意图控制 API 错误响应必须保持简体中文。
// 中文判定复用 F3a 的 assertChineseError（含汉字、不含韩文），响应正文提取复用
// task_categories 测试里的 decodeErrorField（同 package server）。

// TestTaskControlErrorConstantsLocalized 断言十个控制响应常量全部为简体中文。
// 意图控制路径（applyIntentControl）第一行就要过 t.Store.GetNode(DB) 才能到达，
// 在没有 DB 的机器上跑不到底，因此其中 6 个直接断言常量本身
// （intent_intervention·goals_api 先例）。任务控制与批量路径由下面两个测试检查到
// 真实函数执行与 HTTP 响应正文。
func TestTaskControlErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"task_deleting":          errTaskCtrlDeleting,
		"task_terminal_pause":    errTaskCtrlTerminalPause,
		"task_already_paused":    errTaskCtrlAlreadyPaused,
		"batch_size":             errTaskCtrlBatchSizeFmt,
		"intent_inherited":       errIntentCtrlInheritedReadonly,
		"intent_only_running":    errIntentCtrlOnlyRunningPause,
		"intent_only_paused":     errIntentCtrlOnlyPausedResume,
		"intent_state_conflict":  errIntentCtrlStateConflictFmt,
		"intent_only_deletable":  errIntentCtrlOnlyDeletable,
		"intent_reason_required": errIntentCtrlReasonRequired,
	}
	for label, msg := range cases {
		assertChineseError(t, label, msg)
	}
}

// TestTaskControlApplyResponsesLocalized 用 applyTaskControl 真实执行不经过 DB 与引擎
// 状态就能走完的 pause 控制路径，确认常量确实被装进返回的错误。三个分支的顺序都是
// s.m.Task(查表) → 删除闸门 → t.lifecycleSnapshot()（直接读内存字段 Status/Paused），
// 因此只要在 tasks 映射里放一个任务并配上空 Engine，无需 Store 与 DB 就能跑通。
// 成功路径（真正暂停）要经过 ApplyTaskPause(DB)，这里不去驱动——所有错误分支都在那之前
// 返回。
func TestTaskControlApplyResponsesLocalized(t *testing.T) {
		// 终态任务：isTerminalStatus(done) 分支。
	terminalServer := func() (*Server, *Task) {
		tk := &Task{ID: "1", Status: "done"}
		return &Server{m: &Manager{tasks: map[string]*Task{"1": tk}}, engine: &Engine{}}, tk
	}
		// 已暂停的运行中任务：lifecycle.Paused 分支。
	pausedServer := func() (*Server, *Task) {
		tk := &Task{ID: "1", Status: "running", Paused: true}
		return &Server{m: &Manager{tasks: map[string]*Task{"1": tk}}, engine: &Engine{}}, tk
	}
		// 已立起删除闸门的任务：IsDeleting 分支。
	deletingServer := func() (*Server, *Task) {
		tk := &Task{ID: "1", Status: "running"}
		s := &Server{m: &Manager{tasks: map[string]*Task{"1": tk}}, engine: &Engine{}}
		s.engine.deleting.Store("1", true)
		return s, tk
	}

	cases := []struct {
		name  string
		setup func() (*Server, *Task)
		want  string
	}{
		{"terminal", terminalServer, errTaskCtrlTerminalPause},
		{"already-paused", pausedServer, errTaskCtrlAlreadyPaused},
		{"deleting", deletingServer, errTaskCtrlDeleting},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, tk := c.setup()
			_, err := s.applyTaskControl(tk, "pause")
			if err == nil {
				t.Fatalf("期望返回错误，实际返回 nil")
			}
			if err.Error() != c.want {
				t.Fatalf("返回错误 = %q, 期望 = %q", err.Error(), c.want)
			}
			assertChineseError(t, c.name, err.Error())
		})
	}
}

// TestControlTasksBatchResponsesLocalized 用真实 HTTP 检查批量控制端点的响应正文。
// ① 空的 task_ids 在访问 s.m 之前就经 writeErr 返回 400 + 大小错误，因此 &Server{} 也能到达。
// ② 对一个终态任务批量 pause 会以 200 + items[].error 原样带回任务控制错误，
// 由此确认 applyTaskControl 的错误能一路传播到批量 JSON。
func TestControlTasksBatchResponsesLocalized(t *testing.T) {
	t.Run("size-error", func(t *testing.T) {
		s := &Server{}
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/control",
			strings.NewReader(`{"task_ids":[],"action":"pause"}`))
		rec := httptest.NewRecorder()
		s.controlTasksBatch(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, 期望 = 400 (正文 %q)", rec.Code, rec.Body.String())
		}
		got := decodeErrorField(t, rec.Body.Bytes())
		want := fmt.Sprintf(errTaskCtrlBatchSizeFmt, maxBatchControlIDs)
		if got != want {
			t.Fatalf("响应文案 = %q, 期望 = %q", got, want)
		}
		if !strings.Contains(got, "100") {
			t.Fatalf("文案中未体现数量上限 100：%q", got)
		}
		assertChineseError(t, "size-error", got)
	})

	t.Run("item-error-surfaces", func(t *testing.T) {
		tk := &Task{ID: "1", Status: "done"}
		s := &Server{m: &Manager{tasks: map[string]*Task{"1": tk}}, engine: &Engine{}}
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/control",
			strings.NewReader(`{"task_ids":["1"],"action":"pause"}`))
		rec := httptest.NewRecorder()
		s.controlTasksBatch(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("状态码 = %d, 期望 = 200 (正文 %q)", rec.Code, rec.Body.String())
		}
		var out struct {
			Items []struct {
				ID    string `json:"id"`
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			} `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("响应 JSON 解析失败：%v (正文 %q)", err, rec.Body.String())
		}
		if len(out.Items) != 1 {
			t.Fatalf("items 数量 = %d, 期望 = 1", len(out.Items))
		}
		if out.Items[0].OK || out.Items[0].Error != errTaskCtrlTerminalPause {
			t.Fatalf("items[0] = %+v, 期望 error = %q", out.Items[0], errTaskCtrlTerminalPause)
		}
		assertChineseError(t, "item-error", out.Items[0].Error)
	})
}

// TestIntentStateConflictWrapsSentinel 确认意图状态冲突文案是简体中文，同时仍用 %w
// 原样包裹 db.ErrIntentStateConflict，errors.Is 识别不被破坏。
// 枚举值 paused 必须原样保留。
func TestIntentStateConflictWrapsSentinel(t *testing.T) {
	err := fmt.Errorf(errIntentCtrlStateConflictFmt, db.ErrIntentStateConflict)
	if !errors.Is(err, db.ErrIntentStateConflict) {
		t.Fatalf("errors.Is(.., ErrIntentStateConflict) = false, %%w 包装被破坏：%q", err.Error())
	}
	if !strings.Contains(err.Error(), "paused") {
		t.Fatalf("文案中未保留枚举值 paused：%q", err.Error())
	}
	assertChineseError(t, "state-conflict", err.Error())
}
