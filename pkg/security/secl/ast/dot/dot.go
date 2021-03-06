// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package dot

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/security/secl/ast"
)

type node struct {
	id    string
	label string
}

func newNode(id, label string) *node {
	return &node{id: id, label: label}
}

// Marshaler converts a rule to the DOT format
type Marshaler struct {
	w io.Writer
}

func (d *Marshaler) writeString(s string) error {
	_, err := io.WriteString(d.w, s)
	return err
}

func (d *Marshaler) writeNode(node interface{}) error {
	id, err := d.getID(node)
	if err != nil {
		return err
	}

	if err := d.writeString(id + "[label=\"" + d.getLabel(node) + "\"]\n"); err != nil {
		return err
	}

	children, err := d.getChildren(node)
	if err != nil {
		return err
	}

	for _, child := range children {
		if err := d.writeEdge(node, child); err != nil {
			return err
		}

		if err := d.writeNode(child); err != nil {
			return err
		}
	}

	return nil
}

func (d *Marshaler) writeEdge(parent, child interface{}) error {
	parentID, err := d.getID(parent)
	if err != nil {
		return err
	}

	childID, err := d.getID(child)
	if err != nil {
		return err
	}

	return d.writeString(parentID + " -> " + childID + "\n")
}

func (d *Marshaler) getID(n interface{}) (string, error) {
	switch n := n.(type) {
	case *ast.Rule:
		return fmt.Sprintf("Rule%d", n.Pos.Offset), nil
	case *ast.Expression:
		return fmt.Sprintf("Expression%d", n.Pos.Offset), nil
	case *ast.Comparison:
		return fmt.Sprintf("Comparison%d", n.Pos.Offset), nil
	case *ast.ScalarComparison:
		return fmt.Sprintf("ScalarComparison%d", n.Pos.Offset), nil
	case *ast.ArrayComparison:
		return fmt.Sprintf("ArrayComparison%d", n.Pos.Offset), nil
	case *ast.Array:
		return fmt.Sprintf("Array%d", n.Pos.Offset), nil
	case *ast.BooleanExpression:
		return fmt.Sprintf("BooleanExpression%d", n.Pos.Offset), nil
	case *ast.BitOperation:
		return fmt.Sprintf("BitOperation%d", n.Pos.Offset), nil
	case *ast.Unary:
		return fmt.Sprintf("Unary%d", n.Pos.Offset), nil
	case *ast.Primary:
		return fmt.Sprintf("Primary%d", n.Pos.Offset), nil
	case *node:
		return n.id, nil
	default:
		return "", fmt.Errorf("unsupported node type: %s", reflect.TypeOf(n))
	}
}

func (d *Marshaler) getLabel(n interface{}) string {
	switch n := n.(type) {
	case *node:
		return n.label
	default:
		split := strings.SplitN(reflect.TypeOf(n).String(), ".", 2)
		return split[len(split)-1]
	}
}

func (d *Marshaler) getChildren(n interface{}) ([]interface{}, error) {
	switch n := n.(type) {
	case *ast.Rule:
		return []interface{}{n.BooleanExpression}, nil
	case *ast.Expression:
		children := []interface{}{n.Comparison}
		if n.Op != nil {
			children = append(children, newNode(fmt.Sprintf("Op%p", n.Op), fmt.Sprintf("Op\\n%s", *n.Op)))
		}
		if n.Next != nil {
			children = append(children, n.Next)
		}
		return children, nil
	case *ast.BooleanExpression:
		return []interface{}{n.Expression}, nil
	case *ast.Comparison:
		children := []interface{}{n.BitOperation}
		if n.ArrayComparison != nil {
			children = append(children, n.ArrayComparison)
		}
		if n.ScalarComparison != nil {
			children = append(children, n.ScalarComparison)
		}
		return children, nil
	case *ast.ArrayComparison:
		return []interface{}{
			newNode(fmt.Sprintf("Op%p", n.Op), fmt.Sprintf("Op\\n%s", *n.Op)),
			n.Array,
		}, nil
	case *ast.ScalarComparison:
		return []interface{}{
			newNode(fmt.Sprintf("Op%p", n.Op), fmt.Sprintf("Op\\n%s", *n.Op)),
			n.Next,
		}, nil
	case *ast.Array:
		if len(n.Strings) > 0 {
			return []interface{}{
				newNode(fmt.Sprintf("Array%p", n), strings.Join(n.Strings, ",")),
			}, nil
		}
		s := ""
		for i, n := range n.Numbers {
			if i != 0 {
				s += ", " + strconv.Itoa(n)
			} else {
				s += strconv.Itoa(n)
			}
		}
		return []interface{}{
			newNode(fmt.Sprintf("Array%p", n), s),
		}, nil
	case *ast.BitOperation:
		children := []interface{}{n.Unary}
		if n.Op != nil {
			children = append(children, newNode(fmt.Sprintf("Op%p", n.Op), fmt.Sprintf("Op\\n%s", *n.Op)))
		}
		if n.Next != nil {
			children = append(children, n.Next)
		}
		return children, nil
	case *ast.Unary:
		var children []interface{}
		if n.Op != nil {
			children = append(children, newNode(fmt.Sprintf("Op%p", n.Op), fmt.Sprintf("Op\\n%s", *n.Op)))
		}
		if n.Unary != nil {
			children = append(children, n.Unary)
		}
		if n.Primary != nil {
			children = append(children, n.Primary)
		}
		return children, nil
	case *ast.Primary:
		if n.Ident != nil {
			return []interface{}{newNode(fmt.Sprintf("Ident%p", n.Ident), fmt.Sprintf("Ident\\n%s", *n.Ident))}, nil
		}
		if n.Number != nil {
			return []interface{}{newNode(fmt.Sprintf("Number%p", n.Number), fmt.Sprintf("Number\\n%d", *n.Number))}, nil
		}
		if n.String != nil {
			return []interface{}{newNode(fmt.Sprintf("String%p", n.String), fmt.Sprintf("String\\n%s", *n.String))}, nil
		}
		if n.SubExpression != nil {
			return []interface{}{n.SubExpression}, nil
		}
		return nil, fmt.Errorf("empty ast.Primary")
	case *node:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported node type: %s", reflect.TypeOf(n))
	}
}

// MarshalRule marshals the AST of a rule to DOT format
func (d *Marshaler) MarshalRule(r *ast.Rule) error {
	if err := d.writeString("digraph {\n"); err != nil {
		return err
	}
	if err := d.writeNode(r.BooleanExpression); err != nil {
		return err
	}
	return d.writeString("}\n")
}

// NewMarshaler returns a new rule DOT marshaler
func NewMarshaler(w io.Writer) *Marshaler {
	return &Marshaler{w: w}
}
