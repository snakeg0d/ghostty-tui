package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// shaderFile is one GLSL shader discovered on disk.
type shaderFile struct {
	path   string
	name   string
	rel    string
	kind   string
	source string
}

// shaderExtensions are the file suffixes treated as custom shaders.
var shaderExtensions = map[string]bool{".glsl": true}

// loadShaders walks the shader directories for GLSL files.
func loadShaders(dirs []themeDir) ([]shaderFile, []string) {
	var warnings []string
	var shaders []shaderFile
	seen := make(map[string]bool)
	for _, dir := range dirs {
		info, err := os.Stat(dir.path)
		if err != nil || !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(dir.path, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			if !shaderExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
				return nil
			}
			abs, absErr := filepath.Abs(path)
			if absErr == nil {
				path = abs
			}
			if seen[path] {
				return nil
			}
			seen[path] = true
			rel, relErr := filepath.Rel(dir.path, path)
			if relErr != nil {
				rel = entry.Name()
			}
			kind := filepath.Dir(rel)
			if kind == "." {
				kind = ""
			}
			shaders = append(shaders, shaderFile{
				path:   path,
				name:   shaderDisplayName(entry.Name()),
				rel:    rel,
				kind:   kind,
				source: dir.source,
			})
			return nil
		})
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("shader directory %s could not be read: %v", dir.path, err))
		}
	}
	sort.SliceStable(shaders, func(i, j int) bool {
		if shaders[i].source != shaders[j].source {
			return shaders[i].source < shaders[j].source
		}
		return strings.ToLower(shaders[i].name) < strings.ToLower(shaders[j].name)
	})
	return shaders, warnings
}

// shaderDisplayName turns a shader file name into a readable label.
func shaderDisplayName(fileName string) string {
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	words := strings.Fields(base)
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		if word == strings.ToUpper(word) && len(word) > 1 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	if len(words) == 0 {
		return fileName
	}
	return strings.Join(words, " ")
}

// noneEffect is the explicit way to clear the managed shader stack.
func noneEffect() Choice {
	return Choice{
		ID:          "effect:none",
		Name:        "None",
		Description: "No custom shader. Choosing None resets the managed shader stack; nothing else is touched.",
		Source:      "Ghostty Studio",
		Category:    "effect",
		Tags:        []string{"reset"},
		Values:      Values{"custom-shader": []string{""}},
	}
}

// effectChoices converts discovered shaders into catalog choices.
func effectChoices(shaders []shaderFile) []Choice {
	choices := []Choice{noneEffect()}
	ambiguous := map[string]int{}
	for _, shader := range shaders {
		ambiguous[slugify(shader.name)]++
	}
	for _, shader := range shaders {
		id := slugify(shader.name)
		if ambiguous[id] > 1 {
			id = slugify(shader.kind) + "/" + id
		}
		category := "effect"
		if shader.kind != "" {
			category = "effect"
		}
		description := fmt.Sprintf("%s · %s · %s", shader.source, shader.rel, "GLSL compiles in the renderer only; validate-config checks the path, not the shader")
		choices = append(choices, Choice{
			ID:          "effect:" + id,
			Name:        shader.name,
			Description: description,
			Source:      shader.source,
			Category:    category,
			Tags:        effectTags(shader),
			Values: Values{
				"custom-shader":           []string{"", shader.path},
				"custom-shader-animation": []string{"false"},
			},
			Installed: true,
		})
	}
	return choices
}

func effectTags(shader shaderFile) []string {
	tags := []string{"glsl"}
	if shader.kind != "" {
		tags = append(tags, slugify(shader.kind))
	}
	switch shader.source {
	case "Bundled effect":
		tags = append(tags, "bundled")
	default:
		tags = append(tags, "user")
	}
	return tags
}
