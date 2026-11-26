**[English](README.md) · 中文文档**

# anydoc-go

[anydoc](https://github.com/firecrawl/anydoc) 的 Go 语言绑定 —— anydoc 是一个
Rust 库，可将 Word、PowerPoint、Excel、OpenDocument、RTF、EPUB、CSV 和 PDF
等文档转换为 GitHub Flavored Markdown。

anydoc-go 通过 cgo 以**共享库**的方式链接 anydoc。库文件不打包进 Go 模块：
**Build libraries** 工作流会为每个平台发布对应的压缩包，下载后让 cgo
指向它即可；库在运行时加载，因此需要随程序一起分发。使用者**无需安装
Rust 或任何额外工具链**。

## 环境要求

|          |                                                                                 |
| -------- | ------------------------------------------------------------------------------- |
| Go       | ≥ 1.22                                                                          |
| CGO      | `CGO_ENABLED=1`（所有支持平台上的 Go 默认值）                                   |
| C 工具链 | 各系统标准配置 —— macOS 用 Xcode 命令行工具、Linux 用 gcc、Windows 用 mingw-w64 |
| Rust     | 使用本模块**不需要**；只有从源码构建库时才需要                                  |

## 安装

```
go get github.com/nbbug/anydoc-go
```

`go get` 只会拉取 Go 包。请从
[libs-v0.2.5 release](#预编译库下载) 下载当前平台的动态库压缩包，解压后
通过 `CGO_LDFLAGS` 指向它的 `lib/<platform>` 目录，并在运行时把该目录
加入动态库搜索路径：

```
tar -xzf anydoc-dynlib-linux_amd64.tar.gz -C anydoc-lib
CGO_LDFLAGS="-L$(pwd)/anydoc-lib/lib/linux_amd64" go build ./...
LD_LIBRARY_PATH="$(pwd)/anydoc-lib/lib/linux_amd64" ./myapp
```

Windows 在 PowerShell 中用 `$env:CGO_LDFLAGS = "-L..."` 设置同样的值，
并把 `anydoc_go.dll` 放在可执行文件旁。

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

Go 编译器通过构建标签（build tags）为以下每个组合自动选择平台文件。请从
release 下载对应的共享库；它位于 `lib/<platform>/` 下：
`libanydoc_go.so`、`libanydoc_go.dylib`，Windows 则是 `anydoc_go.dll`
及其 import 库。

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
  `aarch64-pc-windows-msvc`），与 WeKnora 绑定一致。DLL 由 MSVC 工具链
  构建（MSVC 不可再分发，因此只能在原生 Windows 主机或 CI 的 Windows
  运行器上构建），并附带其 import 库。Go 的 cgo 仍然使用 mingw gcc
  链接 —— GNU ld 可以读取 MSVC 的 import 库 —— 因此使用者只需要标准的
  mingw-w64 工具链，不需要安装 Visual Studio。运行时 `anydoc_go.dll`
  需放在可执行文件旁或 `PATH` 中。
- **Linux 为 glibc。**Alpine/musl 用户需要自行重新编译共享库（在 Alpine
  镜像内执行 `./scripts/build-all.sh linux_amd64`，或使用
  `cargo-zigbuild`，再使用或发布该产物）。Go 的构建标签无法自动识别 musl，参考项目
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
// ParseMode（托管 OCR 的解析模式：ParseModeAuto 默认 / ParseModeFast /
// ParseModeOCR）、APIKey（缺省回退到 FIRECRAWL_API_KEY，再回退到免密钥）、
// APIURL（缺省回退到 FIRECRAWL_API_URL，再回退到 https://api.firecrawl.dev）。

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

// 模块版本，与锁定的 anydoc crate 版本同步。
const Version = "0.2.5"
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

`Options.ParseMode` 用于选择 anydoc 0.2.5 新增的 Parse PDF 模式：
`ParseModeAuto`（零值）先提取文本、对无文本层页面回退 OCR；
`ParseModeFast` 只提取文本；`ParseModeOCR` 对每一页都做 OCR。未知模式会在
上传任何内容之前报错：

```go
md, err := anydoc.ToMarkdownBytesWithOptions(pdf, &anydoc.FormatPdf, &anydoc.Options{
	Ocr:       anydoc.OcrHosted,
	ParseMode: anydoc.ParseModeOCR,
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
   `=0.2.5`），并暴露一个精简的 C ABI（[src/lib.rs](src/lib.rs)）：
   `anydoc_to_markdown*`、`anydoc_to_document`、格式检测，以及配套的
   `_free` 释放函数。构建脚本将 anydoc 内部私有的 `document_to_markdown`
   序列化器重新导出，从而保留官方 GFM 渲染能力。
2. **每个平台一个共享库** —— `crate-type = ["cdylib"]` 构建平台对应的共享库
   （`.so`、`.dylib`，Windows 为 `.dll` 及其 import 库）。**Build libraries**
   工作流会为每个平台发布该产物；它不会提交到模块中。
3. **带构建标签的平台文件** —— 每个平台一个 Go 文件（
   [lib_linux_amd64.go](lib_linux_amd64.go) 等），由
   `//go:build linux && amd64 && cgo` 选中，并在其中通过 `#cgo LDFLAGS`
   指定搜索路径：`-L${SRCDIR}/lib/linux_amd64 -lanydoc_go`。
   把下载的压缩包解压到对应目录即可被自动找到，否则通过 `CGO_LDFLAGS`
   提供路径；运行时还需要让加载器找到它。
4. **对用户友好的 Go API** —— [anydoc.go](anydoc.go) 把所有指针和释放调用
   隐藏在普通 Go 函数之后；ABI 调用期间锁定 goroutine（错误槽是线程局部的），
   并把文档缓冲区边界安全地解码为 Go 结构体。

## 构建标签与兜底实现

| 构建方式                         | 参与编译的文件                        | 行为                                              |
| -------------------------------- | ------------------------------------- | ------------------------------------------------- |
| cgo + 受支持平台                 | 平台文件 + [anydoc.go](anydoc.go)     | 链接下载的共享库，功能完整                        |
| `CGO_ENABLED=0` 或不受支持的平台 | [anydoc_stub.go](anydoc_stub.go)      | 可编译；所有函数返回提示明确的 `UnavailableError` |

```go
md, err := anydoc.ToMarkdownBytes(data, nil)
if err != nil && anydoc.IsUnavailable(err) {
	// 回退到其他转换器，或把原因告知运维
	log.Printf("anydoc unavailable: %v", err)
}
```

兜底实现保证模块在 cgo 不可用时仍可导入。在受支持平台且开启 cgo 时，缺少
下载的库会在链接阶段报 cgo 的常规错误；按上文下载即可提供该库。没有用于
切换实现的构建标签：cgo + 受支持平台总是链接共享库。

## vendoring

`go mod vendor` 会复制包源码和 [include/anydoc.h](include/anydoc.h)：
[vendor_files.go](vendor_files.go) 中未加构建标签的 `//go:embed` 会固定该
头文件。库文件不在模块中，因此不会被复制。执行 vendor 后，请把要构建的
平台压缩包解压到 vendored 模块内 —— 平台文件中以 `${SRCDIR}` 为基准的
`lib/<platform>` 搜索路径随后会在 vendor 目录内解析：

```
go mod vendor
tar -xzf anydoc-dynlib-linux_amd64.tar.gz -C vendor/github.com/nbbug/anydoc-go
go build -mod=vendor ./...
```

本地 checkout 或使用 `replace` 时，把压缩包解压到模块根目录即可。库放在
其他位置时可用 `CGO_LDFLAGS` 指定目录，运行时还要把它加入加载器搜索路径，
见[预编译库](#预编译库下载)。

## 预编译库下载

[libs-v0.2.5 release](https://github.com/nbbug/anydoc-go/releases/tag/libs-v0.2.5)
为每个支持的平台提供共享库，并附带
[`SHA256SUMS`](https://github.com/nbbug/anydoc-go/releases/download/libs-v0.2.5/SHA256SUMS)。
这些库都不在 Go 模块中，因此只需要下载自己所用平台的压缩包。

| 平台          | 资源                                                                                                                                           |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| linux/amd64   | [anydoc-dynlib-linux_amd64.tar.gz](https://github.com/nbbug/anydoc-go/releases/download/libs-v0.2.5/anydoc-dynlib-linux_amd64.tar.gz)           |
| linux/arm64   | [anydoc-dynlib-linux_arm64.tar.gz](https://github.com/nbbug/anydoc-go/releases/download/libs-v0.2.5/anydoc-dynlib-linux_arm64.tar.gz)           |
| darwin/amd64  | [anydoc-dynlib-darwin_amd64.tar.gz](https://github.com/nbbug/anydoc-go/releases/download/libs-v0.2.5/anydoc-dynlib-darwin_amd64.tar.gz)         |
| darwin/arm64  | [anydoc-dynlib-darwin_arm64.tar.gz](https://github.com/nbbug/anydoc-go/releases/download/libs-v0.2.5/anydoc-dynlib-darwin_arm64.tar.gz)         |
| windows/amd64 | [anydoc-dynlib-windows_amd64.tar.gz](https://github.com/nbbug/anydoc-go/releases/download/libs-v0.2.5/anydoc-dynlib-windows_amd64.tar.gz)       |
| windows/arm64 | [anydoc-dynlib-windows_arm64.tar.gz](https://github.com/nbbug/anydoc-go/releases/download/libs-v0.2.5/anydoc-dynlib-windows_arm64.tar.gz)       |

压缩包包含 `lib/<platform>/` —— `libanydoc_go.so`、`libanydoc_go.dylib`，
或 Windows 的 `anydoc_go.dll` 及其 import 库（`anydoc_go.dll.lib`，另外
复制一份 `libanydoc_go.dll.a` 供 mingw 使用）—— 以及匹配的 C 头文件
`include/anydoc.h`。

使用前请校验（macOS 使用 `shasum -a 256 -c SHA256SUMS`）：

```
sha256sum -c SHA256SUMS
```

### 链接

下载、解压，并通过 `CGO_LDFLAGS` 指定平台目录（cgo 会把这些 flags 追加到
包自身 flags 之后）：

```
tar -xzf anydoc-dynlib-linux_amd64.tar.gz -C anydoc-lib
CGO_LDFLAGS="-L$(pwd)/anydoc-lib/lib/linux_amd64" go build ./...
```

Windows 在 PowerShell 中设置同样的 `-L.../windows_amd64`：
`$env:CGO_LDFLAGS = "-L..."`。如果把压缩包解压到 checkout 或 vendored 模块的
`lib/<platform>/`，平台文件中的 `#cgo LDFLAGS` 会自动找到它，无需设置
`CGO_LDFLAGS`：

```
tar -xzf anydoc-dynlib-linux_amd64.tar.gz -C vendor/github.com/nbbug/anydoc-go
go build -mod=vendor ./...
```

共享库包含完整的 C ABI，因此也可以供 C、C++ 或其他语言绑定直接通过
[include/anydoc.h](include/anydoc.h) 调用。

### 运行

程序启动时加载该库。把压缩包解压到模块的 `lib/<platform>/` 时，链接行会
记录指向该目录的 rpath，因此在那里构建出的二进制无需任何环境变量就能找到
库。若要把库随程序发布到别处，则需要让加载器能找到它：

- Linux：`LD_LIBRARY_PATH`，或把库安装到标准的加载器路径；
- macOS：`DYLD_LIBRARY_PATH` —— 注意 dyld 会在 `go run`、`go test` 启动
  之前就丢掉它，因此自定义目录时要么把库装到标准路径，要么直接运行构建
  出来的二进制；
- Windows：把 `anydoc_go.dll` 放在可执行文件旁或加入 `PATH`。

```
LD_LIBRARY_PATH="$(pwd)/anydoc-lib/lib/linux_amd64" ./myapp
```

取舍：二进制体积更小，多个二进制可以共享一份库，但每次部署都必须携带
库文件，而且程序的表现取决于你随附的库。

## 从源码构建库

只有升级 anydoc、修改 C ABI 或支持新平台时才需要；模块使用者无需关心。

要求：Rust（rustup，任意较新的稳定版 —— anydoc 0.2.5 需要 ≥ 1.88），外加
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
`cargo-zigbuild`，最后使用原生 cargo。macOS 库需在 Mac 上原生构建
（Apple SDK 不可再分发，Linux 主机无法构建 —— CI 使用 macOS 运行器）；
Windows 库用 MSVC 工具链在原生 Windows 上构建（CI 使用 Windows
运行器），其他主机上自动跳过。脚本会锁定精确的 `anydoc` 版本；当
[version.go](version.go) 与 Cargo.toml 的锁定版本不一致时拒绝构建；
[Cargo.lock](Cargo.lock) 存在时使用 `--locked` 构建。

构建产物写入 `lib/<platform>/`，并在完成前与 C 头文件核对：头文件声明的
每个 `anydoc_*` 函数都必须被导出。Linux 和 macOS 用 `nm` 读取动态符号表
完成校验；Windows 上只检查 DLL 与 import 库是否存在且非空（运行器上没有
能同时读取两种架构的 `nm`），导出符号由 rustc 的 `#[no_mangle]` 属性保证。

共享库无需任何后处理：链接器会丢弃 Rust 预编译 std 中嵌入的 LTO IR
（`.llvmbc`/`.llvmcmd` 段被标记为可丢弃 —— ELF 是 `SHF_EXCLUDE`，COFF 是
`IMAGE_SCN_LNK_REMOVE`，ld64 则丢弃 Mach-O 的 `__LLVM` 段），因此发布的
`.so`/`.dylib`/`.dll` 里不含这些内容。

### 在 CI 中重新构建

**Build libraries** 工作流
（[.github/workflows/build-libs.yml](.github/workflows/build-libs.yml)）
会在一次运行中为全部六个平台构建共享库，用打包后的 linux/amd64 压缩包
跑一遍 Go 测试套件做冒烟测试，最后把 6 个压缩包和 `SHA256SUMS` 发布到
`libs-v0.2.5` release（由 `release_tag` 输入指定；重复运行会更新同一个
release）。它不会向仓库提交任何内容。

**Actions → “Build libraries” → Run workflow。**

在修改 `anydoc = "=X.Y.Z"` 锁定版本、C ABI、平台矩阵或构建工具时运行它。
**CI** 工作流在每次 push 和 pull request 时执行 gofmt 检查、版本锁定一致性
检查和无 cgo 模式测试；它还会从源码构建 linux/amd64 共享库，用 Go 1.27.4
（cgo 检查会拒绝 LLVM LTO IR 的版本）链接一次 —— 一旦回归，失败会出现在
CI 而不是使用者的 `go build` —— 然后运行完整的 cgo 测试套件。

## Wasm 与本绑定的对比

anydoc 官方也提供 [WebAssembly 绑定](https://github.com/firecrawl/anydoc/tree/main/wasm)
（`@firecrawl/anydoc-wasm`）。本原生绑定存在的原因在于：Go 原生服务从
wasm 运行时中得不到任何好处。

**相对 wasm 的优势**

- **使用者无需 Rust 或 wasm 工具链。**预编译库直接从 release 下载；
  在 Go 中使用 wasm 需要 [wazero](https://github.com/tetratelabs/wazero)
  等运行时，还要额外构建 `.wasm` 产物。
- **原生性能。**转换以本机机器码在进程内执行，没有 wasm 解释或 JIT 预热
  开销（实测通常快 1.5–3 倍）。
- **完整的文件系统 API。**只有本绑定提供 `ToMarkdown(path)` —— wasm
  没有文件系统。
- **并发。**原生线程与 Go 调度器；wasm 模块在调用线程上单线程执行，
  繁忙的服务必须把转换任务搬运到 worker。

**相对 wasm 的劣势**

- **必须启用 cgo。**无法用于 `CGO_ENABLED=0` 的纯 Go 构建，每个平台都需要
  C 工具链；部分 PaaS/buildpack 场景和静态分析工具对 cgo 不友好。
- **每个平台一份运行时库。**六个平台需要分别下载；wasm 是单一可移植产物。
  每次部署还必须把共享库随二进制一起分发。
- **需要手动下载库。**只执行 `go get` 还不够：每个构建目标都需要对应的
  release 压缩包（或自行构建的库）。wasm 则是自包含的。
- **交叉编译更麻烦。**单台主机无法产出全部平台的库（macOS 目标必须
  在 Mac 上构建，其余依赖 cross/zig）；wasm 在任何地方都能构建出到处
  可用的产物。
- **无法用于浏览器或边缘计算。**wasm 可在浏览器和 serverless 边缘平台
  运行；原生共享库不行。
- **依赖 libc。**Linux 库面向 glibc；wasm 与 libc 无关。

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
- anydoc 0.2.5 将 ZIP 系列格式升级到 `zip` 9.x（至少 9.5.1；当前
  lockfile 锁定 9.7.3），修复了影响所有 8.x 版本的五个安全公告
  （CVE-2026-115905 至 CVE-2026-115909）。release 库产物也基于该
  锁定版本重新构建。
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
