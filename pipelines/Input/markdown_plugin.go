package Input

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// MarkdownPlugin implements a Markdown file input plugin for Mimir AIP
type MarkdownPlugin struct {
	name    string
	version string
}

// NewMarkdownPlugin creates a new Markdown plugin instance
func NewMarkdownPlugin() *MarkdownPlugin {
	return &MarkdownPlugin{
		name:    "MarkdownPlugin",
		version: "1.0.0",
	}
}

// ExecuteStep reads and parses a Markdown file
func (p *MarkdownPlugin) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	fmt.Printf("Executing %s step: %s\n", p.name, stepConfig.Name)

	config := stepConfig.Config

	// Extract configuration
	filePath, ok := config["file_path"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("file_path is required in config")
	}

	// Optional configuration
	extractSections := true // default
	if es, ok := config["extract_sections"].(bool); ok {
		extractSections = es
	}

	extractLinks := true // default
	if el, ok := config["extract_links"].(bool); ok {
		extractLinks = el
	}

	extractMetadata := true // default
	if em, ok := config["extract_metadata"].(bool); ok {
		extractMetadata = em
	}

	// Validate configuration
	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Read and parse Markdown
	data, err := p.parseMarkdown(filePath, extractSections, extractLinks, extractMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Markdown: %w", err)
	}

	// Create result
	result := map[string]any{
		"file_path":        filePath,
		"extract_sections": extractSections,
		"extract_links":    extractLinks,
		"extract_metadata": extractMetadata,
		"content":          data.Content,
		"sections":         data.Sections,
		"links":            data.Links,
		"metadata":         data.Metadata,
		"word_count":       data.WordCount,
		"line_count":       data.LineCount,
		"parsed_at":        data.ParsedAt,
	}

	context := pipelines.NewPluginContext()
	context.Set(stepConfig.Output, result)
	return context, nil
}

// MarkdownData represents parsed Markdown data
type MarkdownData struct {
	Content   string            `json:"content"`
	Sections  []MarkdownSection `json:"sections"`
	Links     []MarkdownLink    `json:"links"`
	Metadata  map[string]any    `json:"metadata"`
	WordCount int               `json:"word_count"`
	LineCount int               `json:"line_count"`
	ParsedAt  string            `json:"parsed_at"`
}

// MarkdownSection represents a section in the Markdown document
type MarkdownSection struct {
	Level     int    `json:"level"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
}

// MarkdownLink represents a link in the Markdown document
type MarkdownLink struct {
	Text    string `json:"text"`
	URL     string `json:"url"`
	Title   string `json:"title,omitempty"`
	IsImage bool   `json:"is_image"`
	Line    int    `json:"line"`
}

// parseMarkdown reads and parses a Markdown file
func (p *MarkdownPlugin) parseMarkdown(filePath string, extractSections, extractLinks, extractMetadata bool) (*MarkdownData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := strings.Join(lines, "\n")

	data := &MarkdownData{
		Content:  content,
		Sections: []MarkdownSection{},
		Links:    []MarkdownLink{},
		Metadata: make(map[string]any),
		ParsedAt: time.Now().Format("2006-01-02T15:04:05Z07:00"),
	}

	// Count words and lines
	data.WordCount = len(strings.Fields(content))
	data.LineCount = len(lines)

	// Extract metadata (frontmatter)
	if extractMetadata {
		data.Metadata = p.extractFrontmatter(lines)
		// Remove frontmatter from content
		if len(data.Metadata) > 0 {
			start := strings.Index(content, "---")
			if start >= 0 {
				end := strings.Index(content[start+3:], "---")
				if end >= 0 {
					content = content[start+3+end+3:]
					data.Content = strings.TrimSpace(content)
				}
			}
		}
	}

	// Extract sections
	if extractSections {
		data.Sections = p.extractSections(lines)
	}

	// Extract links
	if extractLinks {
		data.Links = p.extractLinks(lines)
	}

	return data, nil
}

// extractFrontmatter extracts YAML frontmatter from Markdown
func (p *MarkdownPlugin) extractFrontmatter(lines []string) map[string]any {
	metadata := make(map[string]any)

	// Check for frontmatter delimiter
	if len(lines) < 3 || lines[0] != "---" {
		return metadata
	}

	// Find end of frontmatter
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return metadata
	}

	// Parse simple key-value pairs (basic implementation)
	for i := 1; i < endIdx; i++ {
		line := strings.TrimSpace(lines[i])
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Remove quotes if present
				value = strings.Trim(value, "\"'")
				metadata[key] = value
			}
		}
	}

	return metadata
}

// extractSections extracts headings and their content
func (p *MarkdownPlugin) extractSections(lines []string) []MarkdownSection {
	var sections []MarkdownSection
	var currentSection *MarkdownSection

	for i, line := range lines {
		// Check for headings (# ## ###)
		if strings.HasPrefix(line, "#") {
			// Count leading # characters
			level := 0
			for _, char := range line {
				if char == '#' {
					level++
				} else {
					break
				}
			}

			if level > 0 && level <= 6 {
				// Save previous section
				if currentSection != nil {
					currentSection.LineEnd = i - 1
					sections = append(sections, *currentSection)
				}

				// Start new section
				title := strings.TrimSpace(line[level:])
				currentSection = &MarkdownSection{
					Level:     level,
					Title:     title,
					LineStart: i,
					LineEnd:   i,
				}
			}
		} else if currentSection != nil {
			// Add content to current section
			if currentSection.Content == "" {
				currentSection.Content = line
			} else {
				currentSection.Content += "\n" + line
			}
		}
	}

	// Save last section
	if currentSection != nil {
		currentSection.LineEnd = len(lines) - 1
		sections = append(sections, *currentSection)
	}

	return sections
}

// extractLinks extracts links and images from Markdown
func (p *MarkdownPlugin) extractLinks(lines []string) []MarkdownLink {
	var links []MarkdownLink

	// Regex patterns for different link types
	linkPatterns := []*regexp.Regexp{
		// [text](url "title") - inline links
		regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)(?:\s+"([^"]+)")?\)`),
		// [text](url) - inline links without title
		regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`),
		// ![alt](url "title") - images
		regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)(?:\s+"([^"]+)")?\)`),
		// ![alt](url) - images without title
		regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)\)`),
	}

	for lineNum, line := range lines {
		for _, pattern := range linkPatterns {
			matches := pattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				link := MarkdownLink{
					Line: lineNum + 1,
				}

				if strings.HasPrefix(match[0], "!") {
					// Image link
					link.IsImage = true
					link.Text = match[1] // alt text
				} else {
					// Regular link
					link.Text = match[1] // link text
				}

				link.URL = match[2] // URL
				if len(match) > 3 && match[3] != "" {
					link.Title = match[3] // title (if present)
				}

				links = append(links, link)
			}
		}
	}

	return links
}

// GetPluginType returns the plugin type
func (p *MarkdownPlugin) GetPluginType() string {
	return "Input"
}

// GetPluginName returns the plugin name
func (p *MarkdownPlugin) GetPluginName() string {
	return "markdown"
}

// ValidateConfig validates the plugin configuration
func (p *MarkdownPlugin) ValidateConfig(config map[string]any) error {
	if _, ok := config["file_path"].(string); !ok {
		return fmt.Errorf("file_path is required and must be a string")
	}

	return nil
}
