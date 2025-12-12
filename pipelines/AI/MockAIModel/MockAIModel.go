package mockaimodel

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Mimir-AIP/Mimir-AIP-Go/pipelines"
)

// MockAIModelPlugin provides mock AI model responses for testing
type MockAIModelPlugin struct {
	modelName string
}

// MockEchoModel returns input text with simple modifications
type MockEchoModel struct {
	MockAIModelPlugin
}

// MockSummaryModel returns a mock summary of the input
type MockSummaryModel struct {
	MockAIModelPlugin
}

// MockCreativeModel returns creative mock responses
type MockCreativeModel struct {
	MockAIModelPlugin
}

// NewMockEchoModel creates a new mock echo model
func NewMockEchoModel() *MockEchoModel {
	return &MockEchoModel{
		MockAIModelPlugin: MockAIModelPlugin{modelName: "echo"},
	}
}

// NewMockSummaryModel creates a new mock summary model
func NewMockSummaryModel() *MockSummaryModel {
	return &MockSummaryModel{
		MockAIModelPlugin: MockAIModelPlugin{modelName: "summary"},
	}
}

// NewMockCreativeModel creates a new mock creative model
func NewMockCreativeModel() *MockCreativeModel {
	return &MockCreativeModel{
		MockAIModelPlugin: MockAIModelPlugin{modelName: "creative"},
	}
}

// ExecuteStep for MockEchoModel
func (m *MockEchoModel) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	config := stepConfig.Config

	// Get input text
	input, ok := config["input"].(string)
	if !ok || input == "" {
		return nil, fmt.Errorf("input text is required")
	}

	// Get model parameters
	temperature := 0.7
	if temp, ok := config["temperature"].(float64); ok {
		temperature = temp
	}

	maxTokens := 100
	if tokens, ok := config["max_tokens"].(float64); ok {
		maxTokens = int(tokens)
	}

	// Simulate processing time
	processingTime := time.Duration(100+rand.Intn(200)) * time.Millisecond
	time.Sleep(processingTime)

	// Generate mock response based on temperature
	var response string
	if temperature < 0.3 {
		// Low creativity - simple echo with minor modifications
		response = m.generateConservativeResponse(input, maxTokens)
	} else if temperature < 0.7 {
		// Medium creativity - moderate modifications
		response = m.generateModerateResponse(input, maxTokens)
	} else {
		// High creativity - significant modifications
		response = m.generateCreativeResponse(input, maxTokens)
	}

	result := pipelines.NewPluginContext()
	result.Set(stepConfig.Output, map[string]any{
		"response":        response,
		"model":           "mock-echo-v1",
		"processing_time": processingTime.Seconds(),
		"tokens_used":     len(strings.Fields(response)),
		"temperature":     temperature,
		"finish_reason":   "stop",
	})

	return result, nil
}

// ExecuteStep for MockSummaryModel
func (m *MockSummaryModel) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	config := stepConfig.Config

	// Get input text
	input, ok := config["input"].(string)
	if !ok || input == "" {
		return nil, fmt.Errorf("input text is required")
	}

	// Simulate processing time
	processingTime := time.Duration(200+rand.Intn(300)) * time.Millisecond
	time.Sleep(processingTime)

	// Generate mock summary
	summary := m.generateSummary(input)

	result := pipelines.NewPluginContext()
	result.Set(stepConfig.Output, map[string]any{
		"response":        summary,
		"model":           "mock-summary-v1",
		"processing_time": processingTime.Seconds(),
		"tokens_used":     len(strings.Fields(summary)),
		"summary_length":  len(strings.Fields(summary)),
		"finish_reason":   "stop",
	})

	return result, nil
}

// ExecuteStep for MockCreativeModel
func (m *MockCreativeModel) ExecuteStep(ctx context.Context, stepConfig pipelines.StepConfig, globalContext *pipelines.PluginContext) (*pipelines.PluginContext, error) {
	config := stepConfig.Config

	// Get input text
	input, ok := config["input"].(string)
	if !ok || input == "" {
		return nil, fmt.Errorf("input text is required")
	}

	// Get creative parameters
	style := "poetic"
	if s, ok := config["style"].(string); ok {
		style = s
	}

	// Simulate processing time
	processingTime := time.Duration(300+rand.Intn(400)) * time.Millisecond
	time.Sleep(processingTime)

	// Generate creative response
	creativeResponse := m.generateCreativeWriting(input, style)

	result := pipelines.NewPluginContext()
	result.Set(stepConfig.Output, map[string]any{
		"response":        creativeResponse,
		"model":           "mock-creative-v1",
		"processing_time": processingTime.Seconds(),
		"tokens_used":     len(strings.Fields(creativeResponse)),
		"style":           style,
		"finish_reason":   "stop",
	})

	return result, nil
}

