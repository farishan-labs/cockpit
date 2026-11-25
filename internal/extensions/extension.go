package extensions

import "net/http"

// Tab describes a UI tab contributed by an extension.
type Tab struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
	Route string `json:"route"`
}

// Command describes a CLI command contributed by an extension.
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Extension is implemented by native (in-process) extensions.
type Extension interface {
	Name() string
	Version() string
	Tabs() []Tab
	Commands() []Command
	Health() error
	Mount(mux *http.ServeMux, prefix string) error
}

// Kind identifies how an extension runs.
type Kind string

const (
	KindNative Kind = "native"
	KindRemote Kind = "remote"
)

// Manifest is the on-disk/HTTP manifest schema.
type Manifest struct {
	Name     string    `json:"name"`
	Version  string    `json:"version,omitempty"`
	Kind     Kind      `json:"kind,omitempty"`
	Provides []string  `json:"provides,omitempty"`
	Entry    string    `json:"entry,omitempty"`
	APIBase  string    `json:"api_base,omitempty"`
	Web      string    `json:"web,omitempty"`
	Tabs     []Tab     `json:"tabs,omitempty"`
	Commands []Command `json:"commands,omitempty"`
	Enabled  *bool     `json:"enabled,omitempty"`
}

// EffectiveKind returns the kind, defaulting to remote.
func (m Manifest) EffectiveKind() Kind {
	if m.Kind == "" {
		return KindRemote
	}
	return m.Kind
}

// IsEnabled returns enabled, defaulting to true.
func (m Manifest) IsEnabled() bool {
	if m.Enabled == nil {
		return true
	}
	return *m.Enabled
}
