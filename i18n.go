package main

import (
	"fmt"
	"os"
	"strings"
)

// lang is the UI language, "en" by default; --lang en|cn (or SNIPPETS_LANG) picks it.
var lang = "en"

func pick(en, zh string) string {
	if lang == "zh" {
		return zh
	}
	return en
}

// parseLang accepts the spellings people actually type for the two languages.
func parseLang(v string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "en", "en-us", "english":
		return "en", true
	case "zh", "cn", "zh-cn", "chinese", "中文":
		return "zh", true
	}
	return "", false
}

// applyLang reads --lang/-lang (also --lang=cn) and SNIPPETS_LANG, in that order.
func applyLang(args []string) {
	for i, a := range args {
		for _, p := range []string{"--lang", "-lang"} {
			var v string
			switch {
			case a == p && i+1 < len(args):
				v = args[i+1]
			case strings.HasPrefix(a, p+"="):
				v = a[len(p)+1:]
			default:
				continue
			}
			l, ok := parseLang(v)
			if !ok {
				fmt.Fprintf(os.Stderr, "otty-snippets-view: unsupported --lang %q (use en or cn)\n", v)
				os.Exit(2)
			}
			lang = l
			return
		}
	}
	if l, ok := parseLang(os.Getenv("SNIPPETS_LANG")); ok {
		lang = l
	}
}

// tr looks up a UI string; dict holds {English, Chinese} pairs.
func tr(k string) string {
	p, ok := dict[k]
	if !ok {
		return k
	}
	return pick(p[0], p[1])
}

