package validator

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNewSchemaMigrator(t *testing.T) {
	sm := NewSchemaMigrator()

	if sm == nil {
		t.Fatal("NewSchemaMigrator returned nil")
	}

	if len(sm.supportedVersions) != 2 {
		t.Errorf("Expected 2 default supported versions, got %d", len(sm.supportedVersions))
	}
}

func TestRegisterMigration(t *testing.T) {
	tests := []struct {
		name      string
		migration *types.SchemaMigration
		wantErr   bool
	}{
		{
			name: "valid migration",
			migration: &types.SchemaMigration{
				From: 1,
				To:   2,
				Rules: []types.MigrationRule{
					{
						Type:     types.MigrationRuleTypeRename,
						OldParam: "old",
						NewParam: "new",
					},
				},
			},
			wantErr: false,
		},
		{
			name:      "nil migration",
			migration: nil,
			wantErr:   true,
		},
		{
			name: "invalid version order",
			migration: &types.SchemaMigration{
				From:  2,
				To:    1,
				Rules: []types.MigrationRule{},
			},
			wantErr: true,
		},
		{
			name: "same from and to version",
			migration: &types.SchemaMigration{
				From:  1,
				To:    1,
				Rules: []types.MigrationRule{},
			},
			wantErr: true,
		},
		{
			name: "zero version",
			migration: &types.SchemaMigration{
				From:  0,
				To:    1,
				Rules: []types.MigrationRule{},
			},
			wantErr: true,
		},
		{
			name: "empty rules",
			migration: &types.SchemaMigration{
				From:  1,
				To:    2,
				Rules: []types.MigrationRule{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSchemaMigrator()
			err := sm.RegisterMigration(tt.migration)

			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterMigration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyRenameRule(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]interface{}
		rule      types.MigrationRule
		wantErr   bool
		checkFunc func(map[string]interface{}) error
	}{
		{
			name: "simple rename",
			params: map[string]interface{}{
				"old_param": "value",
			},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeRename,
				OldParam: "old_param",
				NewParam: "new_param",
			},
			wantErr: false,
			checkFunc: func(params map[string]interface{}) error {
				if v, ok := params["new_param"]; !ok || v != "value" {
					return testError("new_param not set correctly")
				}
				if _, ok := params["old_param"]; ok {
					return testError("old_param should be deleted")
				}
				return nil
			},
		},
		{
			name: "rename non-existent param",
			params: map[string]interface{}{
				"other": "value",
			},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeRename,
				OldParam: "old_param",
				NewParam: "new_param",
			},
			wantErr: false,
			checkFunc: func(params map[string]interface{}) error {
				if _, ok := params["new_param"]; ok {
					return testError("new_param should not be created for non-existent old_param")
				}
				return nil
			},
		},
		{
			name:   "missing old_param",
			params: map[string]interface{}{},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeRename,
				OldParam: "",
				NewParam: "new_param",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSchemaMigrator()
			result, err := sm.applyRenameRule(tt.params, tt.rule)

			if (err != nil) != tt.wantErr {
				t.Errorf("applyRenameRule() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.checkFunc != nil {
				if err := tt.checkFunc(result); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestApplyTransformRule(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		rule    types.MigrationRule
		wantErr bool
		want    interface{}
	}{
		{
			name: "string to integer",
			params: map[string]interface{}{
				"port": "8080",
			},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeTransform,
				OldParam: "port",
				FromType: "string",
				ToType:   "integer",
			},
			wantErr: false,
			want:    8080,
		},
		{
			name: "integer to string",
			params: map[string]interface{}{
				"port": 8080,
			},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeTransform,
				OldParam: "port",
				FromType: "integer",
				ToType:   "string",
			},
			wantErr: false,
			want:    "8080",
		},
		{
			name: "string to boolean true",
			params: map[string]interface{}{
				"enabled": "true",
			},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeTransform,
				OldParam: "enabled",
				FromType: "string",
				ToType:   "boolean",
			},
			wantErr: false,
			want:    true,
		},
		{
			name: "string to boolean false",
			params: map[string]interface{}{
				"enabled": "false",
			},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeTransform,
				OldParam: "enabled",
				FromType: "string",
				ToType:   "boolean",
			},
			wantErr: false,
			want:    false,
		},
		{
			name: "rename during transform",
			params: map[string]interface{}{
				"old_port": "8080",
			},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeTransform,
				OldParam: "old_port",
				NewParam: "new_port",
				FromType: "string",
				ToType:   "integer",
			},
			wantErr: false,
			want:    8080,
		},
		{
			name:   "missing parameter",
			params: map[string]interface{}{},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeTransform,
				OldParam: "port",
				FromType: "string",
				ToType:   "integer",
			},
			wantErr: false,
		},
		{
			name: "invalid string to integer",
			params: map[string]interface{}{
				"port": "not_a_number",
			},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeTransform,
				OldParam: "port",
				FromType: "string",
				ToType:   "integer",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSchemaMigrator()
			result, err := sm.applyTransformRule(tt.params, tt.rule)

			if (err != nil) != tt.wantErr {
				t.Errorf("applyTransformRule() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.want != nil {
				if result[tt.rule.OldParam] != nil || result[tt.rule.NewParam] != nil {
					paramName := tt.rule.NewParam
					if paramName == "" {
						paramName = tt.rule.OldParam
					}
					if v, ok := result[paramName]; !ok || v != tt.want {
						t.Errorf("Expected %v, got %v", tt.want, v)
					}
				}
			}
		})
	}
}

