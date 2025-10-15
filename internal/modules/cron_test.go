package modules

import (
	"testing"
)

func TestCronModule_GetName(t *testing.T) {
	module := NewCronModule()
	expected := "cron"
	if got := module.GetName(); got != expected {
		t.Errorf("GetName() = %v, want %v", got, expected)
	}
}

func TestCronModule_GetDescription(t *testing.T) {
	module := NewCronModule()
	expected := "Manage cron jobs and crontab files"
	if got := module.GetDescription(); got != expected {
		t.Errorf("GetDescription() = %v, want %v", got, expected)
	}
}

func TestNewCronModule(t *testing.T) {
	module := NewCronModule()
	if module == nil {
		t.Fatal("NewCronModule() returned nil")
	}
	if module.GetName() != "cron" {
		t.Errorf("NewCronModule() name = %v, want cron", module.GetName())
	}
}

func TestCronModule_Validate(t *testing.T) {
	module := NewCronModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid job operation with present state",
			args: map[string]interface{}{
				"operation": "job",
				"name":      "test-job",
				"job":       "* * * * * /usr/bin/test",
				"state":     "present",
			},
			wantErr: false,
		},
		{
			name: "valid job operation with absent state",
			args: map[string]interface{}{
				"operation": "job",
				"name":      "test-job",
				"state":     "absent",
			},
			wantErr: false,
		},
		{
			name: "job operation missing name",
			args: map[string]interface{}{
				"operation": "job",
				"job":       "* * * * * /usr/bin/test",
			},
			wantErr: true,
			errMsg:  "name parameter is required",
		},
		{
			name: "job operation present state missing job",
			args: map[string]interface{}{
				"operation": "job",
				"name":      "test-job",
				"state":     "present",
			},
			wantErr: true,
			errMsg:  "job parameter is required when state is present",
		},
		{
			name: "valid file operation with present state",
			args: map[string]interface{}{
				"operation": "file",
				"content":   "* * * * * /usr/bin/test",
				"state":     "present",
			},
			wantErr: false,
		},
		{
			name: "valid file operation with absent state",
			args: map[string]interface{}{
				"operation": "file",
				"state":     "absent",
			},
			wantErr: false,
		},
		{
			name: "file operation present state missing content",
			args: map[string]interface{}{
				"operation": "file",
				"state":     "present",
			},
			wantErr: true,
			errMsg:  "content parameter is required when state is present",
		},
		{
			name: "valid system operation with present state",
			args: map[string]interface{}{
				"operation": "system",
				"name":      "test-cron",
				"content":   "* * * * * root /usr/bin/test",
				"state":     "present",
			},
			wantErr: false,
		},
		{
			name: "valid system operation with absent state",
			args: map[string]interface{}{
				"operation": "system",
				"name":      "test-cron",
				"state":     "absent",
			},
			wantErr: false,
		},
		{
			name: "system operation missing name",
			args: map[string]interface{}{
				"operation": "system",
				"content":   "* * * * * root /usr/bin/test",
				"state":     "present",
			},
			wantErr: true,
			errMsg:  "name parameter is required",
		},
		{
			name: "system operation present state missing content",
			args: map[string]interface{}{
				"operation": "system",
				"name":      "test-cron",
				"state":     "present",
			},
			wantErr: true,
			errMsg:  "content parameter is required when state is present",
		},
		{
			name: "valid list operation",
			args: map[string]interface{}{
				"operation": "list",
			},
			wantErr: false,
		},
		{
			name: "invalid operation",
			args: map[string]interface{}{
				"operation": "invalid",
			},
			wantErr: true,
			errMsg:  "invalid operation: invalid",
		},
		{
			name: "default operation (job) with valid args",
			args: map[string]interface{}{
				"name": "test-job",
				"job":  "* * * * * /usr/bin/test",
			},
			wantErr: false,
		},
		{
			name: "default operation (job) missing name",
			args: map[string]interface{}{
				"job": "* * * * * /usr/bin/test",
			},
			wantErr: true,
			errMsg:  "name parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestCronModule_parseCrontab(t *testing.T) {
	module := NewCronModule()

	tests := []struct {
		name     string
		crontab  string
		expected map[string]string
	}{
		{
			name:     "empty crontab",
			crontab:  "",
			expected: map[string]string{},
		},
		{
			name: "single job with Onigirazu marker",
			crontab: `# Onigirazu: backup-job
0 2 * * * /usr/bin/backup.sh`,
			expected: map[string]string{
				"backup-job": "0 2 * * * /usr/bin/backup.sh",
			},
		},
		{
			name: "single job with Ansible marker",
			crontab: `# Ansible: backup-job
0 2 * * * /usr/bin/backup.sh`,
			expected: map[string]string{
				"backup-job": "0 2 * * * /usr/bin/backup.sh",
			},
		},
		{
			name: "multiple jobs",
			crontab: `# Onigirazu: backup-job
0 2 * * * /usr/bin/backup.sh
# Onigirazu: cleanup-job
0 3 * * * /usr/bin/cleanup.sh`,
			expected: map[string]string{
				"backup-job":  "0 2 * * * /usr/bin/backup.sh",
				"cleanup-job": "0 3 * * * /usr/bin/cleanup.sh",
			},
		},
		{
			name: "jobs with comments and empty lines",
			crontab: `# Some comment
# Onigirazu: backup-job
0 2 * * * /usr/bin/backup.sh

# Another comment
# Onigirazu: cleanup-job
0 3 * * * /usr/bin/cleanup.sh`,
			expected: map[string]string{
				"backup-job":  "0 2 * * * /usr/bin/backup.sh",
				"cleanup-job": "0 3 * * * /usr/bin/cleanup.sh",
			},
		},
		{
			name: "job without marker (ignored)",
			crontab: `0 2 * * * /usr/bin/backup.sh
# Onigirazu: cleanup-job
0 3 * * * /usr/bin/cleanup.sh`,
			expected: map[string]string{
				"cleanup-job": "0 3 * * * /usr/bin/cleanup.sh",
			},
		},
		{
			name: "marker with extra spaces",
			crontab: `# Onigirazu:   backup-job
0 2 * * * /usr/bin/backup.sh`,
			expected: map[string]string{
				"backup-job": "0 2 * * * /usr/bin/backup.sh",
			},
		},
		{
			name: "mixed Ansible and Onigirazu markers",
			crontab: `# Ansible: old-job
0 1 * * * /usr/bin/old.sh
# Onigirazu: new-job
0 2 * * * /usr/bin/new.sh`,
			expected: map[string]string{
				"old-job": "0 1 * * * /usr/bin/old.sh",
				"new-job": "0 2 * * * /usr/bin/new.sh",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := module.parseCrontab(tt.crontab)

			if len(result) != len(tt.expected) {
				t.Errorf("parseCrontab() returned %d jobs, expected %d", len(result), len(tt.expected))
			}

			for name, job := range tt.expected {
				if result[name] != job {
					t.Errorf("parseCrontab() job[%s] = %q, expected %q", name, result[name], job)
				}
			}
		})
	}
}

