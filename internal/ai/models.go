package ai

import "time"

const (
	defaultGroqModel   = "openai/gpt-oss-20b"
	defaultGroqTimeout = 60 * time.Second
	defaultGroqBaseURL = "https://api.groq.com/openai/v1"
)

// GroqConfig holds construction-time configuration for the Groq client.
type GroqConfig struct {
	APIKey  string
	Model   string
	Timeout time.Duration
	BaseURL string
}

// groqMessage is a single chat message in the OpenAI-compatible wire format.
type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// groqRequest is the JSON body sent to the Groq chat completions endpoint.
type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
}

// groqChoice is one completion choice returned by the API.
type groqChoice struct {
	Message groqMessage `json:"message"`
}

// groqResponse is the successful JSON response from the Groq API.
type groqResponse struct {
	Choices []groqChoice `json:"choices"`
	Usage   struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// groqErrorBody is the JSON response body when the API returns a non-200 status.
type groqErrorBody struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}
