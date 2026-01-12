package AI

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type LocalLLMConfig struct {
	ModelPath   string  `json:"model_path"`
	ModelName   string  `json:"model_name"`
	ContextSize int     `json:"context_size"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Threads     int     `json:"threads"`
}

type LocalLLMClient struct {
	config     LocalLLMConfig
	mu         sync.Mutex
	downloaded bool
}

type LocalLLMResponse struct {
	Content      string  `json:"content"`
	FinishReason string  `json:"finish_reason"`
	TokensPerSec float64 `json:"tokens_per_sec"`
}

func NewLocalLLMClient(config LocalLLMConfig) (*LocalLLMClient, error) {
	client := &LocalLLMClient{
		config: config,
	}

	if config.ContextSize == 0 {
		client.config.ContextSize = 2048
	}
	if config.MaxTokens == 0 {
		client.config.MaxTokens = 512
	}
	if config.Temperature == 0 {
		client.config.Temperature = 0.7
	}
	if config.Threads == 0 {
		client.config.Threads = runtime.NumCPU()
	}

	if config.ModelName == "" {
		client.config.ModelName = "tinyllama-1.1b-chat.q4_0.gguf"
	}
	if config.ModelPath == "" {
		client.config.ModelPath = "/app/models"
	}

	return client, nil
}

func (c *LocalLLMClient) EnsureModel(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.downloaded {
		return nil
	}

	modelPath := filepath.Join(c.config.ModelPath, c.config.ModelName)

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		fmt.Printf("Local LLM: Model not found, downloading %s...\n", c.config.ModelName)
		if err := c.downloadModel(ctx, modelPath); err != nil {
			return fmt.Errorf("failed to download model: %w", err)
		}
		fmt.Printf("Local LLM: Model downloaded successfully\n")
	}

	c.downloaded = true
	return nil
}

func (c *LocalLLMClient) downloadModel(ctx context.Context, destPath string) error {
	os.MkdirAll(filepath.Dir(destPath), 0755)

	url := fmt.Sprintf("https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/%s", c.config.ModelName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	progress := &progressWriter{name: c.config.ModelName, total: resp.ContentLength}
	_, err = io.Copy(file, io.TeeReader(resp.Body, progress))
	fmt.Println()
	return err
}

type progressWriter struct {
	name    string
	total   int64
	written int64
}

func (p *progressWriter) Write(buf []byte) (int, error) {
	n := len(buf)
	p.written += int64(n)
	if p.total > 0 {
		percent := float64(p.written) / float64(p.total) * 100
		fmt.Printf("\r  Downloading %s: %.1f%% (%d/%d MB)",
			p.name, percent, p.written/1024/1024, p.total/1024/1024)
	}
	return n, nil
}

func (c *LocalLLMClient) Complete(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.EnsureModel(ctx); err != nil {
		return nil, err
	}

	prompt := c.messagesToPrompt(request.Messages)

	temp := c.config.Temperature
	if request.Temperature > 0 {
		temp = request.Temperature
	}

	maxTokens := c.config.MaxTokens
	if request.MaxTokens > 0 {
		maxTokens = request.MaxTokens
	}

	result, err := c.runInference(ctx, prompt, temp, maxTokens)
	if err != nil {
		return nil, err
	}

	return &LLMResponse{
		Content:      strings.TrimSpace(result.Content),
		FinishReason: result.FinishReason,
		Model:        c.config.ModelName,
		Usage: &LLMUsage{
			PromptTokens:     len(strings.Fields(prompt)),
			CompletionTokens: len(strings.Fields(result.Content)),
			TotalTokens:      len(strings.Fields(prompt)) + len(strings.Fields(result.Content)),
		},
	}, nil
}

func (c *LocalLLMClient) CompleteSimple(ctx context.Context, prompt string) (string, error) {
	resp, err := c.Complete(ctx, LLMRequest{
		Messages: []LLMMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (c *LocalLLMClient) GetProvider() LLMProvider {
	return ProviderLocal
}

func (c *LocalLLMClient) GetDefaultModel() string {
	return c.config.ModelName
}

func (c *LocalLLMClient) ValidateConfig() error {
	if c.config.ModelName == "" {
		return fmt.Errorf("model name is required")
	}
	return nil
}

func (c *LocalLLMClient) runInference(ctx context.Context, prompt string, temperature float64, maxTokens int) (*LocalLLMResponse, error) {
	llamaPath := c.findLlamaBinary()
	if llamaPath == "" {
		return c.runViaOllama(ctx, prompt, temperature, maxTokens)
	}
	return c.runLlamaCPP(ctx, llamaPath, prompt, temperature, maxTokens)
}

func (c *LocalLLMClient) findLlamaBinary() string {
	paths := []string{
		"/app/llama.cpp/main",
		"/app/llama.cpp/llama",
		"/usr/local/bin/llama",
		"/usr/bin/llama",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (c *LocalLLMClient) runLlamaCPP(ctx context.Context, llamaPath, prompt string, temperature float64, maxTokens int) (*LocalLLMResponse, error) {
	modelPath := filepath.Join(c.config.ModelPath, c.config.ModelName)

	args := []string{
		"--model", modelPath,
		"--prompt", prompt,
		"--temp", fmt.Sprintf("%f", temperature),
		"--tokens", fmt.Sprintf("%d", maxTokens),
		"--threads", fmt.Sprintf("%d", c.config.Threads),
		"--ctx-size", fmt.Sprintf("%d", c.config.ContextSize),
		"--no-display-prompt",
	}

	cmd := exec.CommandContext(ctx, llamaPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("llama.cpp execution failed: %w, output: %s", err, string(output))
	}

	content := strings.TrimSpace(string(output))
	if content == "" {
		content = "(no output)"
	}

	return &LocalLLMResponse{Content: content, FinishReason: "stop"}, nil
}

func (c *LocalLLMClient) runViaOllama(ctx context.Context, prompt string, temperature float64, maxTokens int) (*LocalLLMResponse, error) {
	reqBody := map[string]any{
		"model":  "tinyllama",
		"prompt": prompt,
		"stream": false,
		"options": map[string]any{
			"temperature": temperature,
			"num_predict": maxTokens,
		},
	}

	reqJSON, _ := json.Marshal(reqBody)

	cmd := exec.CommandContext(ctx, "curl", "-s", "-X", "POST", "http://localhost:11434/api/generate",
		"-H", "Content-Type: application/json", "-d", string(reqJSON))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Ollama not available (install llama.cpp for offline inference): %w", err)
	}

	var ollamaResp struct{ Response string }
	if err := json.Unmarshal(output, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	return &LocalLLMResponse{Content: strings.TrimSpace(ollamaResp.Response), FinishReason: "stop"}, nil
}

func (c *LocalLLMClient) messagesToPrompt(messages []LLMMessage) string {
	if len(messages) == 0 {
		return ""
	}

	var prompt strings.Builder
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			prompt.WriteString("System: ")
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n")
		case "user":
			prompt.WriteString("User: ")
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n")
		case "assistant":
			prompt.WriteString("Assistant: ")
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n")
		}
	}
	prompt.WriteString("Assistant: ")
	return prompt.String()
}

func ListAvailableLocalModels() []map[string]any {
	return []map[string]any{
		{"name": "TinyLlama-1.1B-Chat", "size": "620 MB", "parameters": "1.1B", "quantization": "Q4_0"},
		{"name": "Phi-2", "size": "320 MB", "parameters": "2.7B", "quantization": "Q4_0"},
		{"name": "Gemma-2B-IT", "size": "1.5 GB", "parameters": "2B", "quantization": "Q4_0"},
	}
}

func (c *LocalLLMClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := c.CompleteSimple(ctx, "Hi")
	return err
}

func (c *LocalLLMClient) GetMemoryEstimate() string {
	estimated := c.config.ContextSize * 2 / (1024 * 1024)
	return fmt.Sprintf("~%d MB for context", estimated)
}

func (c *LocalLLMClient) IsModelLoaded() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.downloaded
}