func TestApplyRemoveRule(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		rule    types.MigrationRule
		wantErr bool
	}{
		{
			name: "remove existing parameter",
			params: map[string]interface{}{
				"deprecated": "value",
				"keep":       "value",
			},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeRemove,
				OldParam: "deprecated",
			},
			wantErr: false,
		},
		{
			name: "remove non-existent parameter",
			params: map[string]interface{}{
				"keep": "value",
			},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeRemove,
				OldParam: "deprecated",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSchemaMigrator()
			result, err := sm.applyRemoveRule(tt.params, tt.rule)

			if (err != nil) != tt.wantErr {
				t.Errorf("applyRemoveRule() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if _, ok := result[tt.rule.OldParam]; ok {
					t.Errorf("Parameter %s should be removed", tt.rule.OldParam)
				}
			}
		})
	}
}

func TestApplyAddDefaultRule(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		rule    types.MigrationRule
		wantErr bool
		want    interface{}
	}{
		{
			name:   "add default to missing parameter",
			params: map[string]interface{}{},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeAddDefault,
				NewParam: "new_param",
				Default:  "default_value",
			},
			wantErr: false,
			want:    "default_value",
		},
		{
			name: "don't override existing parameter",
			params: map[string]interface{}{
				"param": "existing_value",
			},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeAddDefault,
				OldParam: "param",
				Default:  "default_value",
			},
			wantErr: false,
		},
		{
			name:   "add nil default",
			params: map[string]interface{}{},
			rule: types.MigrationRule{
				Type:     types.MigrationRuleTypeAddDefault,
				NewParam: "param",
				Default:  nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSchemaMigrator()
			result, err := sm.applyAddDefaultRule(tt.params, tt.rule)

			if (err != nil) != tt.wantErr {
				t.Errorf("applyAddDefaultRule() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.want != nil {
				paramName := tt.rule.NewParam
				if paramName == "" {
					paramName = tt.rule.OldParam
				}
				if v, ok := result[paramName]; !ok || v != tt.want {
					t.Errorf("Expected %v, got %v", tt.want, v)
				}
			}
		})
	}
}

