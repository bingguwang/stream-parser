package main

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	_ "github.com/ying32/govcl/pkgs/winappres"
	"github.com/ying32/govcl/vcl"
	"sort"
)

func (f *TMainForm) jsonTree(str string, captain string) {
	var data interface{}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		vcl.ShowMessage(err.Error())
		return
	}
	f.TreeView1.Items().BeginUpdate()
	f.TreeView1.SetBorderWidth(2)
	defer f.TreeView1.Items().EndUpdate()
	root := f.TreeView1.Items().AddChild(nil, captain)
	f.buildTree(root, data, "")
	f.TreeView1.SetWidth(300)
}
func (f *TMainForm) ClearTreeViewItems() {
	f.TreeView1.Items().Clear()
}

func (f *TMainForm) buildTree(node *vcl.TTreeNode, data interface{}, keyName string) {

	switch data.(type) {
	case map[string]interface{}:

		var child *vcl.TTreeNode
		if keyName != "" {
			child = f.TreeView1.Items().AddChild(node, keyName)
			child = f.TreeView1.Items().AddChild(child, "")
		} else {
			child = f.TreeView1.Items().AddChild(node, "")
		}

		// 因为随机问题，考虑解决办法，有序总比乱序好。。。。
		keys := make([]string, 0)
		for key := range data.(map[string]interface{}) {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if val, ok := data.(map[string]interface{})[key]; ok {
				f.buildTree(child, val, key)
			}
		}

	case []interface{}:
		var child *vcl.TTreeNode
		if keyName != "" {
			child = f.TreeView1.Items().AddChild(node, keyName)
			child = f.TreeView1.Items().AddChild(child, fmt.Sprintf("[%d]", len(data.([]interface{}))))
		} else {
			//child = f.TreeView1.Items().AddChild(node, "Array")
			child = f.TreeView1.Items().AddChild(node, fmt.Sprintf("[%d]", len(data.([]interface{}))))
		}
		for _, val := range data.([]interface{}) {
			f.buildTree(child, val, "")
		}

	default:

		if node != nil && node.IsValid() {
			if keyName == "" {
				f.TreeView1.Items().AddChild(node, fmt.Sprintf("%v", data))
			} else {
				f.TreeView1.Items().AddChild(node, fmt.Sprintf("%s:%v", keyName, data))
			}
		}
	}
}

var jsonData = `{}`
