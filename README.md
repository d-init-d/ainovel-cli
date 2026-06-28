# ainovel-cli — Research Edition (Tiếng Việt)

> CLI đa agent để nghiên cứu, lập kế hoạch, viết, biên tập và duy trì tính nhất quán cho tiểu thuyết dài kỳ.

<p align="center">
  <img src="scripts/sample.gif" alt="ainovel-cli terminal demo" width="820">
</p>

Phiên bản này kế thừa bản Việt hoá của Kent Juno, giữ nguyên kiến trúc sáng tác cốt lõi của dự án tiếng Trung do voocel phát triển, đồng thời bổ sung quy trình **research-first**, tài liệu đính kèm và plugin `d-research` đóng gói độc lập.

## Nguồn gốc và ghi công

Dự án tồn tại nhờ ba lớp phát triển nối tiếp:

1. **Dự án tiếng Trung nguyên bản:** [voocel/ainovel-cli](https://github.com/voocel/ainovel-cli) — kiến trúc đa agent, TUI, store, checkpoint và quy trình sáng tác cốt lõi.
2. **Bản Việt hoá:** [kentjuno/ainovel-cli](https://github.com/kentjuno/ainovel-cli) — Việt hoá giao diện, prompt hệ thống và tài liệu.
3. **Research Edition:** [d-init-d/ainovel-cli](https://github.com/d-init-d/ainovel-cli) — tích hợp [d-init-d/d-research-skill](https://github.com/d-init-d/d-research-skill), `/d-research`, attachment TXT/Markdown/DOCX và evidence ledger.

Xin cảm ơn voocel, Kent Juno và toàn bộ contributor của các dự án nguồn. Bản fork này không tuyên bố quyền tác giả đối với phần công việc của upstream.

## Điểm nổi bật

| Khả năng | Mô tả |
|---|---|
| Đa agent tự chủ | Coordinator điều phối Architect, Writer và Editor trong một vòng lặp dài hạn |
| Research-first | Tìm kiếm web, duyệt trang động bằng Playwright, trích xuất bằng chứng và kiểm tra mâu thuẫn trước khi viết |
| Attachment theo prompt | Đính kèm `.txt`, `.md`, `.markdown`, `.docx` làm knowledge cho prompt kế tiếp mà không thay đổi trạng thái truyện |
| Nhập tiểu thuyết có sẵn | Giữ nguyên `/import` của upstream để tái dựng trạng thái một cuốn sách và tiếp tục hoặc chỉnh sửa |
| Quản lý truyện dài | Outline cuộn, compass, world rules, timeline, nhân vật, phục bút và bộ nhớ phân tầng |
| Khôi phục chính xác | Checkpoint sau các bước quan trọng, có thể tiếp tục sau crash, mất mạng hoặc Ctrl+C |
| Can thiệp thời gian thực | Người dùng có thể chỉnh hướng truyện trong lúc agent đang chạy |
| Đa provider | OpenRouter, OpenAI, Anthropic, Gemini, DeepSeek, Qwen, GLM, Grok, Ollama, Bedrock và proxy tuỳ chỉnh |

## Kiến trúc

```text
Người dùng / Attachment / D Research
                  │
                  ▼
        Coordinator (điều phối)
          ┌───────┼────────┐
          ▼       ▼        ▼
      Architect  Writer   Editor
          │       │        │
          └───────┼────────┘
                  ▼
      Tools + Store + Checkpoints
                  │
                  ▼
       Novel state / Research pack
```

Host chịu trách nhiệm khởi động, quan sát, IO và khôi phục. Quyết định sáng tác vẫn thuộc về agent. Tài liệu người dùng và nội dung web được đánh dấu là dữ liệu không tin cậy, không được phép ghi đè prompt hệ thống hoặc tự biến thành yêu cầu viết lại.

## Yêu cầu hệ thống

### Chức năng sáng tác cốt lõi

- Go `1.25.5` trở lên khi build từ source.
- Một provider/model tương thích, hoặc Ollama chạy local.
- Terminal hỗ trợ màu ANSI; Windows Terminal được khuyến nghị trên Windows.

### D Research

- Node.js `18+`.
- Playwright và Chromium nếu cần browser probe/extract/crawl.
- Kết nối Internet cho web search và nguồn công khai.

Nếu Node.js hoặc Playwright chưa sẵn sàng, chức năng viết truyện cốt lõi vẫn hoạt động. Research sẽ hạ cấp theo capability có sẵn và ghi lại cảnh báo/blocker.

## Cài đặt native

### Windows PowerShell

```powershell
git clone https://github.com/d-init-d/ainovel-cli.git
Set-Location ainovel-cli

go build -trimpath -o ainovel-cli.exe ./cmd/ainovel-cli

Set-Location plugins/d-research
npm ci
npx playwright install chromium
Set-Location ../..

.\ainovel-cli.exe
```

### Linux / macOS

```bash
git clone https://github.com/d-init-d/ainovel-cli.git
cd ainovel-cli

go build -trimpath -o ainovel-cli ./cmd/ainovel-cli

cd plugins/d-research
npm ci
npx playwright install chromium
cd ../..

./ainovel-cli
```

Plugin được tìm theo thứ tự: `research.plugin_path`, `plugins/d-research` trong thư mục hiện tại, `plugins/d-research` cạnh executable, rồi các bản cài trong `~/.codex/skills` và `~/.agents/skills`. Khi di chuyển binary sang thư mục khác, hãy giữ nguyên thư mục `plugins/d-research` bên cạnh binary.

> Dockerfile kế thừa upstream hiện chỉ đóng gói runtime sáng tác cốt lõi. Để sử dụng Chromium/Playwright đầy đủ, bản native là lựa chọn được khuyến nghị.

## Cấu hình

Lần chạy đầu tiên có thể mở trình thiết lập. File cấu hình người dùng nằm tại:

- Windows: `%USERPROFILE%\.ainovel\config.json`
- Linux/macOS: `~/.ainovel/config.json`
- Tuỳ chọn theo dự án: `.ainovel/config.json`

### Ollama

```json
{
  "provider": "ollama",
  "model": "qwen3:14b",
  "providers": {
    "ollama": {
      "base_url": "http://localhost:11434/v1",
      "models": ["qwen3:14b"]
    }
  }
}
```

`base_url` của endpoint tương thích OpenAI do Ollama cung cấp phải có hậu tố `/v1`. Ollama không bắt buộc API key.

### OpenRouter

```json
{
  "provider": "openrouter",
  "model": "google/gemini-2.5-flash",
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-...",
      "base_url": "https://openrouter.ai/api/v1",
      "models": ["google/gemini-2.5-flash"]
    }
  }
}
```

Không commit API key vào Git. Có thể dùng provider/model khác cho từng vai trò qua trường `roles` trong config mẫu tại [`internal/bootstrap/config.example.jsonc`](internal/bootstrap/config.example.jsonc).

### Bật D Research

Thêm khối `research` vào config:

```json
{
  "research": {
    "enabled": true,
    "plugin": "d-research",
    "auto": true,
    "max_queries": 8,
    "max_results_per_query": 6,
    "max_sources": 12,
    "timeout_seconds": 120,
    "browser": {
      "enabled": true,
      "headless": true,
      "timeout_seconds": 30,
      "extract": true
    }
  }
}
```

`auto: true` cho phép Architect chủ động nghiên cứu khi truyện cần căn cứ khoa học, kỹ thuật, lịch sử, pháp lý hoặc dữ liệu thực tế. `/d-research` vẫn có thể được gọi thủ công.

## Quy trình sử dụng

### 1. Bắt đầu một truyện mới

Khởi động TUI, nhập yêu cầu và nhấn Enter:

```text
Viết hard science fiction về một tàu thế hệ mất khả năng tản nhiệt giữa hành trình.
Khoảng 60 chương, ưu tiên tính hợp lý vật lý và xung đột đạo đức.
```

### 2. Nghiên cứu trước khi viết

```text
/d-research giới hạn vật lý của tàu thế hệ và hệ thống tản nhiệt freshness=2024-01-01 max=10
```

Có thể truyền URL và file trực tiếp:

```text
/d-research sinh học ngủ đông file="D:\Docs\hibernation.md" url=https://example.org/paper
```

Kết quả được lưu tại:

```text
<output>/meta/research/<report-id>/
├── report.json
├── evidence-ledger.json
└── compact.json
```

### 3. Đính kèm knowledge cho prompt kế tiếp

```text
/attach "D:\Docs\world-building.docx" "D:\Docs\physics-notes.md"
Hãy dùng tài liệu đính kèm để kiểm tra world rules và kế hoạch chương 4.
```

- `/attach` hỗ trợ TXT, Markdown và DOCX hiện đại.
- Attachment chỉ áp dụng cho prompt bình thường kế tiếp rồi tự xoá.
- Nội dung lớn được trích xuất cục bộ và chọn các đoạn liên quan đến prompt, không nhét nguyên file vào context.
- `/attachments` hiển thị hàng đợi; `/detach` xoá hàng đợi.
- Nếu lệnh kế tiếp là `/d-research`, attachment được chuyển thành `file_paths` cho research.
- `.doc` nhị phân đời cũ không được đọc trực tiếp; hãy lưu lại thành `.docx`, `.txt` hoặc `.md`.

### 4. Nhập một tiểu thuyết để tiếp tục hoặc sửa

```text
/import "D:\Books\source-novel.txt"
```

`/import` là chức năng upstream và giữ nguyên ngữ nghĩa: file được coi là tiểu thuyết nguồn để tái dựng foundation/chapter state. Không dùng `/import` cho knowledge hoặc tài liệu nghiên cứu; dùng `/attach` thay thế.

## Lệnh TUI

| Lệnh | Chức năng |
|---|---|
| `/help` | Danh sách lệnh và phím tắt |
| `/model [role]` | Chuyển model mặc định hoặc model theo vai trò |
| `/diag` | Chẩn đoán tiến độ, vòng lặp và tính nhất quán |
| `/d-research <goal> ...` | Chạy research-first và lưu research pack |
| `/research <goal> ...` | Alias của `/d-research` |
| `/attach <path...>` | Đính kèm knowledge cho prompt kế tiếp |
| `/attachments` | Xem attachment đang chờ |
| `/detach` | Xoá attachment đang chờ |
| `/import <path> [from=N]` | Nhập tiểu thuyết để tiếp tục/sửa |
| `/simulate` | Xây dựng hồ sơ phong cách từ thư mục `simulate` |
| `/importsim <profile.json>` | Nhập hồ sơ mô phỏng phong cách |
| `/cocreate` | Tạm dừng để đồng sáng tác kế hoạch |
| `/export [path] ...` | Xuất TXT hoặc EPUB |

Gõ `/` để mở command palette. Đường dẫn có khoảng trắng phải đặt trong dấu nháy. Trên terminal hỗ trợ kéo-thả đường dẫn, có thể gõ `/attach ` rồi thả file vào ô nhập.

## Ranh giới giữa attachment và import

| Hành động | Có thay đổi novel state? | Cách agent hiểu file |
|---|---:|---|
| `/attach` | Không | Knowledge không tin cậy cho prompt kế tiếp |
| `/d-research ... file=...` | Chỉ cập nhật research pack sau khi nghiên cứu | Nguồn bằng chứng |
| `/import` | Có | Tiểu thuyết nguồn cần tái dựng trạng thái |

Ranh giới này ngăn tài liệu khoa học hoặc world-building bị hiểu nhầm thành một cuốn sách cần viết lại.

## Dữ liệu đầu ra

```text
output/novel/
├── chapters/                 # Chương đã hoàn tất
├── drafts/                   # Bản nháp
├── reviews/                  # Đánh giá của Editor
├── summaries/                # Tóm tắt chương/cung/tập
├── premise.md
├── outline.json
├── compass.json
├── characters.json
├── world_rules.json
└── meta/
    ├── checkpoints.jsonl
    ├── sessions/
    └── research/
```

Vị trí cụ thể có thể thay đổi theo `output_dir` và workspace của người dùng.

## An toàn và quyền riêng tư

- D Research chỉ truy cập nguồn công khai, ở chế độ read-only.
- Không vượt login wall, paywall, CAPTCHA, rate limit hoặc kiểm soát truy cập.
- Nguồn bị chặn được ghi thành blocker thay vì bị bịa dữ liệu thay thế.
- File và nội dung web được coi là dữ liệu không tin cậy để giảm rủi ro prompt injection.
- Đoạn trích từ attachment hoặc research có thể được gửi tới model provider đã cấu hình. Không đính kèm dữ liệu mật nếu deployment/provider chưa được phép nhận dữ liệu đó.
- DOCX được giải nén với giới hạn an toàn để chống ZIP bomb; file cực lớn có thể bị từ chối hoặc chỉ trích xuất trong ngân sách an toàn.

## Phát triển và kiểm thử

```bash
go test ./internal/document ./internal/entry/tui ./internal/research ./internal/host/imp
go test ./internal/agents
go build ./cmd/ainovel-cli

cd plugins/d-research
npm ci
npm run self-test
```

Tài liệu kỹ thuật bổ sung:

- [`docs/d-research-plugin.md`](docs/d-research-plugin.md)
- [`docs/architecture.md`](docs/architecture.md)
- [`docs/context-management.md`](docs/context-management.md)
- [`docs/observability.md`](docs/observability.md)

## Giấy phép

- Mã nguồn ainovel-cli và các thay đổi kế thừa được phân phối theo **Apache License 2.0**; xem [`LICENSE`](LICENSE).
- Bundle `plugins/d-research` có giấy phép riêng **CC BY-NC 4.0**; xem [`plugins/d-research/LICENSE`](plugins/d-research/LICENSE).

Khi phân phối lại, hãy giữ nguyên các thông báo bản quyền, giấy phép và phần ghi công cho [voocel/ainovel-cli](https://github.com/voocel/ainovel-cli), [kentjuno/ainovel-cli](https://github.com/kentjuno/ainovel-cli) và [d-init-d/d-research-skill](https://github.com/d-init-d/d-research-skill).
