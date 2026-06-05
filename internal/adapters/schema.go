package adapters

// YAMLAdapter represents a declarative adapter definition loaded from YAML.
type YAMLAdapter struct {
	Name       string               `yaml:"name"`
	ConfigPath string               `yaml:"config_path"` // e.g., ".config/myapp"
	Categories []YAMLCategoryPattern `yaml:"categories"`
}

// YAMLCategoryPattern maps a category name to scan patterns for discovering
// files and directories under the adapter's config path.
type YAMLCategoryPattern struct {
	Name      string   `yaml:"name"`       // e.g., "skills"
	SubPath   string   `yaml:"sub_path"`   // relative to config_path
	IsDir     bool     `yaml:"is_dir"`     // true when sub_path is a directory to scan
	Patterns  []string `yaml:"patterns"`   // glob patterns for matching files
	RootFiles []string `yaml:"root_files"` // explicit file names at config root
}
