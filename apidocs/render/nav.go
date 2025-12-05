package render

import (
	"fmt"
	"strings"
)

// suffixNode 表示 tags 次级分组的树节点，用于将 "A/B/C" 拆分为层级结构并保留插入顺序。
type suffixNode struct {
	name     string
	children map[string]*suffixNode
	items    [][2]string
	order    []string
}

// buildSuffixTree 根据所有次级分组字符串构建分组树；itemsMap 为每个原始分组对应的接口项列表。
func buildSuffixTree(sufs []string, itemsMap map[string][][2]string) *suffixNode {
	root := &suffixNode{name: "", children: make(map[string]*suffixNode), order: make([]string, 0, 8)}
	for _, suf := range sufs {
		parts := strings.Split(suf, "/")
		cur := root
		for _, seg := range parts {
			seg = strings.TrimSpace(seg)
			if seg == "" {
				continue
			}
			if cur.children == nil {
				cur.children = make(map[string]*suffixNode)
				if cur.order == nil {
					cur.order = make([]string, 0, 8)
				}
			}
			if _, ok := cur.children[seg]; !ok {
				cur.children[seg] = &suffixNode{name: seg, children: make(map[string]*suffixNode), order: make([]string, 0, 8)}
				cur.order = append(cur.order, seg)
			}
			cur = cur.children[seg]
		}
		if len(itemsMap[suf]) > 0 {
			cur.items = append(cur.items, itemsMap[suf]...)
		}
	}
	return root
}

// renderSuffixTreeMenu 将分组树渲染为侧边菜单的 HTML；acc 为累积锚点前缀，depth 用于缩进计算。
func renderSuffixTreeMenu(b *strings.Builder, pre string, node *suffixNode, acc string, depth int) {
	if node == nil || len(node.children) == 0 {
		return
	}
	names := node.order
	for _, name := range names {
		child := node.children[name]
		childAcc := acc
		if childAcc == "" {
			childAcc = "group-" + slugify(pre)
		}
		childAcc = childAcc + "-" + slugify(name)
		pad := 14 * (depth + 1)
		b.WriteString("<details class=\"subgrp\" open><summary style=\"padding-left:" + fmt.Sprintf("%d", pad) + "px\"><a href=\"#" + childAcc + "\">" + htmlEscape(name) + "</a></summary>")
		for _, it := range child.items {
			ip := 28 + 14*depth
			b.WriteString("<div class=\"item\" style=\"padding-left:" + fmt.Sprintf("%d", ip) + "px\"><a class=\"item-link\" href=\"#" + it[1] + "\">" + htmlEscape(it[0]) + "</a></div>")
		}
		renderSuffixTreeMenu(b, pre, child, childAcc, depth+1)
		b.WriteString("</details>")
	}
}

// renderSuffixTreeHeadings 将分组树渲染为正文中的分组标题（h2），保证与菜单层级顺序一致。
func renderSuffixTreeHeadings(b *strings.Builder, pre string, node *suffixNode, acc string) {
	if node == nil || len(node.children) == 0 {
		return
	}
	names := node.order
	for _, name := range names {
		child := node.children[name]
		childAcc := acc
		if childAcc == "" {
			childAcc = "group-" + slugify(pre)
		}
		childAcc = childAcc + "-" + slugify(name)
		b.WriteString("<h2 id=\"" + childAcc + "\">" + htmlEscape(name) + "</h2>")
		renderSuffixTreeHeadings(b, pre, child, childAcc)
	}
}

// toNavVM 将树结构转换为视图模型，用于模板渲染侧边导航与正文标题。
func toNavVM(pre string, node *suffixNode, acc string) []*NavGroupVM {
	if node == nil || len(node.children) == 0 {
		return nil
	}
	names := node.order
	res := make([]*NavGroupVM, 0, len(names))
	for _, name := range names {
		child := node.children[name]
		childAcc := acc
		if childAcc == "" {
			childAcc = "group-" + slugify(pre)
		}
		childAcc = childAcc + "-" + slugify(name)
		itms := make([]NavItemVM, 0, len(child.items))
		for _, it := range child.items {
			itms = append(itms, NavItemVM{Summary: it[0], Anchor: it[1]})
		}
		vm := &NavGroupVM{Name: name, Id: childAcc, Items: itms}
		vm.Children = toNavVM(pre, child, childAcc)
		res = append(res, vm)
	}
	return res
}

// buildTopNavGroups 构造顶层分组视图模型。
func buildTopNavGroups(preOrder []string, sfxOrder map[string][]string, groups map[string]map[string][][2]string) []*NavGroupVM {
	tops := make([]*NavGroupVM, 0, len(preOrder))
	for _, pre := range preOrder {
		tree := buildSuffixTree(sfxOrder[pre], groups[pre])
		children := toNavVM(pre, tree, "")
		tops = append(tops, &NavGroupVM{Name: pre, Id: "group-" + slugify(pre), Children: children})
	}
	return tops
}

// buildMainGroups 构造正文分组视图模型，用于生成 h1/h2 标题并对应锚点。
func buildMainGroups(preOrder []string, sfxOrder map[string][]string, groups map[string]map[string][][2]string) []*MainGroupVM {
	res := make([]*MainGroupVM, 0, len(preOrder))
	for _, pre := range preOrder {
		tree := buildSuffixTree(sfxOrder[pre], groups[pre])
		children := toNavVM(pre, tree, "group-"+slugify(pre))
		res = append(res, &MainGroupVM{PreName: pre, PreId: "group-" + slugify(pre), Children: children})
	}
	return res
}
