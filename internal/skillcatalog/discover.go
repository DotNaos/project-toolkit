package skillcatalog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const readFileErrorFormat = "failed to read %s: %w"

type Summary struct {
	ID      string
	Title   string
	Valid   bool
	Format  string
	RootDir string
	Errors  []string
}

type parsedMarkdown struct {
	Metadata map[string]any
	Body     string
}

var frontMatterPattern = regexp.MustCompile(`(?s)^---\r?\n(.*?)\r?\n---\r?\n?`)

func Discover(skillsRoot string) ([]Summary, error) {
	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory %s: %w", skillsRoot, err)
	}

	summaries := make([]Summary, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		summary, err := inspectSkill(filepath.Join(skillsRoot, entry.Name()), entry.Name())
		if err != nil {
			return nil, err
		}

		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].ID < summaries[j].ID
	})

	return summaries, nil
}

func inspectSkill(rootDir, skillID string) (Summary, error) {
	normalizedMetadataPath := filepath.Join(rootDir, "skill.yaml")
	normalizedPromptPath := filepath.Join(rootDir, "prompt.md")
	legacySkillPath := filepath.Join(rootDir, "SKILL.md")

	hasMetadata := pathExists(normalizedMetadataPath)
	hasPrompt := pathExists(normalizedPromptPath)
	if hasMetadata || hasPrompt {
		return inspectNormalizedSkill(rootDir, skillID, normalizedMetadataPath, normalizedPromptPath)
	}

	if pathExists(legacySkillPath) {
		return inspectLegacySkill(rootDir, skillID, legacySkillPath)
	}

	return inspectMarkdownFallback(rootDir, skillID)
}

func inspectNormalizedSkill(rootDir, skillID, metadataPath, promptPath string) (Summary, error) {
	errors := []string{}
	metadata := map[string]any{}

	if pathExists(metadataPath) {
		parsed, parseErrors := readYAMLObject(metadataPath)
		metadata = parsed
		errors = append(errors, parseErrors...)
	} else {
		errors = append(errors, "missing skill.yaml")
	}

	if pathExists(promptPath) {
		content, err := os.ReadFile(promptPath)
		if err != nil {
			return Summary{}, fmt.Errorf(readFileErrorFormat, promptPath, err)
		}
		if strings.TrimSpace(string(content)) == "" {
			errors = append(errors, "prompt.md is empty")
		}
	} else {
		errors = append(errors, "missing prompt.md")
	}

	title := readString(metadata, "title")
	if title == "" {
		title = readString(metadata, "name")
	}
	if title == "" {
		title = skillID
	}

	return Summary{
		ID:      skillID,
		Title:   title,
		Valid:   len(errors) == 0,
		Format:  selectFormat(len(errors) == 0, "normalized"),
		RootDir: rootDir,
		Errors:  errors,
	}, nil
}

func inspectLegacySkill(rootDir, skillID, skillPath string) (Summary, error) {
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return Summary{}, fmt.Errorf(readFileErrorFormat, skillPath, err)
	}

	parsed := parseMarkdown(string(content))
	title := readString(parsed.Metadata, "title")
	if title == "" {
		title = readString(parsed.Metadata, "name")
	}
	if title == "" {
		title = extractHeading(parsed.Body)
	}
	if title == "" {
		title = skillID
	}

	prompt := strings.TrimSpace(parsed.Body)
	if prompt == "" {
		prompt = strings.TrimSpace(strings.Join([]string{title, readString(parsed.Metadata, "description")}, "\n\n"))
	}

	errors := []string{}
	if prompt == "" {
		errors = append(errors, "SKILL.md does not contain usable prompt content")
	}

	return Summary{
		ID:      skillID,
		Title:   title,
		Valid:   len(errors) == 0,
		Format:  selectFormat(len(errors) == 0, "legacy"),
		RootDir: rootDir,
		Errors:  errors,
	}, nil
}

func inspectMarkdownFallback(rootDir, skillID string) (Summary, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return Summary{}, fmt.Errorf("failed to read skill directory %s: %w", rootDir, err)
	}

	markdownFiles := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			markdownFiles = append(markdownFiles, entry.Name())
		}
	}
	sort.Strings(markdownFiles)

	if len(markdownFiles) != 1 {
		return Summary{
			ID:      skillID,
			Title:   skillID,
			Valid:   false,
			Format:  "invalid",
			RootDir: rootDir,
			Errors:  []string{"missing skill definition (expected skill.yaml + prompt.md, SKILL.md, or a single markdown file)"},
		}, nil
	}

	promptPath := filepath.Join(rootDir, markdownFiles[0])
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return Summary{}, fmt.Errorf(readFileErrorFormat, promptPath, err)
	}

	parsed := parseMarkdown(string(content))
	title := extractHeading(parsed.Body)
	if title == "" {
		title = skillID
	}
	prompt := strings.TrimSpace(parsed.Body)
	if prompt == "" {
		prompt = strings.TrimSpace(string(content))
	}

	errors := []string{}
	if prompt == "" {
		errors = append(errors, "markdown skill file is empty")
	}

	return Summary{
		ID:      skillID,
		Title:   title,
		Valid:   len(errors) == 0,
		Format:  selectFormat(len(errors) == 0, "markdown-fallback"),
		RootDir: rootDir,
		Errors:  errors,
	}, nil
}

func ResolveSkillsRoot() (string, error) {
	_, currentFile, _, ok := runtimeCaller()
	if !ok {
		return "", fmt.Errorf("failed to resolve package root")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	return filepath.Join(repoRoot, "skills"), nil
}

var runtimeCaller = func() (uintptr, string, int, bool) {
	return runtimeCallerImpl()
}

func runtimeCallerImpl() (uintptr, string, int, bool) {
	return runtime.Caller(0)
}

func readYAMLObject(sourcePath string) (map[string]any, []string) {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return map[string]any{}, []string{fmt.Sprintf("failed to parse %s: %s", filepath.Base(sourcePath), err.Error())}
	}

	var parsed any
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		return map[string]any{}, []string{fmt.Sprintf("failed to parse %s: %s", filepath.Base(sourcePath), err.Error())}
	}

	obj, ok := parsed.(map[string]any)
	if !ok {
		return map[string]any{}, []string{fmt.Sprintf("%s must contain a YAML object", filepath.Base(sourcePath))}
	}

	return obj, nil
}

func parseMarkdown(source string) parsedMarkdown {
	match := frontMatterPattern.FindStringSubmatch(source)
	if len(match) < 2 {
		return parsedMarkdown{Metadata: map[string]any{}, Body: source}
	}

	metadata := map[string]any{}
	if err := yaml.Unmarshal([]byte(match[1]), &metadata); err != nil {
		metadata = map[string]any{}
	}

	return parsedMarkdown{
		Metadata: metadata,
		Body:     source[len(match[0]):],
	}
}

func extractHeading(source string) string {
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}

	return ""
}

func readString(record map[string]any, key string) string {
	value, ok := record[key]
	if !ok {
		return ""
	}

	text, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(text)
}

func pathExists(targetPath string) bool {
	_, err := os.Stat(targetPath)
	return err == nil
}

func selectFormat(valid bool, format string) string {
	if valid {
		return format
	}

	return "invalid"
}
