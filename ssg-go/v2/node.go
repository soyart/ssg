package ssg

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/soyart/ssg/ssg-go"
)

type NodeType uint8

const (
	NodeTypeDir NodeType = 1 << iota
	NodeTypeFile
	NodeTypeMarker
)

type Node struct {
	Type NodeType `json:"type"`
	Path string   `json:"path"`
	Base string   `json:"base"`

	Children []Node
	data     []byte
}

func (t NodeType) String() string {
	switch t {
	case NodeTypeDir:
		return "NODE-DIR"
	case NodeTypeFile:
		return "NODE-FILE"
	case NodeTypeMarker:
		return "NODE-MARKER"
	}
	return "NODE-UNKNWON"
}

func (n Node) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"type":     n.Type,
		"path":     n.Path,
		"base":     n.Base,
		"children": n.Children,
	}
	if len(n.data) != 0 {
		m["data"] = string(n.data)
	}

	return json.Marshal(m)
}

func (n Node) String() string {
	j, err := json.Marshal(n)
	if err != nil {
		return `"JSON-ERROR"`
	}
	return string(j)
}

func (n *Node) Data() ([]byte, error) {
	switch n.Type {
	case NodeTypeFile, NodeTypeMarker:
		return n.data, nil
	}

	return nil, fmt.Errorf("'%s' is dir", n.Path)
}

func nodeType(e fs.DirEntry) NodeType {
	if e.IsDir() {
		return NodeTypeDir
	}
	switch e.Name() {
	case ssg.MarkerFooter, ssg.MarkerHeader:
		return NodeTypeMarker
	}

	return NodeTypeFile
}

func populate(n *Node) error {
	if n.Type != NodeTypeDir {
		data, err := ssg.ReadFile(n.Path)
		if err != nil {
			return fmt.Errorf("failed to populate file '%s' data: %w", n.Path, err)
		}
		n.data = data
		return nil
	}

	entries, err := os.ReadDir(n.Path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		child := Node{
			Type: nodeType(entry),
			Path: filepath.Join(n.Path, entry.Name()),
			Base: entry.Name(),
		}
		err := populate(&child)
		if err != nil {
			return fmt.Errorf("failed to populate child '%s': %w", child.Path, err)
		}
		n.Children = append(n.Children, child)
	}

	return nil
}

func Walk(path string) (Node, error) {
	root := Node{
		Type: NodeTypeDir,
		Path: path,
		Base: "",
	}
	err := populate(&root)
	if err != nil {
		return Node{}, fmt.Errorf("failed to populate '%s': %w", path, err)
	}

	return root, nil
}
