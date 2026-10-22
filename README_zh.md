# otty-snippets-view

[English](README.md)

Otty 详情面板里的一个视图。用来管理 Otty 的文本 snippet（`~/.config/otty/recipes/*.ottyrecipe`）和它们的分组，并把 snippet 发送到当前聚焦的终端 pane。

## 安装

一行命令装上最新 release，并注册成详情面板的视图。以后再跑同一行就是更新：

```sh
curl -fsSL https://raw.githubusercontent.com/elonnzhang/otty-snippets-view/main/install.sh | sh
```

脚本会把二进制放到 `~/.config/otty/views/otty-snippets-view/`，把 `details-view` 写进 `~/.config/otty/config.toml`（保留原来的 `--lang`），reload 配置，并停掉正在运行的旧视图，下次按 ⌘⇧R 打开就是新版。旧二进制留作 `.bak`，装上的 tag 记在 `.version` 里；如果已经是这个版本就直接跳过。它不会替你打开 `ipc-allow-send-keys`，只在你没开时提醒一句；遇到属于别的工具的 `details-view` 会拒绝，除非用 `FORCE=1` 启动脚本。

| 参数 | 说明 |
| --- | --- |
| `--check` | 只打印已装版本和最新版本 |
| `--dir <path>` | 安装目录，默认 `~/.config/otty/views/otty-snippets-view` |
| `--version <tag>` | 装指定 release，而不是最新 |
| `--from <binary>` | 装本地构建，不下载；`--from dev` 直接构建当前源码目录 |
| `--force` | 同版本也重装（管道方式用 `FORCE=1 sh`） |
| `--no-config` | 只放二进制，不动 Otty 配置 |
| `--no-restart` | 不动正在运行的视图（它会继续用旧版本，直到重开面板） |
| `--config-file <p>` | 读写指定的配置文件，而不是 Otty 自己的 |

`| sh` 之后的参数要写成 `sh -s --`，例如 `curl -fsSL …/install.sh | sh -s -- --check`。

最新 tag 从 `releases/latest` 的跳转地址里取，所以既不需要 `gh` 也不需要 GitHub token，也不会撞 API 限流。

在源码目录里用 `./release.sh` 走开发流程：vet、测试，然后构建并安装当前工作区（即 `sh install.sh --from dev`）；`--fast` 跳过 vet/测试，`--zip` 顺带打 release 用的 zip。单独跑 `sh install.sh --from dev` 就是去掉这两项检查的同一条路，而且面板开着也能用——每次安装的 command 都会变，Otty 因此会启动新构建。Herdr 插件不用管，它的 `run.sh` 发现有更新的 `.go` 文件时会自己重建。

Otty 重载配置不会重启正在运行的视图，所以脚本每次都会改写 command 并停掉旧进程：面板开着时会直接换成新构建，否则重开面板（按两次 ⌘⇧R）即可用上。

## 参数

| 参数 | 说明 |
| --- | --- |
| `--lang en\|cn` | 界面语言，默认英文。也可写 `--lang=cn`，`zh`、`zh-CN`、`中文` 同样认 |
| `SNIPPETS_LANG` | 作用同 `--lang`，命令行参数优先 |
| `OTTY_RECIPES_DIR` | recipe 目录，默认 `~/.config/otty/recipes` |
| `SNIPPETS_HOME` | `groups.json` 和 `trash/` 所在目录，默认 `~/.config/otty/views/otty-snippets-view` |

要中文界面，把 command 写成 `command=".../otty-snippets-view --lang cn"`。

## 界面

从上到下：

1. Tab 栏：「列表」是平铺的，每行带组名；「分组」按组成树，可折叠，用来管理分组。按 Tab、`1`/`2` 或鼠标点击切换。
2. Snippet 列表。
3. 详情区：显示名称、alias、所属组，以及带语法高亮的完整内容。语言自动识别，认不出就按 shell 处理。用鼠标拖上边框（`── 详情 ═══ ──`）调高度，`+` / `-` 也行。
4. 快捷键提示，两行：键在上，说明在正下方。状态消息和确认框显示在提示区上面那一行，键始终可见。

当前 Tab 和详情区高度会保存在 `groups.json`。

## 快捷键

| 键 | 作用 |
| --- | --- |
| `⏎` / 双击 | 在聚焦的 pane 里执行（在组上是折叠） |
| `o` | 只填入，不执行 |
| 右键 | 右键菜单：snippet 上是填入 / 编辑 / 移动 / 删除 / 新建；分组上是新建子分组 / 重命名 / 移动 / 删除；空白处是新建分组 / snippet |
| `y` | 复制到剪贴板 |
| `a` | 新建 snippet（在「分组」里会建到当前组） |
| `e` | 编辑 snippet；在组上是重命名 |
| `m` | 用菜单移到某个组；在分组上是把分组移到别的组下 |
| `c` | 设置选中 snippet 或分组的颜色（`1`–`6` 选色，`0` 清除），右键菜单里也有 |
| `M` | 在选中的分组下新建子分组 |
| 拖动 | 把 snippet 拖到组上（「分组」里可用） |
| `d` / `x` / `Delete` | 删除，文件移入 `trash/` |
| `n` / `r` | 新建分组 / 重命名分组 |
| `[` / `]` | 调整分组顺序 |
| `/` | 搜索，`Esc` 清除 |
| `R` / `⌃R` | 从磁盘重新加载 |
| `↑↓` `jk` `PgUp/PgDn` `g/G` `←→` | 移动；左右键也能折叠、展开 |

表单里：`Tab` / `⇧Tab` 切换字段，`⌃S` 保存，`Esc` 取消。alias 必填，不能有空格。

