package engine

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestLoopItemsRenderNestedTemplates(t *testing.T) {
	e := &ExecutionEngine{templateEngine: template.NewEngine()}
	vars := map[string]interface{}{
		"rp":    "2",
		"items": []interface{}{map[string]interface{}{"name": "b", "value": "{{ rp }}"}},
	}
	loop := &types.Loop{Items: []interface{}{
		map[string]interface{}{"name": "a", "value": "{{ rp }}"},
		[]interface{}{"{{ rp }}", "x"},
		"{{ rp }}",
		"plain",
	}}
	items, err := e.getLoopItems(context.Background(), loop, vars)
	if err != nil {
		t.Fatal(err)
	}
	if got := items[0].(map[string]interface{})["value"]; got != "2" {
		t.Errorf("map item value = %v", got)
	}
	if got := items[1].([]interface{})[0]; got != "2" {
		t.Errorf("list item = %v", got)
	}
	if items[2] != "2" || items[3] != "plain" {
		t.Errorf("scalar items = %v, %v", items[2], items[3])
	}

	fromVar, err := e.getLoopItems(context.Background(), &types.Loop{Expr: "items"}, vars)
	if err != nil {
		t.Fatal(err)
	}
	if got := fromVar[0].(map[string]interface{})["value"]; got != "2" {
		t.Errorf("item from a variable = %v", got)
	}
}