// Plugin interface implementations for MockEchoModel
func (m *MockEchoModel) GetPluginType() string { return "AIModels" }
func (m *MockEchoModel) GetPluginName() string { return "mock_echo" }
func (m *MockEchoModel) ValidateConfig(config map[string]any) error {
	if _, ok := config["input"]; !ok {
		return fmt.Errorf("input parameter is required")
	}
	return nil
}

// Plugin interface implementations for MockSummaryModel
func (m *MockSummaryModel) GetPluginType() string { return "AIModels" }
func (m *MockSummaryModel) GetPluginName() string { return "mock_summary" }
func (m *MockSummaryModel) ValidateConfig(config map[string]any) error {
	if _, ok := config["input"]; !ok {
		return fmt.Errorf("input parameter is required")
	}
	return nil
}

// Plugin interface implementations for MockCreativeModel
func (m *MockCreativeModel) GetPluginType() string { return "AIModels" }
func (m *MockCreativeModel) GetPluginName() string { return "mock_creative" }
func (m *MockCreativeModel) ValidateConfig(config map[string]any) error {
	if _, ok := config["input"]; !ok {
		return fmt.Errorf("input parameter is required")
	}
	return nil
}

// Response generation methods

func (m *MockEchoModel) generateConservativeResponse(input string, maxTokens int) string {
	words := strings.Fields(input)
	if len(words) == 0 {
		return "Please provide some input text."
	}

	// Simple word substitution
	substitutions := map[string]string{
		"the":     "the",
		"and":     "plus",
		"or":      "alternatively",
		"but":     "however",
		"is":      "appears to be",
		"are":     "seem to be",
		"was":     "had been",
		"were":    "had been",
		"will":    "shall",
		"would":   "might",
		"can":     "is able to",
		"could":   "might be able to",
		"should":  "ought to",
		"have":    "possess",
		"has":     "possesses",
		"do":      "perform",
		"does":    "performs",
		"did":     "performed",
		"make":    "create",
		"made":    "created",
		"go":      "proceed",
		"went":    "proceeded",
		"come":    "arrive",
		"came":    "arrived",
		"take":    "accept",
		"took":    "accepted",
		"see":     "observe",
		"saw":     "observed",
		"know":    "understand",
		"knew":    "understood",
		"think":   "believe",
		"thought": "believed",
	}

	result := make([]string, 0, len(words))
	for _, word := range words {
		lowerWord := strings.ToLower(word)
		if sub, exists := substitutions[lowerWord]; exists && rand.Float32() < 0.3 {
			if word[0] >= 'A' && word[0] <= 'Z' {
				result = append(result, strings.Title(sub))
			} else {
				result = append(result, sub)
			}
		} else {
			result = append(result, word)
		}
	}

	response := strings.Join(result, " ")

	// Truncate if too long
	if len(strings.Fields(response)) > maxTokens {
		words = strings.Fields(response)[:maxTokens]
		response = strings.Join(words, " ")
	}

	return response
}

func (m *MockEchoModel) generateModerateResponse(input string, maxTokens int) string {
	words := strings.Fields(input)
	if len(words) == 0 {
		return "Please provide some input text."
	}

	// Moderate modifications with some rephrasing
	prefixes := []string{
		"I understand that",
		"It seems that",
		"From what I can tell,",
		"Based on the input,",
		"It appears that",
		"The message indicates that",
	}

	suffixes := []string{
		"This is an interesting point.",
		"That's worth considering.",
		"This provides some insight.",
		"This is quite informative.",
		"This deserves attention.",
	}

	response := input

	// Add prefix
	if rand.Float32() < 0.4 {
		response = prefixes[rand.Intn(len(prefixes))] + " " + response
	}

	// Add suffix
	if rand.Float32() < 0.4 {
		response = response + " " + suffixes[rand.Intn(len(suffixes))]
	}

	// Truncate if too long
	if len(strings.Fields(response)) > maxTokens {
		words = strings.Fields(response)[:maxTokens]
		response = strings.Join(words, " ")
	}

	return response
}

func (m *MockEchoModel) generateCreativeResponse(input string, maxTokens int) string {
	words := strings.Fields(input)
	if len(words) == 0 {
		return "Please provide some input text."
	}

	// Creative modifications with significant changes
	creativePhrases := []string{
		"In the grand tapestry of ideas,",
		"Like a shooting star across the night sky,",
		"Within the boundless realm of thought,",
		"As the universe whispers its secrets,",
		"In the magnificent dance of concepts,",
		"Amidst the symphony of understanding,",
	}

	emphasisWords := []string{
		"absolutely",
		"profoundly",
		"exquisitely",
		"magnificently",
		"wonderfully",
		"brilliantly",
		"extraordinarily",
	}

	response := creativePhrases[rand.Intn(len(creativePhrases))] + " "

	// Add some emphasis words randomly
	for i, word := range words {
		if rand.Float32() < 0.2 {
			emphasis := emphasisWords[rand.Intn(len(emphasisWords))]
			words[i] = emphasis + " " + word
		}
	}

	response += strings.Join(words, " ")

	// Add creative ending
	if rand.Float32() < 0.6 {
		endings := []string{
			", creating ripples of inspiration.",
			", painting vivid strokes of imagination.",
			", composing melodies of insight.",
			", weaving threads of wisdom.",
			", illuminating paths of understanding.",
		}
		response += endings[rand.Intn(len(endings))]
	}

	// Truncate if too long
	if len(strings.Fields(response)) > maxTokens {
		words = strings.Fields(response)[:maxTokens]
		response = strings.Join(words, " ")
	}

	return response
}

