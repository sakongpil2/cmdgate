package allowlist

import (
	"testing"
)

func TestFindExactMatch(t *testing.T) {
	input := `
commands:
  - id: restart-kubelet
    cmd: "systemctl restart kubelet"
  - id: stop-kubelet
    cmd: "systemctl stop kubelet"
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd, ok := cfg.FindCommand([]string{"systemctl", "restart", "kubelet"})
	if !ok {
		t.Fatalf("expected match")
	}
	if cmd.ID != "restart-kubelet" {
		t.Errorf("id = %q, want restart-kubelet", cmd.ID)
	}
}

func TestFindNoMatch(t *testing.T) {
	input := `
commands:
  - id: restart-kubelet
    cmd: "systemctl restart kubelet"
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := cfg.FindCommand([]string{"systemctl", "restart", "sshd"})
	if ok {
		t.Errorf("expected no match")
	}
}

func TestFindWithPlaceholderWildcard(t *testing.T) {
	input := `
commands:
  - id: journalctl-lines
    cmd: "journalctl -u kubelet -n <number:lines> --no-pager"
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd, ok := cfg.FindCommand([]string{"journalctl", "-u", "kubelet", "-n", "50", "--no-pager"})
	if !ok {
		t.Fatalf("expected match")
	}
	if cmd.ID != "journalctl-lines" {
		t.Errorf("id = %q, want journalctl-lines", cmd.ID)
	}
}

func TestFindWithPlaceholder(t *testing.T) {
	input := `
commands:
  - id: journalctl-lines
    cmd: "journalctl -u kubelet -n <number:lines> --no-pager"
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd, placeholders, ok := cfg.FindCommandWithPlaceholders([]string{"journalctl", "-u", "kubelet", "-n", "50", "--no-pager"})
	if !ok {
		t.Fatalf("expected match")
	}
	if cmd.ID != "journalctl-lines" {
		t.Errorf("id = %q, want journalctl-lines", cmd.ID)
	}
	if len(placeholders) != 1 {
		t.Fatalf("placeholders = %d, want 1", len(placeholders))
	}
	if placeholders[0].Name != "lines" || placeholders[0].Value != "50" {
		t.Errorf("placeholder = %+v, want lines=50", placeholders[0])
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Config
		wantErr bool
	}{
		{
			name: "valid minimal allowlist",
			input: `
version: 1.0.0
mode: allowlist-only
commands:
  - id: systemctl-restart-kubelet
    desc: kubelet restart
    cmd: "systemctl restart kubelet"
`,
			want: &Config{
				Version: "1.0.0",
				Mode:    "allowlist-only",
				Commands: []Command{
					{
						ID:   "systemctl-restart-kubelet",
						Desc: "kubelet restart",
						Cmd:  "systemctl restart kubelet",
					},
				},
			},
		},
		{
			name: "allowlist with matchers",
			input: `
version: 1.1.0
mode: allowlist-only
commands:
  - id: install-k8s-rpms
    desc: install k8s rpms
    cmd: "rpm -ivh <rpmFiles:k8s-rpms>"
matchers:
  k8s-rpms:
    type: rpmFiles
    multiple: true
    metadataNameIn:
      - kubelet
      - kubectl
`,
			want: &Config{
				Version: "1.1.0",
				Mode:    "allowlist-only",
				Commands: []Command{
					{
						ID:   "install-k8s-rpms",
						Desc: "install k8s rpms",
						Cmd:  "rpm -ivh <rpmFiles:k8s-rpms>",
					},
				},
				Matchers: Matchers{
					"k8s-rpms": MatcherDef{
						Type:           "rpmFiles",
						Multiple:       true,
						MetadataNameIn: []string{"kubelet", "kubectl"},
					},
				},
			},
		},
		{
			name:    "invalid YAML",
			input:   "version: [",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Version != tt.want.Version {
				t.Errorf("version = %q, want %q", cfg.Version, tt.want.Version)
			}
			if cfg.Mode != tt.want.Mode {
				t.Errorf("mode = %q, want %q", cfg.Mode, tt.want.Mode)
			}
			if len(cfg.Commands) != len(tt.want.Commands) {
				t.Fatalf("commands length = %d, want %d", len(cfg.Commands), len(tt.want.Commands))
			}
			for i, gotCmd := range cfg.Commands {
				wantCmd := tt.want.Commands[i]
				if gotCmd.ID != wantCmd.ID {
					t.Errorf("command[%d].id = %q, want %q", i, gotCmd.ID, wantCmd.ID)
				}
				if gotCmd.Desc != wantCmd.Desc {
					t.Errorf("command[%d].desc = %q, want %q", i, gotCmd.Desc, wantCmd.Desc)
				}
				if gotCmd.Cmd != wantCmd.Cmd {
					t.Errorf("command[%d].cmd = %q, want %q", i, gotCmd.Cmd, wantCmd.Cmd)
				}
			}
			if len(cfg.Matchers) != len(tt.want.Matchers) {
				t.Fatalf("matchers length = %d, want %d", len(cfg.Matchers), len(tt.want.Matchers))
			}
			for name, wantMatcher := range tt.want.Matchers {
				gotMatcher, ok := cfg.Matchers[name]
				if !ok {
					t.Errorf("matcher %q missing", name)
					continue
				}
				if gotMatcher.Type != wantMatcher.Type {
					t.Errorf("matcher[%q].type = %q, want %q", name, gotMatcher.Type, wantMatcher.Type)
				}
				if gotMatcher.Multiple != wantMatcher.Multiple {
					t.Errorf("matcher[%q].multiple = %v, want %v", name, gotMatcher.Multiple, wantMatcher.Multiple)
				}
				if len(gotMatcher.MetadataNameIn) != len(wantMatcher.MetadataNameIn) {
					t.Errorf("matcher[%q].metadataNameIn length = %d, want %d", name, len(gotMatcher.MetadataNameIn), len(wantMatcher.MetadataNameIn))
					continue
				}
				for i, gotName := range gotMatcher.MetadataNameIn {
					if gotName != wantMatcher.MetadataNameIn[i] {
						t.Errorf("matcher[%q].metadataNameIn[%d] = %q, want %q", name, i, gotName, wantMatcher.MetadataNameIn[i])
					}
				}
			}
		})
	}
}
