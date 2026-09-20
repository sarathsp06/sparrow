package recipes

import (
	"embed"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var catalog embed.FS

// All returns the shipped recipes embedded in the Sparrow binary.
func All() ([]Recipe, error) {
	entries, err := catalog.ReadDir(".")
	if err != nil {
		return nil, err
	}

	out := make([]Recipe, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".yaml") {
			continue
		}
		raw, err := catalog.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var recipe Recipe
		if err := yaml.Unmarshal(raw, &recipe); err != nil {
			return nil, err
		}
		if err := recipe.Validate(); err != nil {
			return nil, err
		}
		if recipe.Version == 1 && recipe.Name != "" && recipe.Webhook.URL != "" {
			out = append(out, recipe)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
