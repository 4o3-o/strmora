// Package inventory parses local directory trees without any network access.
package inventory

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

type Entry struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}
type Tree struct {
	DuplicatePaths []string `json:"duplicate_paths,omitempty"`
	Root           string   `json:"root"`
	Entries        []Entry  `json:"entries"`
	Encoding       string   `json:"encoding"`
}

// Parse accepts UTF-8 and BOM-marked UTF-16. Parsing does not establish export completeness.
func Parse(b []byte) (Tree, error) {
	var tree Tree
	var s string
	switch {
	case bytes.HasPrefix(b, []byte{0xff, 0xfe}) || bytes.HasPrefix(b, []byte{0xfe, 0xff}):
		tree.Encoding = "UTF-16 LE"
		var order binary.ByteOrder = binary.LittleEndian
		if b[0] == 0xfe {
			order = binary.BigEndian
			tree.Encoding = "UTF-16 BE"
		}
		b = b[2:]
		if len(b)%2 != 0 {
			return tree, fmt.Errorf("odd UTF-16 byte count")
		}
		u := make([]uint16, len(b)/2)
		for i := range u {
			u[i] = order.Uint16(b[i*2:])
		}
		for i := 0; i < len(u); i++ {
			if u[i] >= 0xd800 && u[i] <= 0xdbff {
				if i+1 >= len(u) || u[i+1] < 0xdc00 || u[i+1] > 0xdfff {
					return tree, fmt.Errorf("unpaired surrogate")
				}
				i++
			} else if u[i] >= 0xdc00 && u[i] <= 0xdfff {
				return tree, fmt.Errorf("unpaired surrogate")
			}
		}
		s = string(utf16.Decode(u))
	default:
		tree.Encoding = "UTF-8"
		b = bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})
		if !utf8.Valid(b) {
			return tree, fmt.Errorf("invalid UTF-8; UTF-16 requires BOM")
		}
		s = string(b)
	}
	var stack []int
	seen := map[string]bool{}
	rootSeen := false
	for n, line := range strings.Split(s, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !rootSeen {
			if !strings.HasPrefix(line, "|——") {
				return tree, fmt.Errorf("line %d: root marker required", n+1)
			}
			tree.Root = strings.TrimPrefix(line, "|——")
			if err := validName(tree.Root); err != nil {
				return tree, err
			}
			rootSeen = true
			continue
		}
		depth := 0
		rest := line
		for strings.HasPrefix(rest, "| ") {
			depth++
			rest = rest[2:]
		}
		if depth < 1 || !strings.HasPrefix(rest, "|-") {
			return tree, fmt.Errorf("line %d: invalid prefix", n+1)
		}
		name := rest[2:]
		if err := validName(name); err != nil {
			return tree, fmt.Errorf("line %d: %w", n+1, err)
		}
		if depth > len(stack)+1 {
			return tree, fmt.Errorf("line %d: missing parent", n+1)
		}
		stack = stack[:depth-1]
		p := name
		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			tree.Entries[parent].Kind = "directory"
			p = path.Join(tree.Entries[parent].Path, name)
		}
		if seen[p] {
			tree.DuplicatePaths = append(tree.DuplicatePaths, p)
		}
		seen[p] = true
		tree.Entries = append(tree.Entries, Entry{p, "unknown"})
		stack = append(stack, len(tree.Entries)-1)
	}
	if !rootSeen {
		return tree, fmt.Errorf("empty tree")
	}
	return tree, nil
}
func validName(s string) error {
	if s == "" || s == "." || s == ".." || strings.ContainsAny(s, "/\\\x00") {
		return fmt.Errorf("invalid path component")
	}
	return nil
}