func (m *MockSummaryModel) generateSummary(input string) string {
	words := strings.Fields(input)
	if len(words) <= 5 {
		return "The provided text is quite brief and contains the following key points: " + input
	}

	// Count words and estimate reading time
	wordCount := len(words)
	estimatedReadingTime := wordCount / 200 // Assume 200 words per minute

	// Extract key phrases (simple approach)
	keyPhrases := make([]string, 0)
	for i, word := range words {
		// Look for capitalized words or words after punctuation
		if (i == 0 || words[i-1][len(words[i-1])-1] == '.' ||
			words[i-1][len(words[i-1])-1] == '!' ||
			words[i-1][len(words[i-1])-1] == '?') &&
			len(word) > 3 {
			keyPhrases = append(keyPhrases, word)
		}
	}

	summary := fmt.Sprintf("This text contains approximately %d words and discusses ", wordCount)

	if len(keyPhrases) > 0 {
		summary += "key topics including: " + strings.Join(keyPhrases[:min(3, len(keyPhrases))], ", ")
	} else {
		summary += "various concepts and ideas"
	}

	summary += fmt.Sprintf(". Estimated reading time: %d minutes.", max(1, estimatedReadingTime))

	return summary
}

func (m *MockCreativeModel) generateCreativeWriting(input string, style string) string {
	words := strings.Fields(input)
	if len(words) == 0 {
		return "Please provide some input text to inspire creativity."
	}

	var response string

	switch style {
	case "poetic":
		response = m.generatePoeticResponse(input)
	case "story":
		response = m.generateStoryResponse(input)
	case "technical":
		response = m.generateTechnicalResponse(input)
	case "humorous":
		response = m.generateHumorousResponse(input)
	default:
		response = m.generatePoeticResponse(input)
	}

	return response
}

func (m *MockCreativeModel) generatePoeticResponse(input string) string {
	lines := []string{
		"In verses soft and dreams entwined,",
		"Where " + strings.ToLower(input[:min(20, len(input))]) + " begins to shine,",
		"A tapestry of thoughts takes flight,",
		"Through imagination's endless night.",
		"",
		"Each word a star, each phrase a song,",
		"In this creative realm we belong.",
		"Where possibilities dance and play,",
		"In the magic of what we say.",
	}

	return strings.Join(lines, "\n")
}

func (m *MockCreativeModel) generateStoryResponse(input string) string {
	return fmt.Sprintf("Once upon a time, in a world where ideas roam free, there existed a concept known as '%s'. This remarkable notion embarked on a grand adventure, seeking to understand its place in the vast universe of human thought. Along the way, it encountered various challenges and triumphs, each shaping its journey toward enlightenment. In the end, this concept discovered that its true power lay not in its definition, but in the endless possibilities it inspired in those who encountered it.",
		input[:min(30, len(input))])
}

func (m *MockCreativeModel) generateTechnicalResponse(input string) string {
	return fmt.Sprintf("Technical Analysis:\n\nInput Processing: The provided text contains %d characters and %d words.\n\nSemantic Analysis: Primary concepts identified include user intent and contextual meaning.\n\nAlgorithmic Response Generation: Utilizing advanced pattern recognition and natural language processing techniques.\n\nOutput Optimization: Response crafted to maximize clarity and informativeness while maintaining technical precision.\n\nConclusion: The input has been successfully processed and transformed into an optimized technical response.",
		len(input), len(strings.Fields(input)))
}

func (m *MockCreativeModel) generateHumorousResponse(input string) string {
	jokes := []string{
		fmt.Sprintf("Why did the %s go to therapy? It had too many unresolved issues! But seriously, your input about '%s' is quite thought-provoking.",
			input[:min(10, len(input))], input[:min(20, len(input))]),
		fmt.Sprintf("I asked my computer to tell me a joke about '%s', but it just responded with '%s'. I think it's trying to be punny!",
			input[:min(15, len(input))], input[:min(25, len(input))]),
		fmt.Sprintf("If '%s' were a superhero, what would its superpower be? The ability to make people go 'hmmm'? That's pretty powerful if you ask me!",
			input[:min(20, len(input))]),
	}

	return jokes[rand.Intn(len(jokes))]
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
