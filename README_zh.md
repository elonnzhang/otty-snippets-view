# otty-snippets-view

[English](README.md)

Otty 详情面板里的一个视图。用来管理 Otty 的文本 snippet（`~/.config/otty/recipes/*.ottyrecipe`）和它们的分组，并把 snippet 发送到当前聚焦的终端 pane。

## 安装

编译：

```sh
cd ~/.config/otty/views/otty-snippets-view
go build -o otty-snippets-view .
```

在 `~/.config/otty/config.toml` 里注册视图：

```toml
details-view = 'id="snippets" kind="tui" title="Snippets" icon="file" command="/Users/elon/.config/otty/views/otty-snippets-view/otty-snippets-view" mobile="false"'
```

值必须写成带引号的 `key="value"`，写 JSON 会被拒绝。改完执行 `otty config reload`，再把详情面板关掉重开（按两次 ⌘⇧R）。重载配置不会重启正在运行的视图。

要把 snippet 发到 pane，配置里需要 `ipc-allow-send-keys = true`。没开的话，执行或填入会报「被拒绝 (检查 ipc-allow-send-keys)」。

配置直接指向二进制。改了 Go 代码后重新编译，再 reload 配置（或重开面板）即可。

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
| `o` / 右键 | 只填入，不执行 |
| `y` | 复制到剪贴板 |
| `a` | 新建 snippet（在「分组」里会建到当前组） |
| `e` | 编辑 snippet；在组上是重命名 |
| `m` | 用菜单移到某个组；在分组上是把分组移到别的组下 |
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

## 文件

- `~/.config/otty/recipes/*.ottyrecipe`：snippet 本身，跟 Otty 设置里 Recipes 编辑的是同一批文件。写入是原子的，格式和 Otty 保持一致。含未知字段的文件不会被改写，请到 Otty 里编辑。
- `groups.json`：分组、手动分配（`"none"` 表示固定在未分组）和界面状态。文件损坏时会被改名为 `groups.json.bad`。
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

`main.go` 里的 `noquery/` 必须保留为第一个 import。Bubble Tea 的 `init` 会查询终端背景色，最长等 5 秒，这段时间面板是空白的。`noquery` 提前给出了答案；module 起名 `a/otty-snippets-view`，就是为了让它的 import 路径排在 `bubbletea` 前面。

代码结构：`store.go`（recipe 和元数据文件）、`meta.go`（分组规则、与 Otty 交互）、`model.go`（状态、行、布局）、`update.go`（键盘、鼠标）、`actions.go`、`view.go` / `detail.go`（渲染）、`highlight.go`、`form.go`、`i18n.go`（全部界面文字）。`dist/` 放分发用的构建产物，不是源码。

## 分发

`dist/otty-snippets-view.zip` 里是通用版二进制（arm64 + x86_64）。`dist/` 不进 git，按「开发」里的命令自己打，或者从 release 下载。同事只需要这个二进制和一行配置：

```toml
details-view = 'id="snippets" kind="tui" title="Snippets" icon="file" command="/path/to/otty-snippets-view" mobile="false"'
```

Agent 可以按 [SETUP-PROMPT.md](SETUP-PROMPT.md) 里的 prompt 自动完成安装。然后 `otty config reload`，再重开详情面板。第一次运行如果被 Gatekeeper 拦下，右键 → 打开，或者执行 `xattr -d com.apple.quarantine otty-snippets-view`。snippet 数据不打包在里面：读的是各自的 `~/.config/otty/recipes`，分组文件 `groups.json` 会新建在 `~/.config/otty/views/otty-snippets-view` 下。要把 snippet 发送到 pane，还需要 `ipc-allow-send-keys = true`。
