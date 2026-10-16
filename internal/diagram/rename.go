package diagram

// renamedCSSVariables maps each deprecated CSS variable name to its current
// name. The deprecated names are still accepted until the next major
// version.
var renamedCSSVariables = map[string]string{
	"--background-color":      "--diagram-bg-color",
	"--text-color":            "--diagram-text-color",
	"--border-color":          "--diagram-border-color",
	"--font-family-normal":    "--diagram-font-family",
	"--trigger-color":         "--trigger-bg-color",
	"--command-color":         "--command-bg-color",
	"--event-color":           "--event-bg-color",
	"--exception-color":       "--exception-bg-color",
	"--view-color":            "--view-bg-color",
	"--font-size-slicename":   "--slicename-font-size",
	"--font-weight-slicename": "--slicename-font-weight",
	"--font-size-swimlane":    "--swimlane-font-size",
	"--font-weight-swimlane":  "--swimlane-font-weight",
	"--font-size-testname":    "--testname-font-size",
	"--font-weight-testname":  "--testname-font-weight",
	"--font-size-label":       "--label-font-size",
	"--font-weight-label":     "--label-font-weight",
	"--font-family-props":     "--props-font-family",
	"--font-size-props":       "--props-font-size",
	"--font-weight-props":     "--props-font-weight",
}

// RenamedCSSVariable returns the current name of a deprecated CSS variable
// name. It reports false when name is not deprecated.
func RenamedCSSVariable(name string) (string, bool) {
	current, ok := renamedCSSVariables[name]
	return current, ok
}

// renameCSSVariables returns a copy of overrides where every deprecated
// name is replaced by its current name. When both names are set, the
// current name wins.
func renameCSSVariables(overrides map[string]string) map[string]string {
	out := make(map[string]string, len(overrides))
	for k, v := range overrides {
		if current, ok := renamedCSSVariables[k]; ok {
			if _, set := overrides[current]; set {
				continue
			}
			k = current
		}
		out[k] = v
	}
	return out
}