func TestCronModule_buildCrontab(t *testing.T) {
	module := NewCronModule()

	tests := []struct {
		name     string
		jobs     map[string]string
		contains []string
	}{
		{
			name: "empty jobs",
			jobs: map[string]string{},
			contains: []string{
				"# Managed by Onigirazu",
			},
		},
		{
			name: "single job",
			jobs: map[string]string{
				"backup-job": "0 2 * * * /usr/bin/backup.sh",
			},
			contains: []string{
				"# Managed by Onigirazu",
				"# Onigirazu: backup-job",
				"0 2 * * * /usr/bin/backup.sh",
			},
		},
		{
			name: "multiple jobs",
			jobs: map[string]string{
				"backup-job":  "0 2 * * * /usr/bin/backup.sh",
				"cleanup-job": "0 3 * * * /usr/bin/cleanup.sh",
			},
			contains: []string{
				"# Managed by Onigirazu",
				"# Onigirazu: backup-job",
				"0 2 * * * /usr/bin/backup.sh",
				"# Onigirazu: cleanup-job",
				"0 3 * * * /usr/bin/cleanup.sh",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := module.buildCrontab(tt.jobs)

			for _, expected := range tt.contains {
				if !containsString(result, expected) {
					t.Errorf("buildCrontab() result does not contain %q\nGot: %s", expected, result)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