func TestCheckMigrationNeeded(t *testing.T) {
	sm := NewSchemaMigrator()
	_ = sm.RegisterMigration(&types.SchemaMigration{
		From: 1,
		To:   2,
		Rules: []types.MigrationRule{
			{Type: types.MigrationRuleTypeRename, OldParam: "old", NewParam: "new"},
		},
	})

	tests := []struct {
		name string
		from int
		to   int
		want bool
	}{
		{"same version", 1, 1, false},
		{"forward migration", 1, 2, true},
		{"backward migration", 2, 1, false},
		{"no path", 1, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sm.CheckMigrationNeeded(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("CheckMigrationNeeded(%d, %d) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestGetMigrationPath(t *testing.T) {
	sm := NewSchemaMigrator()
	_ = sm.RegisterMigration(&types.SchemaMigration{
		From: 1,
		To:   2,
		Rules: []types.MigrationRule{
			{Type: types.MigrationRuleTypeRename, OldParam: "a", NewParam: "b"},
		},
	})
	_ = sm.RegisterMigration(&types.SchemaMigration{
		From: 2,
		To:   3,
		Rules: []types.MigrationRule{
			{Type: types.MigrationRuleTypeRename, OldParam: "c", NewParam: "d"},
		},
	})

	tests := []struct {
		name    string
		from    int
		to      int
		wantLen int
		wantErr bool
	}{
		{"same version", 1, 1, 0, false},
		{"direct migration", 1, 2, 1, false},
		{"multi-step migration", 1, 3, 2, false},
		{"backward migration", 2, 1, 0, true},
		{"no path", 1, 4, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := sm.GetMigrationPath(tt.from, tt.to)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetMigrationPath() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(path) != tt.wantLen {
				t.Errorf("GetMigrationPath() len = %d, want %d", len(path), tt.wantLen)
			}
		})
	}
}

func TestApplyMigration(t *testing.T) {
	sm := NewSchemaMigrator()

	tests := []struct {
		name      string
		params    map[string]interface{}
		migration types.SchemaMigration
		wantErr   bool
		checkFunc func(map[string]interface{}) error
	}{
		{
			name: "rename migration",
			params: map[string]interface{}{
				"old_name": "value",
			},
			migration: types.SchemaMigration{
				From: 1,
				To:   2,
				Rules: []types.MigrationRule{
					{
						Type:     types.MigrationRuleTypeRename,
						OldParam: "old_name",
						NewParam: "new_name",
					},
				},
			},
			wantErr: false,
			checkFunc: func(params map[string]interface{}) error {
				if v, ok := params["new_name"]; !ok || v != "value" {
					return testError("rename failed")
				}
				return nil
			},
		},
		{
			name: "multiple rules",
			params: map[string]interface{}{
				"param1": "value1",
				"param2": "value2",
			},
			migration: types.SchemaMigration{
				From: 1,
				To:   2,
				Rules: []types.MigrationRule{
					{
						Type:     types.MigrationRuleTypeRename,
						OldParam: "param1",
						NewParam: "renamed1",
					},
					{
						Type:     types.MigrationRuleTypeRemove,
						OldParam: "param2",
					},
				},
			},
			wantErr: false,
			checkFunc: func(params map[string]interface{}) error {
				if _, ok := params["renamed1"]; !ok {
					return testError("first rule not applied")
				}
				if _, ok := params["param2"]; ok {
					return testError("second rule not applied")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sm.ApplyMigration(tt.params, tt.migration)

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyMigration() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.checkFunc != nil {
				if err := tt.checkFunc(result); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestApplyMigrations(t *testing.T) {
	sm := NewSchemaMigrator()

	migrations := []types.SchemaMigration{
		{
			From: 1,
			To:   2,
			Rules: []types.MigrationRule{
				{
					Type:     types.MigrationRuleTypeRename,
					OldParam: "param1",
					NewParam: "param1_v2",
				},
			},
		},
		{
			From: 2,
			To:   3,
			Rules: []types.MigrationRule{
				{
					Type:     types.MigrationRuleTypeRename,
					OldParam: "param1_v2",
					NewParam: "param1_v3",
				},
			},
		},
	}

	params := map[string]interface{}{
		"param1": "value",
	}

	result, err := sm.ApplyMigrations(params, migrations)
	if err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}

	if v, ok := result["param1_v3"]; !ok || v != "value" {
		t.Errorf("Final param1_v3 = %v, want value", v)
	}
}

func TestValidateVersion(t *testing.T) {
	sm := NewSchemaMigrator()
	_ = sm.SetSupportedVersions([]int{1, 2, 3})

	tests := []struct {
		name    string
		version int
		wantErr bool
	}{
		{"valid version 1", 1, false},
		{"valid version 3", 3, false},
		{"unsupported version", 4, true},
		{"invalid version 0", 0, true},
		{"invalid version -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sm.ValidateVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVersion(%d) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}

func TestGetLatestVersion(t *testing.T) {
	sm := NewSchemaMigrator()
	_ = sm.SetSupportedVersions([]int{1, 2, 5, 3})

	latest := sm.GetLatestVersion()
	if latest != 5 {
		t.Errorf("GetLatestVersion() = %d, want 5", latest)
	}
}

func TestCanMigrateTo(t *testing.T) {
	sm := NewSchemaMigrator()
	_ = sm.RegisterMigration(&types.SchemaMigration{
		From: 1,
		To:   2,
		Rules: []types.MigrationRule{
			{Type: types.MigrationRuleTypeRename, OldParam: "a", NewParam: "b"},
		},
	})

	tests := []struct {
		name string
		from int
		to   int
		want bool
	}{
		{"same version", 1, 1, true},
		{"supported migration", 1, 2, true},
		{"unsupported migration", 1, 3, false},
		{"backward", 2, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sm.CanMigrateTo(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("CanMigrateTo(%d, %d) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestGetMigrationInfo(t *testing.T) {
	sm := NewSchemaMigrator()
	migration := &types.SchemaMigration{
		From: 1,
		To:   2,
		Rules: []types.MigrationRule{
			{Type: types.MigrationRuleTypeRename, OldParam: "a", NewParam: "b"},
		},
		Notes: "test migration",
	}
	_ = sm.RegisterMigration(migration)

	info := sm.GetMigrationInfo(1, 2)
	if info == nil {
		t.Fatal("GetMigrationInfo returned nil")
	}
	if info.Notes != "test migration" {
		t.Errorf("GetMigrationInfo notes = %s, want 'test migration'", info.Notes)
	}

	info2 := sm.GetMigrationInfo(1, 3)
	if info2 != nil {
		t.Fatal("GetMigrationInfo should return nil for non-existent migration")
	}
}

func TestSetSupportedVersions(t *testing.T) {
	sm := NewSchemaMigrator()

	tests := []struct {
		name     string
		versions []int
		wantErr  bool
	}{
		{"valid versions", []int{1, 2, 3}, false},
		{"empty versions", []int{}, true},
		{"invalid version 0", []int{0, 1}, true},
		{"invalid version -1", []int{-1, 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sm.SetSupportedVersions(tt.versions)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetSupportedVersions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultSchemaMigrator(t *testing.T) {
	sm := DefaultSchemaMigrator()

	if sm == nil {
		t.Fatal("DefaultSchemaMigrator returned nil")
	}

	// Should have built-in v1 to v2 migration
	info := sm.GetMigrationInfo(1, 2)
	if info == nil {
		t.Fatal("DefaultSchemaMigrator should have v1 to v2 migration")
	}

	// Test v1 to v2 migration (port: string -> integer)
	params := map[string]interface{}{
		"port": "8080",
	}

	result, err := sm.ApplyMigration(params, *info)
	if err != nil {
		t.Fatalf("ApplyMigration() error = %v", err)
	}

	if port, ok := result["port"].(int); !ok || port != 8080 {
		t.Errorf("port = %v, want 8080", result["port"])
	}
}

func TestTransformFunctions(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{"transformToString", testTransformToString},
		{"transformToInteger", testTransformToInteger},
		{"transformToBoolean", testTransformToBoolean},
		{"transformToArray", testTransformToArray},
		{"transformToObject", testTransformToObject},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func testTransformToString(t *testing.T) {
	tests := []struct {
		input interface{}
		want  string
	}{
		{"hello", "hello"},
		{42, "42"},
		{3.14, "3.14"},
		{true, "true"},
	}

	for _, tt := range tests {
		got, _ := transformToString(tt.input, "")
		if got != tt.want {
			t.Errorf("transformToString(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func testTransformToInteger(t *testing.T) {
	tests := []struct {
		input interface{}
		want  int
	}{
		{42, 42},
		{int32(42), 42},
		{int64(42), 42},
		{3.9, 3},
		{"42", 42},
		{true, 1},
		{false, 0},
	}

	for _, tt := range tests {
		got, _ := transformToInteger(tt.input, "")
		if got != tt.want {
			t.Errorf("transformToInteger(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func testTransformToBoolean(t *testing.T) {
	tests := []struct {
		input interface{}
		want  bool
	}{
		{true, true},
		{false, false},
		{"true", true},
		{"yes", true},
		{"false", false},
		{"no", false},
		{1, true},
		{0, false},
	}

	for _, tt := range tests {
		got, _ := transformToBoolean(tt.input, "")
		if got != tt.want {
			t.Errorf("transformToBoolean(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func testTransformToArray(t *testing.T) {
	result, _ := transformToArray([]interface{}{"a", "b"}, "")
	if len(result.([]interface{})) != 2 {
		t.Errorf("transformToArray length = %d, want 2", len(result.([]interface{})))
	}

	result, _ = transformToArray("a,b,c", "")
	arr := result.([]interface{})
	if len(arr) != 3 || arr[0] != "a" {
		t.Errorf("transformToArray string split = %v", arr)
	}
}

func testTransformToObject(t *testing.T) {
	obj := map[string]interface{}{"key": "value"}
	result, _ := transformToObject(obj, "")
	if result == nil {
		t.Error("transformToObject should preserve object")
	}

	jsonStr := `{"key":"value"}`
	result, _ = transformToObject(jsonStr, "")
	if result == nil {
		t.Error("transformToObject should parse JSON string")
	}
}

// Helper function for tests
func testError(msg string) error {
	return &TestError{msg: msg}
}

type TestError struct {
	msg string
}

func (e *TestError) Error() string {
	return e.msg
}
