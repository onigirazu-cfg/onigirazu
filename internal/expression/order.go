package expression

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/expr-lang/expr/ast"
)

const orderKey = "__onigirazu_dict_order"

// Orders and their maps live for one Eval only. Retaining maps prevents pointer
// reuse during evaluation; neither state nor closures enter the program cache.
type mapOrder struct {
	keys map[uintptr][]string
	maps []map[string]interface{}
}

func (o *mapOrder) remember(m map[string]interface{}, keys []string) {
	if o == nil {
		return
	}
	o.keys[reflect.ValueOf(m).Pointer()] = keys
	o.maps = append(o.maps, m)
}
func (o *mapOrder) names(m map[string]interface{}) []string {
	if o != nil {
		if keys, ok := o.keys[reflect.ValueOf(m).Pointer()]; ok {
			return keys
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func (o *mapOrder) mapKeys(rv reflect.Value) []reflect.Value {
	if m, ok := rv.Interface().(map[string]interface{}); ok {
		names := o.names(m)
		keys := make([]reflect.Value, len(names))
		for i, k := range names {
			keys[i] = reflect.ValueOf(k)
		}
		return keys
	}
	keys := rv.MapKeys()
	sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface()) })
	return keys
}

type orderPatch struct{}

func (orderPatch) Visit(node *ast.Node) {
	if m, ok := (*node).(*ast.MapNode); ok {
		args := []ast.Node{&ast.IdentifierNode{Value: orderKey}}
		for _, n := range m.Pairs {
			p, ok := n.(*ast.PairNode)
			if !ok {
				return
			}
			args = append(args, p.Key, p.Value)
		}
		ast.Patch(node, &ast.CallNode{Callee: &ast.IdentifierNode{Value: "jinja_dict"}, Arguments: args})
		return
	}
	call, ok := (*node).(*ast.CallNode)
	if !ok {
		return
	}
	name, ok := call.Callee.(*ast.IdentifierNode)
	if !ok {
		return
	}
	switch name.Value {
	// to_nice_json and to_yaml sort keys in Ansible (sort_keys=True, PyYAML
	// default), so they keep the sorted Go path
	case "to_json", "combine", "dict2items", "jinja_method":
		ast.Patch(node, &ast.CallNode{Callee: &ast.IdentifierNode{Value: "jinja_ordered"}, Arguments: append([]ast.Node{&ast.IdentifierNode{Value: orderKey}, &ast.StringNode{Value: name.Value}}, call.Arguments...)})
	}
}
func orderedDict(p ...interface{}) (interface{}, error) {
	o, _ := p[0].(*mapOrder) // nil: keys print sorted
	m := map[string]interface{}{}
	keys := []string{}
	for i := 1; i < len(p); i += 2 {
		k := fmt.Sprint(p[i])
		if _, ok := m[k]; !ok {
			keys = append(keys, k)
		}
		m[k] = p[i+1]
	}
	o.remember(m, keys)
	return m, nil
}
func orderedCall(p ...interface{}) (interface{}, error) {
	o, _ := p[0].(*mapOrder) // nil: keys print sorted
	name, ok := p[1].(string)
	if !ok {
		return nil, fmt.Errorf("ordered operation needs a name")
	}
	args := p[2:]
	switch name {
	case "to_json":
		return pyJSONOrdered(args[0], 0, o)
	case "combine":
		return combineOrdered(o, args...)
	case "dict2items":
		return o.items(args[0])
	case "jinja_method":
		if len(args) >= 2 && args[1] == "items" {
			return o.items(args[0])
		}
		return callMethod(args...)
	}
	return nil, fmt.Errorf("unknown ordered operation %s", name)
}
func (o *mapOrder) items(v interface{}) (interface{}, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return Dict2Items(v)
	}
	out := make([]interface{}, 0, len(m))
	for _, k := range o.names(m) {
		out = append(out, map[string]interface{}{"key": k, "value": m[k]})
	}
	return out, nil
}
