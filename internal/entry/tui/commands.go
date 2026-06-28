package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/voocel/ainovel-cli/internal/entry/startup"
	"github.com/voocel/ainovel-cli/internal/host"
)

type slashCommandSpec struct {
	Name        string
	Aliases     []string
	Group       string
	Usage       string
	Description string
	AutoExecute bool
	Hidden      bool
	NeedsIdle   bool
	Run         func(m Model, args []string) (tea.Model, tea.Cmd)
}

type slashCommand struct {
	name    string
	args    []string
	rawArgs string
}

func parseSlashCommand(text string) (slashCommand, bool) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return slashCommand{}, false
	}
	body := strings.TrimSpace(strings.TrimPrefix(text, "/"))
	if body == "" {
		return slashCommand{}, false
	}
	separator := strings.IndexFunc(body, unicode.IsSpace)
	if separator < 0 {
		return slashCommand{name: strings.ToLower(body)}, true
	}
	name := body[:separator]
	rawArgs := strings.TrimSpace(body[separator:])
	return slashCommand{name: strings.ToLower(name), args: splitCommandArgs(rawArgs), rawArgs: rawArgs}, true
}

func splitCommandArgs(text string) []string {
	var args []string
	var b strings.Builder
	var quote rune
	flush := func() {
		if b.Len() == 0 {
			return
		}
		args = append(args, b.String())
		b.Reset()
	}
	for _, r := range text {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			b.WriteRune(r)
		case r == '"' || r == '\'':
			quote = r
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return args
}

func (s slashCommandSpec) matches(name string) bool {
	if s.Name == name {
		return true
	}
	for _, alias := range s.Aliases {
		if strings.EqualFold(alias, name) {
			return true
		}
	}
	return false
}

func commandRegistryInstance() commandRegistry {
	return newCommandRegistry([]slashCommandSpec{
		{
			Name:        "help",
			Group:       "system",
			Usage:       "/help",
			Description: "Xem danh sách lệnh",
			AutoExecute: true,
			Run: func(m Model, _ []string) (tea.Model, tea.Cmd) {
				m.help = newHelpState(m.width, m.height)
				m.textarea.Blur()
				return m, nil
			},
		},
		{
			Name:        "model",
			Group:       "system",
			Usage:       "/model [role]",
			Description: "Chuyển đổi mô hình mặc định hoặc theo vai trò",
			AutoExecute: true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				roleHint := ""
				if len(args) > 0 {
					roleHint = args[0]
					if normalizeRoleKey(roleHint) == "" {
						m.applyEvent(host.Event{
							Time: time.Now(), Category: "ERROR", Summary: "Vai trò không xác định: " + roleHint, Level: "error",
						})
						m.refreshEventViewport()
						return m, nil
					}
				}
				m.modelSwitch = newModelSwitchState(m.runtime, roleHint)
				m.textarea.Blur()
				return m, nil
			},
		},
		{
			Name:        "diag",
			Group:       "analysis",
			Usage:       "/diag",
			Description: "Chẩn đoán tình trạng sáng tác tiểu thuyết",
			AutoExecute: true,
			Run: func(m Model, _ []string) (tea.Model, tea.Cmd) {
				m.reportSeq++
				m.report = newReportState(m.width, m.height, m.reportSeq, time.Now())
				m.textarea.Blur()
				return m, loadReport(m.runtime.Dir(), m.reportSeq)
			},
		},
		{
			Name:        "attach",
			Group:       "system",
			Usage:       "/attach <file.txt|file.md|file.docx> [file khác...]",
			Description: "Đính kèm tài liệu tham khảo cho prompt kế tiếp",
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				queued, err := queueAttachments(m.attachments, args)
				if err != nil {
					m.applyEvent(host.Event{Time: time.Now(), Category: "ERROR", Summary: err.Error(), Level: "error"})
					m.refreshEventViewport()
					return m, nil
				}
				m.attachments = queued
				m.applyEvent(host.Event{
					Time: time.Now(), Category: "SYSTEM",
					Summary: fmt.Sprintf("Đã đính kèm %d tài liệu cho prompt kế tiếp", len(m.attachments)), Level: "info",
				})
				m.refreshEventViewport()
				return m, nil
			},
		},
		{
			Name:        "attachments",
			Group:       "system",
			Usage:       "/attachments",
			Description: "Xem tài liệu đang chờ gửi cùng prompt kế tiếp",
			AutoExecute: true,
			Run: func(m Model, _ []string) (tea.Model, tea.Cmd) {
				names := make([]string, 0, len(m.attachments))
				for _, attachment := range m.attachments {
					names = append(names, filepath.Base(attachment.Path))
				}
				summary := "Không có attachment đang chờ"
				if len(names) > 0 {
					summary = "Attachment đang chờ: " + strings.Join(names, ", ")
				}
				m.applyEvent(host.Event{Time: time.Now(), Category: "SYSTEM", Summary: summary, Level: "info"})
				m.refreshEventViewport()
				return m, nil
			},
		},
		{
			Name:        "detach",
			Group:       "system",
			Usage:       "/detach",
			Description: "Bỏ toàn bộ tài liệu đang chờ",
			AutoExecute: true,
			Run: func(m Model, _ []string) (tea.Model, tea.Cmd) {
				m.attachments = nil
				m.applyEvent(host.Event{Time: time.Now(), Category: "SYSTEM", Summary: "Đã bỏ toàn bộ attachment đang chờ", Level: "info"})
				m.refreshEventViewport()
				return m, nil
			},
		},
		{
			Name:        "d-research",
			Aliases:     []string{"research"},
			Group:       "analysis",
			Usage:       "/d-research <goal> [file=<path>] [url=<url>] [freshness=YYYY-MM-DD] [max=N]",
			Description: "Yêu cầu Kiến trúc sư chạy nghiên cứu d-research và lưu research_pack",
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				if !m.runtime.ResearchEnabled() {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: "d-research chưa bật trong config: đặt research.enabled=true rồi khởi động lại", Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				researchArgs := append([]string(nil), args...)
				for _, attachment := range m.attachments {
					researchArgs = append(researchArgs, "file="+attachment.Path)
				}
				prompt, err := buildDResearchCommandPrompt(researchArgs)
				if err != nil {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: err.Error(), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				m.attachments = nil
				switch m.mode {
				case modeNew:
					m.err = nil
					if m.startupMode == startupModeQuick {
						plan, err := startup.PrepareQuick(startup.Request{
							Mode:        startup.ModeQuick,
							UserPrompt:  prompt,
							OutputDir:   m.runtime.Dir(),
							Interactive: true,
						})
						if err != nil {
							m.err = err
							return m, nil
						}
						return m, startRuntime(m.runtime, plan)
					}
					m.cocreate = newCoCreateState(prompt)
					return m, m.sendCoCreate()
				case modeRunning:
					if m.snapshot.IsRunning {
						return m, steerRuntime(m.runtime, prompt)
					}
					return m, continueRuntime(m.runtime, prompt)
				case modeDone:
					m.mode = modeRunning
					return m, continueRuntime(m.runtime, prompt)
				default:
					return m, nil
				}
			},
		},
		{
			Name:        "import",
			Group:       "writing",
			Usage:       "/import <path> [from=N]",
			Description: "Nhập truyện bên ngoài để tiếp tục viết",
			NeedsIdle:   true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				m.importSeq++
				state, listenCmd, err := startImport(m.runtime, m.importSeq, args, m.width, m.height)
				if err != nil {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: "Khởi động nhập truyện thất bại: " + err.Error(), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				m.importer = state
				m.textarea.Blur()
				return m, listenCmd
			},
		},
		{
			Name:        "cocreate",
			Aliases:     []string{"plan"},
			Group:       "writing",
			Usage:       "/cocreate",
			Description: "Tạm dừng sáng tác, đồng sáng tác lên kế hoạch cho các giai đoạn tiếp theo",
			AutoExecute: true,
			Run: func(m Model, _ []string) (tea.Model, tea.Cmd) {
				if m.mode != modeRunning {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: "Đồng sáng tác giai đoạn chỉ khả dụng khi đang sáng tác", Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				if !m.runtime.PauseForCoCreate() {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: "Không thể vào đồng sáng tác giai đoạn: toàn bộ tác phẩm đã hoàn thành hoặc đang trong quá trình đồng sáng tác", Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				m.cocreate = newStageCoCreateState()
				m.resizeTextarea()
				m.textarea.Blur()
				return m, m.sendCoCreate()
			},
		},
		{
			Name:        "simulate",
			Group:       "writing",
			Usage:       "/simulate",
			Description: "Đọc ./simulate để tạo hoặc cập nhật tăng dần hồ sơ mô phỏng phong cách viết",
			NeedsIdle:   true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				m.simSeq++
				state, listenCmd, err := startSimulate(m.runtime, m.simSeq, args, m.width, m.height)
				if err != nil {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: "Khởi động hồ sơ mô phỏng phong cách viết thất bại: " + err.Error(), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				m.simulator = state
				m.textarea.Blur()
				return m, listenCmd
			},
		},
		{
			Name:        "importsim",
			Group:       "writing",
			Usage:       "/importsim <profile.json>",
			Description: "Nhập hồ sơ mô phỏng phong cách có sẵn và hợp nhất theo dấu vân tay ngữ liệu",
			NeedsIdle:   true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				m.simSeq++
				state, listenCmd, err := startImportSimulation(m.runtime, m.simSeq, args, m.width, m.height)
				if err != nil {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: "Nhập hồ sơ mô phỏng phong cách thất bại: " + err.Error(), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				m.simulator = state
				m.textarea.Blur()
				return m, listenCmd
			},
		},
		{
			Name:        "export",
			Group:       "writing",
			Usage:       "/export [path] [from=N] [to=M] [--overwrite]",
			Description: "Xuất truyện các chương đã hoàn thành sang TXT/EPUB",
			AutoExecute: true,
			Run: func(m Model, args []string) (tea.Model, tea.Cmd) {
				cmd, err := startExport(m.runtime, args)
				if err != nil {
					m.applyEvent(host.Event{
						Time: time.Now(), Category: "ERROR", Summary: "Khởi động xuất truyện thất bại: " + err.Error(), Level: "error",
					})
					m.refreshEventViewport()
					return m, nil
				}
				m.applyEvent(host.Event{
					Time: time.Now(), Category: "SYSTEM", Summary: "Đang xuất truyện...", Level: "info",
				})
				m.refreshEventViewport()
				return m, cmd
			},
		},
	})
}

func commandSpecs() []slashCommandSpec {
	return commandRegistryInstance().Visible()
}

func (m Model) handleSlashCommand(cmd slashCommand) (tea.Model, tea.Cmd) {
	spec, ok := commandRegistryInstance().Find(cmd.name)
	if !ok {
		m.applyEvent(host.Event{
			Time: time.Now(), Category: "ERROR", Summary: "Lệnh không xác định: /" + cmd.name, Level: "error",
		})
		m.refreshEventViewport()
		return m, nil
	}
	if spec.NeedsIdle && m.snapshot.IsRunning {
		m.applyEvent(host.Event{
			Time: time.Now(), Category: "ERROR", Summary: "Lệnh chỉ có thể thực thi khi ở trạng thái rảnh: /" + spec.Name, Level: "error",
		})
		m.refreshEventViewport()
		return m, nil
	}
	return spec.Run(m, cmd.args)
}