var dict = map[string][2]string{
	"h.input": {"input", "输入"}, "h.filter": {"filter", "过滤"},
	"h.select": {"select", "选择"}, "h.done": {"done", "完成"}, "h.clear": {"clear", "清除"},
	"h.save": {"save", "保存"}, "h.cancel": {"cancel", "取消"}, "h.move": {"move", "移到"},
	"h.ok": {"ok", "确认"}, "h.next": {"Next Tab", "下一项"},
	"h.fold": {"fold", "折叠"}, "h.newg": {"new g", "新组"}, "h.rename": {"rename", "重命名"},
	"h.del": {"del", "删除"}, "h.sort": {"sort", "排序"}, "h.tab": {"tab", "切换"},
	"h.moveg":  {"under", "移到组下"},
	"h.subg":   {"new sub", "子分组"},
	"h.search": {"search", "搜索"}, "h.find": {"find", "搜索"},
	"h.run": {"run", "执行"}, "h.fill": {"input", "填入"}, "h.copy": {"copy", "复制"},
	"h.new": {"new", "新建"}, "h.edit": {"edit", "编辑"}, "h.delete": {"del", "删除"},

	"tab.list":  {"List", "列表"},
	"tab.group": {"Groups", "分组"},

	"ungrouped":   {"Ungrouped", "未分组"},
	"count":       {"%d snippets · %d groups", "%d snippet · %d 组"},
	"group.count": {"%d snippets", "%d 个 snippet"},
	"group.auto":  {"Snippets named [%s]-… join this group automatically", "名称以 [%s]- 开头的 snippet 会自动归入此组"},
	"divider":     {" detail ", " 详情 "},

	"menu.title":         {`Move "%s" to:`, "移动「%s」到:"},
	"menu.group.title":   {`Move group "%s" under:`, "移动分组「%s」到:"},
	"menu.no.parent":     {"No parent (top level)", "不放在组下（顶层）"},
	"menu.move.group":    {"move group", "移到组下"},
	"pick.group":         {"select a group first", "先选中一个分组"},
	"pick.snip":          {"pick a snippet, not a group", "请选中 snippet，不是分组"},
	"group.no.move":      {"Ungrouped has no place in the tree", "未分组不能放进组里"},
	"pick.cycle":         {"a group can't go under itself or its children", "分组不能移到它自己或子分组下面"},
	"moved.group":        {`✓ group "%s" is now under "%s"`, "✓ 分组「%s」已移到「%s」下"},
	"moved.top":          {`✓ group "%s" is back at the top level`, "✓ 分组「%s」已回到顶层"},
	"group.parent":       {"in %s", "位于 %s"},
	"group.del.tree":     {"Delete group \"%s\"? %d snippets move to Ungrouped, and its %d sub-groups go too", "删除分组「%s」? %d 个 snippet 回到未分组，%d 个子分组也会一并删除"},
	"group.deleted.tree": {`✓ deleted group "%s" and its sub-groups`, "✓ 已删除分组「%s」及其子分组"},
	"group.under.new":    {"✓ new group under %s", "✓ 已在 %s 下新建分组"},
	"menu.new":           {"+ New group…", "+ 新建分组…"},
	"drag.target":        {`── → moving "%s" to "%s"`, "── → 移动「%s」到「%s」"},
	"empty.query":        {`No snippet matches "%s"`, "没有匹配「%s」的 snippet"},
	"empty.all":          {"No snippets yet — press a to create one", "还没有 snippet，按 a 新建"},
	"pick.first":         {"select a snippet first", "先选中一个 snippet"},
	"already.in":         {`already in "%s"`, "已经在「%s」"},
	"moved.to":           {`✓ moved to "%s"`, "✓ 已移到「%s」"},
	"prompt.new":         {"New group name ", "新分组名 "},
	"prompt.ren":         {"Rename ", "重命名 "},
	"prompt.bad":         {"✗ name is empty or taken", "✗ 名称为空或已存在"},
	"group.saved":        {`✓ group "%s" saved`, "✓ 已保存分组「%s」"},
	"group.del":          {`Delete group "%s"? %d snippets move to Ungrouped`, "删除分组「%s」? %d 个 snippet 回到未分组"},
	"group.deleted":      {`✓ deleted group "%s"`, "✓ 已删除分组「%s」"},
	"snip.del":           {`Delete "%s"? The file moves to trash/`, "删除「%s」? 文件移入 trash/"},
	"snip.deleted":       {`✓ deleted "%s" (moved to trash/)`, "✓ 已删除「%s」(移入 trash/)"},
	"del.failed":         {"✗ delete failed: %v", "✗ 删除失败: %v"},
	"save.failed":        {"✗ save failed: %v", "✗ 保存失败: %v"},
	"snip.new":           {"✓ created \"%s\"", "✓ 已新建「%s」"},
	"snip.saved":         {"✓ saved \"%s\"", "✓ 已保存「%s」"},
	"refreshed":          {"✓ refreshed", "✓ 已刷新"},
	"foreign":            {"✗ has unknown fields — edit it in Otty's Recipes", "✗ 含未知字段，请在 Otty 的 Recipes 里编辑"},

	"form.edit":      {"Edit snippet", "编辑 snippet"},
	"form.new":       {"New snippet → %s", "新建 snippet → %s"},
	"form.name":      {"Name  ", "名称  "},
	"form.alias":     {"alias ", "alias "},
	"form.text":      {"Text", "内容"},
	"form.tiphint":   {"  [group]-name files it automatically · {{clipboard}} {{date}}", "  名称写成 [组]-名字 自动分组 · {{clipboard}} {{date}}"},
	"form.noname":    {"✗ name can't be empty", "✗ 名称不能为空"},
	"form.noalias":   {"✗ alias can't be empty or contain spaces", "✗ alias 不能为空或含空格"},
	"form.notext":    {"✗ text can't be empty", "✗ 内容不能为空"},
	"form.aliasused": {`✗ alias already used by "%s"`, "✗ alias 已被「%s」占用"},

	"send.nopane":    {"✗ no terminal pane to write to", "✗ 没有可写入的终端 pane"},
	"send.denied":    {"✗ rejected (check ipc-allow-send-keys)", "✗ 被拒绝 (检查 ipc-allow-send-keys)"},
	"send.failed":    {"✗ send failed: %s", "✗ 发送失败: %s"},
	"send.rejected":  {"✗ rejected: %s", "✗ 被拒绝: %s"},
	"send.sensitive": {"✗ herdr pane %s is an SSH session (check ipc-allow-sensitive-sessions)", "✗ herdr pane %s 是 SSH 会话 (检查 ipc-allow-sensitive-sessions)"},
	"send.ran":       {"✓ ran", "✓ 已执行"},
	"send.filled":    {"✓ filled in", "✓ 已填入"},
	"copy.failed":    {"✗ copy failed: %v", "✗ 复制失败: %v"},
	"copy.done":      {"✓ copied", "✓ 已复制"},
}

func tabNames() [2]string { return [2]string{tr("tab.list"), tr("tab.group")} }

// ungroupedName is the label of the catch-all group. It is only ever a display
// label: the metadata uses "" (or "none" when pinned) for it.
func ungroupedName() string { return tr("ungrouped") }

// isUngroupedLabel reports whether name is the ungrouped label in either
// language, so a group can't be created with that name.
func isUngroupedLabel(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return n == strings.ToLower(dict["ungrouped"][0]) || n == strings.ToLower(dict["ungrouped"][1])
}
