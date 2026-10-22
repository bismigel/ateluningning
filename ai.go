package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// --- DeepSeek API types ---

type DeepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DeepSeekChatRequest struct {
	Model    string            `json:"model"`
	Messages []DeepSeekMessage `json:"messages"`
	Stream   bool              `json:"stream"`
}

type DeepSeekChoice struct {
	Message DeepSeekMessage `json:"message"`
}

type DeepSeekChatResponse struct {
	Choices []DeepSeekChoice `json:"choices"`
}

// HoroscopeData is the structured horoscope returned by the model.
type HoroscopeData struct {
	Status            bool   `json:"status"`
	Prediction        string `json:"prediction"`
	Number            string `json:"number"`
	Color             string `json:"color"`
	Strength          string `json:"strength"`
	Weakness          string `json:"weakness"`
	LoveCompatibility string `json:"love_compatibility"`
	ZodiacSign        string `json:"zodiacSign"`
}

// AIClient talks to the DeepSeek chat-completions API.
type AIClient struct {
	apiKey string
	model  string
	debug  bool
	http   *http.Client
}

// NewAIClient builds a client for the given key/model.
func NewAIClient(apiKey, model string, debug bool) *AIClient {
	return &AIClient{
		apiKey: apiKey,
		model:  model,
		debug:  debug,
		http:   &http.Client{Timeout: RequestTimeout},
	}
}

// Chat sends a single-turn prompt.
func (c *AIClient) Chat(prompt string) (string, error) {
	return c.chat([]DeepSeekMessage{{Role: "user", Content: prompt}})
}

// ChatWithSession sends a prompt with prior conversation history as context.
func (c *AIClient) ChatWithSession(prompt string, history []DeepSeekMessage) (string, error) {
	messages := make([]DeepSeekMessage, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, DeepSeekMessage{Role: "user", Content: prompt})
	return c.chat(messages)
}

// chat performs the request with retry/backoff. This is the single place a
// retry policy lives; both Chat and ChatWithSession flow through it.
func (c *AIClient) chat(messages []DeepSeekMessage) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("DeepSeek API key is not configured")
	}

	fullURL := fmt.Sprintf("https://%s%s", DeepSeekAPIHost, DeepSeekAPIPath)
	requestBody := DeepSeekChatRequest{Model: c.model, Messages: messages, Stream: false}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal DeepSeek request body: %w", err)
	}

	if c.debug {
		log.Printf("DEBUG: Sending DeepSeek API request to: %s", fullURL)
		log.Printf("DEBUG: DeepSeek Request Payload: %s", string(jsonBody))
	}

	const maxRetries = 3
	var resp *http.Response
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Rebuild the request every attempt: http.Client closes the request body,
		// so reusing a single *http.Request would send an empty body on retry.
		req, reqErr := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonBody))
		if reqErr != nil {
			return "", fmt.Errorf("failed to create DeepSeek API request: %w", reqErr)
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

		resp, err = c.http.Do(req)
		if err == nil {
			break
		}

		if isTransientError(err) && attempt < maxRetries-1 {
			wait := time.Duration(attempt+1) * 2 * time.Second
			log.Printf("DeepSeek API timeout (attempt %d/%d), retrying in %v...", attempt+1, maxRetries, wait)
			time.Sleep(wait)
			continue
		}
		break
	}
	if err != nil {
		return "", fmt.Errorf("DeepSeek API request failed after %d attempts: %w", maxRetries, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 50*1024))
	if err != nil {
		return "", fmt.Errorf("error reading DeepSeek API response: %w", err)
	}

	if c.debug {
		log.Printf("DEBUG: DeepSeek Response Status: %d", resp.StatusCode)
		log.Printf("DEBUG: DeepSeek Response Body: %s", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DeepSeek API returned status: %d, body: %s", resp.StatusCode, string(body))
	}

	var parsed DeepSeekChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("failed to parse DeepSeek API response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("DeepSeek API returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

// Horoscope asks the model for a structured daily horoscope.
func (c *AIClient) Horoscope(sign string) (*HoroscopeData, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is not configured for horoscope")
	}

	lowerSign := strings.ToLower(sign)
	if !validSigns[lowerSign] {
		return nil, fmt.Errorf("invalid zodiac sign: %s", sign)
	}

	currentDate := time.Now().Format("January 2, 2006")
	prompt := fmt.Sprintf(`Generate a daily horoscope for %s for today, %s. Provide the response in JSON format only, with the following keys:
- "status": boolean, always true if content is generated
- "zodiacSign": string, the zodiac sign (e.g., "Leo")
- "prediction": string, the main horoscope text
- "love_compatibility": string, love related insights
- "strength": string, strengths for the sign
- "weakness": string, weaknesses for the sign
- "color": string, lucky color(s)
- "number": string, lucky number(s)

Do not include any other text, markdown, or explanation outside the JSON object. Example:
{"status": true, "zodiacSign": "Leo", "prediction": "Your day will be productive...", "love_compatibility": "Compatible with...", "strength": "Courageous", "weakness": "Arrogant", "color": "Gold", "number": "1, 5"}`,
		CapitalizeFirstLetter(lowerSign), currentDate)

	raw, err := c.chat([]DeepSeekMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, fmt.Errorf("error getting raw DeepSeek response for horoscope: %w", err)
	}

	var data HoroscopeData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		if c.debug {
			log.Printf("DEBUG: Failed to unmarshal DeepSeek horoscope response. Raw response: %s", raw)
		}
		return nil, fmt.Errorf("DeepSeek did not return horoscope in expected JSON format. Please try again. Raw output: ```json\n%s\n``` (check bot logs for full details)", raw)
	}

	data.Status = true
	if data.ZodiacSign == "" {
		data.ZodiacSign = CapitalizeFirstLetter(lowerSign)
	}
	return &data, nil
}

// isTransientError reports whether err looks like a timeout/cancellation worth
// retrying.
func isTransientError(err error) bool {
	e := strings.ToLower(err.Error())
	return strings.Contains(e, "timeout") ||
		strings.Contains(e, "deadline") ||
		strings.Contains(e, "canceled")
}

// aiErrorText builds the persona-flavoured error shown to users, with a
// timeout-specific hint and a caller-provided tail for other failures.
func aiErrorText(lead string, err error, tail string) string {
	msg := AteLuningningErrorPrefix + lead
	if isTransientError(err) {
		return msg + "The DeepSeek servers might be busy. Please try again in a moment!"
	}
	return msg + fmt.Sprintf("Reason: %v%s", err, tail)
}
