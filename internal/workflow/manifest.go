// Package workflow reads the shared Operator Context manifest (workflow.toml).
//
// The manifest's canonical source is farishan/agent/workflow.toml; the runtime
// copy (often a symlink) lives at ~/.config/fhl/workflow.toml. Consumers read
// paths and conventions from it instead of hard-coding them.
//
// usermd-check does NOT validate this manifest (--workflow is prohibited); use
// agent/tools/workflow-manifest.py for local validation/sync.
package workflow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Meta holds manifest metadata.
type Meta struct {
	Name            string `toml:"name"`
	Version         string `toml:"version"`
	Schema          int    `toml:"schema"`
	Status          string `toml:"status"`
	CanonicalRepo   string `toml:"canonical_repo"`
	CanonicalPath   string `toml:"canonical_path"`
	Updated         string `toml:"updated"`
}

// Identity holds identity-related paths.
type Identity struct {
	SoulPath       string `toml:"soul_path"`
	UserMDPath     string `toml:"user_md_path"`
	AgentsMD       string `toml:"agents_md"`
	MachineProfile string `toml:"machine_profile"`
	OpenClawSplit  bool   `toml:"openclaw_split"`
}

// Memory holds OptMem configuration.
type Memory struct {
	Engine    string `toml:"engine"`
	ToolMac   string `toml:"tool_mac"`
	ToolLinux string `toml:"tool_linux"`
	Store     string `toml:"store"`
	Seed      string `toml:"seed"`
	WakeReq   bool   `toml:"wake_required"`
}

// Spec holds the spec engine configuration.
type Spec struct {
	Engine string `toml:"engine"`
	Root   string `toml:"root"`
}

// Harness holds harness configuration.
type Harness struct {
	Primary          string   `toml:"primary"`
	TargetPrimary    string   `toml:"target_primary"`
	MigrationStatus  string   `toml:"migration_status"`
	Available        []string `toml:"available"`
}

// Mode holds work/personal mode configuration.
type Mode struct {
	Default      string `toml:"default"`
	PersonalCore string `toml:"personal_core"`
	WorkOverlay  string `toml:"work_overlay"`
}

// Paths holds personal core paths.
type Paths struct {
	AgentRepo   string `toml:"agent_repo"`
	AgentTools  string `toml:"agent_tools"`
	UserMDRepo  string `toml:"user_md_repo"`
	CockpitRepo string `toml:"cockpit_repo"`
	EssaysRepo  string `toml:"essays_repo"`
	Principles  string `toml:"principles_repo"`
	SkillsDir   string `toml:"skills_dir"`
}

// Conventions holds convention flags.
type Conventions struct {
	Principles        string `toml:"principles"`
	SearchGuard       bool   `toml:"search_guard"`
	BroadSearchBlocked bool  `toml:"broad_search_blocked"`
	TrashRule         string `toml:"trash_rule"`
	CommitAuthor      string `toml:"commit_author"`
	WorkPersonalSplit bool   `toml:"work_personal_split"`
}

// Github holds GitHub account info.
type Github struct {
	PersonalAccount string `toml:"personal_account"`
	PersonalRemote  string `toml:"personal_remote"`
	WorkAccount     string `toml:"work_account"`
	WorkRemote      string `toml:"work_remote"`
}

// Providers holds provider configuration.
type Providers struct {
	Default   string   `toml:"default"`
	Fallbacks []string `toml:"fallbacks"`
	Swappable bool     `toml:"swappable"`
}

// Cockpit holds cockpit integration paths.
type Cockpit struct {
	Config            string `toml:"config"`
	UserMDPathKey     string `toml:"user_md_path_key"`
	DoctorRunsUsermd  bool   `toml:"doctor_runs_usermd_check"`
	LaunchExports     string `toml:"launch_exports"`
	OperatorPanel     bool   `toml:"operator_panel"`
}

// UserMD holds user-md integration.
type UserMD struct {
	Binary string   `toml:"binary"`
	Checks []string `toml:"checks"`
}

// OpenSpec holds spec-engine adoption tracking.
type OpenSpec struct {
	InstalledIn []string `toml:"installed_in"`
	PendingIn   []string `toml:"pending_in"`
}

// Manifest is the parsed workflow.toml.
type Manifest struct {
	Meta         Meta         `toml:"meta"`
	Identity     Identity     `toml:"identity"`
	Memory       Memory       `toml:"memory"`
	Spec         Spec         `toml:"spec"`
	Harness      Harness      `toml:"harness"`
	Mode         Mode         `toml:"mode"`
	Paths        Paths        `toml:"paths"`
	Conventions  Conventions  `toml:"conventions"`
	Github       Github       `toml:"github"`
	Providers    Providers    `toml:"providers"`
	Cockpit      Cockpit      `toml:"cockpit"`
	UserMD       UserMD       `toml:"usermd"`
	OpenSpec     OpenSpec     `toml:"openspec"`
}

// Path returns the runtime manifest path.
func Path() string {
	if v := os.Getenv("FHL_WORKFLOW_PATH"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "fhl", "workflow.toml")
}

// Load reads and parses the manifest from the given path (or the default).
func Load(path string) (Manifest, error) {
	if path == "" {
		path = Path()
	}
	var m Manifest
	if _, err := toml.DecodeFile(path, &m); err != nil {
		return m, fmt.Errorf("decode workflow manifest %s: %w", path, err)
	}
	return m, nil
}

// Expand expands a leading ~ in a path.
func Expand(p string) string {
	if len(p) > 0 && p[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[1:])
	}
	return p
}