snippet 内容里的占位符：`{{clipboard}}` `{{date}}` `{{time}}`。`{{cursor}}` 只在 Otty 自己的展开器里有用，这里会被去掉。

## 分组规则

- 名称写成 `[组名]-名字` 会自动归入这个组。组不存在就自动新建，匹配不区分大小写。在组里显示时只显示 `名字`。
- 手动移动会覆盖前缀分组。移回前缀对应的组时，手动设置自动取消。
- 删除分组后，组里的 snippet 回到未分组，不会因为名称前缀又把组建回来。
- 重命名分组时，靠前缀归进来的成员仍然留在组里。
- 「未分组」只是显示名，中英文下都不能用这个名字建组。

### 多级分组

- 分组是一棵树。选中分组按 `M` 在它下面新建子分组，按 `m` 把它移到别的分组下（或移回顶层）。在「分组」tab 里也可以把一个分组标题拖到另一个分组上。
- 分组不能移到自己或自己的子分组下面。
- 折叠一个分组时，它的子分组和子分组里的 snippet 一起收起。
- `[` / `]` 只在该分组的同级之间调整顺序。
- 删除分组会连同它的子分组一起删除，里面的 snippet 都回到未分组。

## Herdr 插件

同一个二进制也可以作为 Herdr 的插件面板运行，把 snippet 显示在你正在用的窗格旁边。先安装一次插件（安装时需要 Go，清单里会执行构建）：

```sh
herdr plugin install elonnzhang/otty-snippets-view/herdr-plugin
```

然后在 `~/.config/herdr/config.toml` 里绑定快捷键：

```toml
[[keys.command]]
key = "prefix+shift+s"
type = "plugin_action"
command = "local.otty-snippets.open"
description = "snippets"
```

`⏎` 会把 snippet 输入到打开侧边栏时所在的窗格（来自 Herdr 的调用上下文，不会输入到侧边栏自己），`o` 只填入、不回车。`SNIPPETS_TARGET_PANE` 可以指定目标窗格。snippet 和分组与 Otty 视图用的是同一批文件，两边同步。

Herdr 没有在自己侧边栏内放 widget 的接口，所以插件打开的是一个分屏窗格——这是它提供的最接近的放置方式。详见 [herdr-plugin/README.md](herdr-plugin/README.md)。

## 文件

- `~/.config/otty/recipes/*.ottyrecipe`：snippet 本身，跟 Otty 设置里 Recipes 编辑的是同一批文件。写入是原子的，格式和 Otty 保持一致。含未知字段的文件不会被改写，请到 Otty 里编辑。
- `groups.json`：分组、手动分配（`"none"` 表示固定在未分组）、颜色和界面状态。文件损坏时会被改名为 `groups.json.bad`。
- `trash/`：删掉的 recipe，文件名是 `<时间戳>-<id>.ottyrecipe`。

在 Otty 的 Recipes 设置里改了 snippet，一秒内这里就会更新。

## 开发

```sh
go vet ./... && go test ./...
go build -o otty-snippets-view .
```

打成通用版给同事（macOS，arm64 + x86_64）：

```sh
mkdir -p dist
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/arm64 .
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/amd64 .
lipo -create dist/arm64 dist/amd64 -output dist/otty-snippets-view
rm dist/arm64 dist/amd64
ditto -c -k --keepParent dist/otty-snippets-view dist/otty-snippets-view.zip
```

`-s -w` 去掉符号表和调试信息，体积从 21 MB 降到 16 MB，zip 约 5.4 MB。

体积主要来自语法高亮的 chroma：它的 `lexers` 包内嵌了全部 296 个语言定义（单独就占约 5 MB），只要 import 一次就会全部带上。要只带几个语言，得自己维护一份小的 `embed.FS` 和注册表，代价是没法自动识别语言，所以保持现状。

`main.go` 里的 `noquery/` 必须保留。Bubble Tea 的 `init` 会查询终端背景色，最长等 5 秒，不回答的终端在这段时间面板是空白的。`noquery` 提前给出答案。module 改成标准路径 `github.com/…/otty-snippets-view` 后，import 顺序不再能保证谁先跑；渲染器会保留先拿到的那个答案，所以它只是把赢面放到我们这边。

代码结构：`store.go`（recipe 和元数据文件）、`meta.go`（分组规则、与 Otty 交互）、`model.go`（状态、行、布局）、`update.go`（键盘、鼠标）、`actions.go`、`view.go` / `detail.go`（渲染）、`highlight.go`、`form.go`、`i18n.go`（全部界面文字）。`dist/` 放分发用的构建产物，不是源码。

## 分发

同事直接用[安装](#安装)里那一行命令：它会从 [latest release](https://github.com/elonnzhang/otty-snippets-view/releases/latest) 拉通用版二进制（arm64 + x86_64）并自己写好配置，以后再跑同一行就是更新。snippet 数据不打包在里面：读的是各自的 `~/.config/otty/recipes`，分组文件 `groups.json` 会新建在 `~/.config/otty/views/otty-snippets-view` 下。要把 snippet 发送到 pane 需要 `ipc-allow-send-keys = true`，这个开关脚本留给他自己决定。

想手动安装的话，从 release 下载 `otty-snippets-view.zip`，解压后加上这一行：

```toml
details-view = 'id="snippets" kind="tui" title="Snippets" icon="file" command="/path/to/otty-snippets-view" mobile="false"'
```

Agent 可以按 [SETUP-PROMPT.md](SETUP-PROMPT.md) 里的 prompt 自动完成安装。第一次运行如果被 Gatekeeper 拦下，右键 → 打开，或者执行 `xattr -d com.apple.quarantine otty-snippets-view`。
