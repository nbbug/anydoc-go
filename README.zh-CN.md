**[English](README.md) · 中文文档**

# anydoc-go

[anydoc](https://github.com/firecrawl/anydoc) 的 Go 语言绑定 —— anydoc 是一个
Rust 库，可将 Word、PowerPoint、Excel、OpenDocument、RTF、EPUB、CSV 和 PDF
等文档转换为 GitHub Flavored Markdown。

anydoc-go 通过 cgo 以**静态库**的方式链接 anydoc，并在模块内为所有支持的平台
打包了预编译的静态库。使用者**无需安装 Rust 或任何额外工具链**：`go get`
即可拉取静态库，`go build` 直接完成链接。

## 环境要求

|          |                                                                                 |
| -------- | ------------------------------------------------------------------------------- |
| Go       | ≥ 1.22                                                                          |
| CGO      | `CGO_ENABLED=1`（所有支持平台上的 Go 默认值）                                   |
| C 工具链 | 各系统标准配置 —— macOS 用 Xcode 命令行工具、Linux 用 gcc、Windows 用 mingw-w64 |
| Rust     | 使用本模块**不需要**；只有重新编译静态库时才需要                                |

## 安装

```
go get github.com/nbbug/anydoc-go
```

## 快速上手

```go
package main

import (
	"fmt"
	"os"

	anydoc "github.com/nbbug/anydoc-go"
)

func main() {
	// 转换文件：格式从文件内容自动识别。
	markdown, err := anydoc.ToMarkdown("report.docx")
	if err != nil {
		panic(err)
	}
	fmt.Print(markdown)

	// 转换内存中的字节。CSV 这类无签名的格式必须显式指定。
	data, err := os.ReadFile("data.csv")
	if err != nil {
		panic(err)
	}
	md, err := anydoc.ToMarkdownBytes(data, &anydoc.FormatCsv)
	// ...
	_ = md
}
```

可直接运行的示例见 [examples/](examples/)：

```
go run ./examples/convert -file report.docx
go run ./examples/parse   -file report.docx
```

## 支持的平台

模块为以下每个组合打包了预编译静态库，Go 编译器通过构建标签（build tags）
自动选择正确的文件。每个静态库都是 `lib/<platform>/` 目录下的一个文件：
除 Windows 外均为 `libanydoc_go.a`，Windows 为 MSVC 的 COFF 归档
`anydoc_go.lib`。

| 平台          | Rust 目标                   | 链接所用的 C 工具链       |
| ------------- | --------------------------- | ------------------------- |
| linux/amd64   | `x86_64-unknown-linux-gnu`  | gcc                       |
| linux/arm64   | `aarch64-unknown-linux-gnu` | gcc (aarch64)             |
| darwin/amd64  | `x86_64-apple-darwin`       | clang（Xcode 命令行工具） |
| darwin/arm64  | `aarch64-apple-darwin`      | clang（Xcode 命令行工具） |
| windows/amd64 | `x86_64-pc-windows-msvc`    | mingw-w64 gcc             |
| windows/arm64 | `aarch64-pc-windows-msvc`   | mingw-w64 gcc             |

说明：

- **Windows 使用 MSVC 目标**（`x86_64-pc-windows-msvc` 与
  `aarch64-pc-windows-msvc`），与 WeKnora 绑定一致。静态库由 MSVC 工具链
  构建（MSVC 不可再分发，因此只能在原生 Windows 主机或 CI 的 Windows
  运行器上构建），产物为 `anydoc_go.lib`。Go 的 cgo 仍然使用 mingw gcc
  链接 —— GNU ld 可以读取 MSVC 的 `.lib` 归档 —— 因此使用者只需要标准的
  mingw-w64 工具链，不需要安装 Visual Studio。
- **Linux 为 glibc。**Alpine/musl 用户需要自行重新编译静态库（在 Alpine
  镜像内执行 `./scripts/build-all.sh linux_amd64`，或使用
  `cargo-zigbuild` 后提交产物）。Go 的构建标签无法自动识别 musl，参考项目
  WeKnora 的绑定为此提供了平行的 `-tags musl` 变体；本模块保持最小平台矩阵，
  如有 Alpine 需求可沿用同样的模式扩展。
- 在其他平台（或 `CGO_ENABLED=0`）下构建仍然**可以编译通过**，见下文
  [构建标签与兜底实现](#构建标签与兜底实现)。

## API

```go
// 将文档文件转换为 Markdown。格式从内容自动识别；
// 扩展名作为无签名格式（CSV）的回退。
func ToMarkdown(path string) (string, error)

// 将内存中的文档转换为 Markdown。传入 Format 以指定解析器，
// 或传 nil 从内容自动识别。
func ToMarkdownBytes(data []byte, format *Format) (string, error)

// 带扩展行为的转换。Options 为 nil 时行为与 ToMarkdown /
// ToMarkdownBytes 一致；Ocr = OcrHosted 时，需要 OCR 的 PDF 会发送到
// Firecrawl Parse，而不是报 needs_ocr。
func ToMarkdownWithOptions(path string, opts *Options) (string, error)
func ToMarkdownBytesWithOptions(data []byte, format *Format, opts *Options) (string, error)

// Options：Ocr（OcrReject 默认 / OcrHosted / OcrCustom + OcrHandler）、
// APIKey（缺省回退到 FIRECRAWL_API_KEY，再回退到免密钥）、APIURL（缺省
// 回退到 FIRECRAWL_API_URL，再回退到 https://api.firecrawl.dev）。

// 转换并将内嵌图片重写为 ![alt](images/image-N.ext)，使图片保留原位。
// PDF 没有文档模型，与 ToMarkdownBytes 的转换方式相同。
func ToMarkdownWithAssetLinks(data []byte, format *Format) (string, error)

// 将内存中的文档解析为文档模型，模型同时携带内嵌资源。
// 不支持 PDF（PDF 没有模型形式）——PDF 请使用 ToMarkdownBytes。
func ToDocument(data []byte, format *Format) (*Document, error)

// 格式检测。
func FormatFromBytes(data []byte) (Format, bool)
func FormatFromExtension(ext string) (Format, bool)
func FormatFromPath(path string) (Format, bool)

// 当前平台打包的静态库字节，以及将其写入磁盘的辅助函数。
// 见下文「静态库的嵌入」。
func EmbeddedLib() []byte
func ExtractLib(dir string) (string, error)

// 模块版本，与锁定的 anydoc crate 版本同步。
const Version = "0.2.4"
```

**格式**（[format.go](format.go)）：`FormatDoc`、`FormatDocx`、`FormatOdt`、
`FormatPdf`、`FormatPpt`、`FormatPptx`、`FormatRtf`、`FormatEpub`、
`FormatXlsx`、`FormatOds`、`FormatOdp`、`FormatCsv`。

**错误**（[errors.go](errors.go)）：转换函数返回 `*ConvertError`，其 `Kind`
与 Node/Python 绑定一致 —— `unsupported`、`malformed`、`encrypted`、
`resource_limit`、`missing_part`、`io`、`pdf_no_model`、`needs_ocr` ——
并带有可读的 `Detail`。`needs_ocr` 表示 PDF 中有扫描或纯图片页面（自
anydoc 0.2.4 起，这些页面不再被静默丢弃，而是报错点名需要 OCR 的页码），
并额外携带结构化的 `NeedsOcr{Pages, PageCount}` 字段。传入无效的显式
`Format` 会报 `unknown_format`，Firecrawl Parse 回退失败会报 `hosted`。

**文档模型**（[model.go](model.go)）：`Document`（blocks、notes、assets）、
`Block`（`heading`、`paragraph`、`list`、`table`、`block_quote`、
`code_block`、`rule`、`math`）、`Inline`（`text`、`link`、`image`、`anchor`、
`note_ref`、`line_break`、`math`、`checkbox`），以及 `Style`、`LinkTarget`、
`ImageSource`、`List`、`ListItem`、`Table`、`CellSlot`、`Cell`、`Note`、
`Asset` —— 完整且自包含的表示，内嵌资源字节存放在 `Asset.Data` 中。

### OCR 回退（可选）

PDF 中有扫描或纯图片页面时，anydoc 会报 `needs_ocr` 并点名页码。可以
开启两种可选回退，由它们接管转换。

**`OcrHosted`** 把整个文档发送到
[Firecrawl Parse](https://firecrawl.dev/parse) 提取文字：

```go
md, err := anydoc.ToMarkdownBytesWithOptions(pdf, &anydoc.FormatPdf, &anydoc.Options{
	Ocr: anydoc.OcrHosted,
})
```

无需账号即可使用；设置 `FIRECRAWL_API_KEY`（或 `Options.APIKey`）可提高
速率限制，`FIRECRAWL_API_URL`/`Options.APIURL` 可指向其他端点。失败时错误
kind 为 `hosted`。

**`OcrCustom`** 把文档交给注入的 `OcrHandler` —— 可以是本地 OCR 模型
（如 ONNX）、本地 OCR 服务，或你自己的任意 API：

```go
type onnxOcr struct {
	session *ort.Session // 你的 ONNX 推理会话
}

func (o *onnxOcr) OcrMarkdown(pdf []byte) (string, error) {
	return runOcr(o.session, pdf) // 页面 → 文字 → Markdown，全程本地
}

md, err := anydoc.ToMarkdownBytesWithOptions(pdf, &anydoc.FormatPdf, &anydoc.Options{
	Ocr:        anydoc.OcrCustom,
	OcrHandler: &onnxOcr{session: session},
})

// OcrHandlerFunc 可以适配普通函数：
//   anydoc.OcrHandlerFunc(func(pdf []byte) (string, error) { ... })
```

回调收到整个 PDF，返回提取的 Markdown；它返回的错误原样透传。`OcrCustom`
未设置 `OcrHandler` 时报 `unsupported` 错误。

只有 anydoc 自己无法转换的文档（报 `needs_ocr` 的 PDF）才会进入回退；
其余内容一律走本地转换（且除 `OcrHosted` 外不离开本机）。

## 工作原理

1. **Rust C ABI** —— 本仓库同时也是一个 Rust crate（[Cargo.toml](Cargo.toml)），
   它封装了 crates.io 上发布的 `anydoc` crate（精确锁定版本，当前为
   `=0.2.4`），并暴露一个精简的 C ABI（[src/lib.rs](src/lib.rs)）：
   `anydoc_to_markdown*`、`anydoc_to_document`、格式检测，以及配套的
   `_free` 释放函数。构建脚本将 anydoc 内部私有的 `document_to_markdown`
   序列化器重新导出，从而保留官方 GFM 渲染能力。
2. **每个平台一个静态库** —— `crate-type = ["staticlib"]` 构建出
   `libanydoc_go.a`（约 20 MB）。六个静态库提交在 [lib/](lib/) 目录下。
3. **带构建标签的平台文件** —— 每个平台一个 Go 文件（
   [lib_linux_amd64.go](lib_linux_amd64.go) 等），由
   `//go:build linux && amd64 && cgo` 选中，并在其中通过 `#cgo LDFLAGS`
   指向对应静态库：
   `-L${SRCDIR}/lib/linux_amd64 -lanydoc_go -lm -lstdc++ -ldl -lpthread`。
4. **`//go:embed` 嵌入** —— 每个平台文件通过
   `//go:embed lib/linux_amd64/libanydoc_go.a` 嵌入其静态库。该指令使静态库
   成为**包的显式依赖**，`go mod vendor`、模块 zip 和 `go list` 都不会静默
   丢弃它。字节可通过 `EmbeddedLib()`/`ExtractLib()` 获取；不调用它们的
   二进制文件没有任何开销 —— 链接器会剔除未引用的约 20 MB 数据。
5. **对用户友好的 Go API** —— [anydoc.go](anydoc.go) 把所有指针和释放调用
   隐藏在普通 Go 函数之后；ABI 调用期间锁定 goroutine（错误槽是线程局部的），
   并把文档缓冲区边界安全地解码为 Go 结构体。

## 构建标签与兜底实现

| 构建方式                                       | 参与编译的文件                        | 行为                                              |
| ---------------------------------------------- | ------------------------------------- | ------------------------------------------------- |
| cgo + 受支持平台                               | 平台文件 + [anydoc.go](anydoc.go)     | 链接静态库，功能完整                              |
| cgo + 受支持平台 + `-tags dynamic`             | 动态平台文件 + [anydoc.go](anydoc.go) | 链接从 release 下载的共享库                       |
| `CGO_ENABLED=0` 或不受支持的平台               | [anydoc_stub.go](anydoc_stub.go)      | 可编译；所有函数返回提示明确的 `UnavailableError` |

```go
md, err := anydoc.ToMarkdownBytes(data, nil)
if err != nil && anydoc.IsUnavailable(err) {
	// 回退到其他转换器，或把原因告知运维
	log.Printf("anydoc unavailable: %v", err)
}
```

兜底实现保证模块在任何环境都可导入 —— 构建永远不会因为这个包而失败，
只会在运行时以清晰的说明快速报错。

## vendoring

`go mod vendor` 会一并复制静态库和 C 头文件：`//go:embed` 指令将其声明为
包的依赖，因此使用 `-mod=vendor` 的构建依然有效 —— `#cgo LDFLAGS` 中的
路径以 `${SRCDIR}` 为基准，在 vendor 目录内同样能正确解析。请在默认的
cgo 开启环境下执行 `go mod vendor`：它会捕获全部六个 `lib/<platform>/`
目录（所有平台，而非仅当前主机）。以 `CGO_ENABLED=0` 执行 vendor 只捕获
头文件 —— 兜底构建也只需要这些。

注意模块体积：六个静态库合计约 120 MB，`go get` 会拉取较大的模块 zip
（远低于代理的 500 MB 上限，但值得知晓）。如果注册表或带宽对此敏感，
一次性 vendor 是常见做法 —— 或者参考下面的动态库方案。

## 动态库（可选）

默认情况下模块链接内嵌的静态归档。使用 `dynamic` 构建标签则改为链接
共享库：

```
go build -tags dynamic ./...
```

模块本身不打包任何动态库 —— 共享库由 **Build dynamic libraries** 工作流
构建，发布到 [releases 页面](https://github.com/nbbug/anydoc-go/releases)，
每个平台一个 `anydoc-dynlib-<platform>.tar.gz`（共享库 + C 头文件）。下载
对应平台的包，让它进入 cgo 的搜索路径：

- 解压到任意目录，构建时用 `CGO_LDFLAGS` 指定平台目录（cgo 会把这些
  flags 追加到包自身 flags 之后）：
  ```
  tar -xzf anydoc-dynlib-darwin_arm64.tar.gz -C ~/anydoc-dynlib
  CGO_LDFLAGS="-L$HOME/anydoc-dynlib/darwin_arm64" go build -tags dynamic ./...
  ```
- 或者 vendor 模块（`go mod vendor`），把解压出的文件放进
  `vendor/github.com/nbbug/anydoc-go/dynlib/<platform>/`；动态平台文件
  本身就搜索 `${SRCDIR}/dynlib/<platform>`。

运行时操作系统要能找到库：Linux 设置 `LD_LIBRARY_PATH`、macOS 设置
`DYLD_LIBRARY_PATH`，Windows 把 DLL 放在可执行文件旁或加入 `PATH`。

取舍：动态构建让二进制更小、多个二进制可共享一份库，但部署时必须携带
库文件，且二进制的质量取决于你下载的库。除非模块体积成为瓶颈，否则
保持默认的静态链接。

## 从源码构建静态库

只有升级 anydoc、修改 C ABI 或支持新平台时才需要；模块使用者无需关心。

要求：Rust（rustup，任意较新的稳定版 —— anydoc 0.2.4 需要 ≥ 1.88），外加
一种交叉编译工具链：`cross`（需要 Docker）或 `cargo-zigbuild`（自带 zig，
无需 Docker）。

```bash
# 当前机器能构建的全部平台。
./scripts/build-all.sh

# 按 lib 目录名选择子集。
./scripts/build-all.sh linux_amd64 darwin_arm64

# 直接构建单个目标并指定驱动：
TARGET=aarch64-unknown-linux-gnu BUILD_TOOL=cross ./scripts/build-anydoc-lib.sh
TARGET=x86_64-pc-windows-msvc ./scripts/build-anydoc-lib.sh   # 仅限 Windows 主机
```

[scripts/build-all.sh](scripts/build-all.sh) 优先使用 `cross`，其次回退到
`cargo-zigbuild`，最后使用原生 cargo。macOS 静态库需在 Mac 上原生构建
（Apple SDK 不可再分发，Linux 主机无法构建 —— CI 使用 macOS 运行器）；
Windows 静态库用 MSVC 工具链在原生 Windows 上构建（CI 使用 Windows
运行器），其他主机上自动跳过。脚本会锁定精确的 `anydoc` 版本；当
[version.go](version.go) 与 Cargo.toml 的锁定版本不一致时拒绝构建；
[Cargo.lock](Cargo.lock) 存在时使用 `--locked` 构建。

### 在 CI 中重新构建

**Build static libraries** 工作流
（[.github/workflows/build-libs.yml](.github/workflows/build-libs.yml)）
会构建全部六个静态库，先通过 Go 包实际链接并转换做冒烟测试，然后把产物
提交并推送到主分支：

**Actions → “Build static libraries” → Run workflow。**

在修改 `anydoc = "=X.Y.Z"` 锁定版本、升级工具链、或仓库中的静态库过期时
运行它。另一个 **CI** 工作流在每次 push 和 pull request 时执行 gofmt 检查、
版本锁定一致性检查、无 cgo 模式测试，以及（静态库已存在后）完整的 cgo
测试套件。

## Wasm 与本绑定的对比

anydoc 官方也提供 [WebAssembly 绑定](https://github.com/firecrawl/anydoc/tree/main/wasm)
（`@firecrawl/anydoc-wasm`）。本静态库绑定存在的原因在于：Go 原生服务从
wasm 运行时中得不到任何好处。

**相对 wasm 的优势**

- **使用者无需 Rust 或 wasm 工具链。**预编译静态库随 `go get` 直接到达；
  在 Go 中使用 wasm 需要 [wazero](https://github.com/tetratelabs/wazero)
  等运行时，还要额外构建 `.wasm` 产物。
- **原生性能。**转换以本机机器码在进程内执行，没有 wasm 解释或 JIT 预热
  开销（实测通常快 1.5–3 倍）。
- **完整的文件系统 API。**只有本绑定提供 `ToMarkdown(path)` —— wasm
  没有文件系统。
- **并发。**原生线程与 Go 调度器；wasm 模块在调用线程上单线程执行，
  繁忙的服务必须把转换任务搬运到 worker。
- **少一个依赖。**静态库直接链入二进制，无需维护与 Go 模块版本同步的
  运行时库。

**相对 wasm 的劣势**

- **必须启用 cgo。**无法用于 `CGO_ENABLED=0` 的纯 Go 构建，每个平台都需要
  C 工具链；部分 PaaS/buildpack 场景和静态分析工具对 cgo 不友好。
- **每个平台一份静态库。**六个文件 × 约 20 MB 存放在模块中；wasm 是单一
  可移植产物。链接后的二进制也会增大相应体积。
- **交叉编译更麻烦。**单台主机无法产出全部平台的静态库（macOS 目标必须
  在 Mac 上构建，其余依赖 cross/zig）；wasm 在任何地方都能构建出到处
  可用的产物。
- **无法用于浏览器或边缘计算。**wasm 可在浏览器和 serverless 边缘平台
  运行；原生静态库不行。
- **依赖 libc。**Linux 静态库面向 glibc；wasm 与 libc 无关。

**选择建议：**在服务端转换文档的 Go 服务中使用本模块；浏览器/边缘工具、
小型 CLI 一次性任务、或纯 Go 部署流水线，更适合 wasm 绑定。

## 安全说明

anydoc 在你的进程内解析不受信任的恶意输入：

- Rust ABI 捕获 panic 并将其报告为 `malformed` 错误，而不是让进程崩溃
  （见 [src/lib.rs](src/lib.rs) 的 `guarded`）。但无法阻止栈溢出或
  分配失败导致的终止。
- 锁定的 anydoc 版本包含 pdf-inspector 针对此前无上界的 PDF 解析的修复
  （见 anydoc changelog）—— 升级锁定版本时应使用 `--locked` 重新构建，
  避免随意引入未经审计的版本。
- 开启 `OcrHosted` 后，报 `needs_ocr` 的 PDF 会离开你的进程并上传到
  Firecrawl Parse（除非设置 `FIRECRAWL_API_KEY`，否则免密钥）。处理敏感
  文档前请先了解其服务条款；默认行为不会向任何外部服务发送任何内容。
- 与任何原生解析器一样，处理不受信任上传的进程应做好沙箱或资源限制。

## 许可证

MIT。本模块链接 `anydoc` crate（MIT © Sideguide Technologies Inc.）。
详见 [LICENSE](LICENSE)。

## 致谢

绑定架构参考了上游 Go 绑定（[firecrawl/anydoc#30](https://github.com/firecrawl/anydoc/pull/30)）
以及 [WeKnora](https://github.com/Tencent/WeKnora) 内置的加固版本
（`third_party/anydoc-go`），包括 cgo 线程锁定、panic 防护、文档模型的
扁平序列化，以及构建脚本对 crate 源码的补丁方式。
