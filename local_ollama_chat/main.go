package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	defaultOllamaURL = "http://localhost:11434"
	defaultMCPURL    = "http://localhost:3000"
	defaultModel     = "llama3.2:latest"
)

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Tools    []Tool    `json:"tools,omitempty"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

type ChatResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`
}

// MCP Types
type MCPRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type MCPResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type MCPTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type MCPToolsListResult struct {
	Tools []MCPTool `json:"tools"`
}

type MCPCallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type MCPCallToolResult struct {
	Content interface{} `json:"content"`
	IsError bool        `json:"isError"`
}

type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type OllamaClient struct {
	baseURL string
	mcpURL  string
	client  *http.Client
	tools   []Tool
	debug   bool
}

func NewOllamaClient(baseURL, mcpURL string, debug bool) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		mcpURL:  mcpURL,
		client:  &http.Client{},
		debug:   debug,
	}
}

func (c *OllamaClient) loadMCPTools() error {
	if c.mcpURL == "" {
		return nil
	}

	req := MCPRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling MCP request: %w", err)
	}

	resp, err := c.client.Post(c.mcpURL+"/mcp", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error connecting to MCP server: %w", err)
	}
	defer resp.Body.Close()

	var mcpResp MCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		return fmt.Errorf("error decoding MCP response: %w", err)
	}

	if mcpResp.Error != nil {
		return fmt.Errorf("MCP error: %s", mcpResp.Error.Message)
	}

	// Convert the result to MCPToolsListResult
	resultBytes, err := json.Marshal(mcpResp.Result)
	if err != nil {
		return fmt.Errorf("error marshaling MCP result: %w", err)
	}

	var toolsResult MCPToolsListResult
	if err := json.Unmarshal(resultBytes, &toolsResult); err != nil {
		return fmt.Errorf("error unmarshaling tools result: %w", err)
	}

	// Convert MCP tools to Ollama tool format
	c.tools = make([]Tool, len(toolsResult.Tools))
	for i, mcpTool := range toolsResult.Tools {
		c.tools[i] = Tool{
			Type: "function",
			Function: Function{
				Name:        mcpTool.Name,
				Description: mcpTool.Description,
				Parameters:  mcpTool.InputSchema,
			},
		}
	}

	return nil
}

func (c *OllamaClient) callMCPTool(name string, arguments map[string]interface{}) (string, error) {
	req := MCPRequest{
		Jsonrpc: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: MCPCallToolParams{
			Name:      name,
			Arguments: arguments,
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("error marshaling MCP tool call: %w", err)
	}

	if c.debug {
		fmt.Printf("🔍 Sending MCP request: %s\n", string(jsonData))
	}

	resp, err := c.client.Post(c.mcpURL+"/mcp", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error calling MCP tool: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading MCP response: %w", err)
	}

	if c.debug {
		fmt.Printf("🔍 MCP response status: %d\n", resp.StatusCode)
		fmt.Printf("🔍 MCP response body: %s\n", string(bodyBytes))
	}

	var mcpResp MCPResponse
	if err := json.Unmarshal(bodyBytes, &mcpResp); err != nil {
		return "", fmt.Errorf("error decoding MCP tool response: %w", err)
	}

	if mcpResp.Error != nil {
		return "", fmt.Errorf("MCP tool error: %s", mcpResp.Error.Message)
	}

	// Convert the result to MCPCallToolResult
	resultBytes, err := json.Marshal(mcpResp.Result)
	if err != nil {
		return "", fmt.Errorf("error marshaling MCP tool result: %w", err)
	}

	var toolResult MCPCallToolResult
	if err := json.Unmarshal(resultBytes, &toolResult); err != nil {
		return "", fmt.Errorf("error unmarshaling tool result: %w", err)
	}

	// Handle different content formats from your MCP server
	var result strings.Builder
	
	// Check if there's an error
	if toolResult.IsError {
		return "", fmt.Errorf("MCP tool returned error")
	}
	
	// Handle the content - it could be various formats
	contentBytes, err := json.Marshal(toolResult.Content)
	if err != nil {
		return "", fmt.Errorf("error marshaling content: %w", err)
	}
	
	if c.debug {
		fmt.Printf("🔍 Tool content: %s\n", string(contentBytes))
	}
	
	// Try to parse as your server's format (array of arrays)
	var arrayOfArrays [][]map[string]interface{}
	if err := json.Unmarshal(contentBytes, &arrayOfArrays); err == nil {
		// Successfully parsed as array of arrays
		for _, array := range arrayOfArrays {
			for i, item := range array {
				if i > 0 {
					result.WriteString("\n")
				}
				// Convert the map to a readable format
				itemBytes, _ := json.MarshalIndent(item, "", "  ")
				result.WriteString(string(itemBytes))
			}
		}
		return result.String(), nil
	}
	
	// Try to parse as standard MCP format (array of content objects)
	var standardContent []MCPContent
	if err := json.Unmarshal(contentBytes, &standardContent); err == nil {
		for i, content := range standardContent {
			if i > 0 {
				result.WriteString("\n")
			}
			result.WriteString(content.Text)
		}
		return result.String(), nil
	}
	
	// If neither format works, return the raw content as string
	return string(contentBytes), nil
}

func (c *OllamaClient) Chat(model string, messages []Message) (*ChatResponse, error) {
	reqBody := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}

	// Add tools if available
	if len(c.tools) > 0 {
		reqBody.Tools = c.tools
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/api/chat", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &chatResp, nil
}

func (c *OllamaClient) processChatWithTools(model string, messages []Message) ([]Message, error) {
	for {
		resp, err := c.Chat(model, messages)
		if err != nil {
			return nil, err
		}

		messages = append(messages, resp.Message)

		// Check if the model wants to use tools
		if len(resp.Message.ToolCalls) == 0 {
			break
		}

		// Process tool calls
		for _, toolCall := range resp.Message.ToolCalls {
			fmt.Printf("🔧 Using tool: %s\n", toolCall.Function.Name)
			
			var args map[string]interface{}
			
			// Handle both string and object formats for arguments
			var argsStr string
			if len(toolCall.Function.Arguments) > 0 {
				// Check if it's already a JSON string or an object
				if toolCall.Function.Arguments[0] == '{' {
					argsStr = string(toolCall.Function.Arguments)
				} else {
					// If it's not JSON, treat it as a plain string
					argsStr = string(toolCall.Function.Arguments)
				}
			}
			
			if c.debug {
				fmt.Printf("🔍 Tool arguments received: %s\n", argsStr)
			}
			
			if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
				if c.debug {
					fmt.Printf("❌ Error parsing tool arguments: %v\n", err)
				}
				return nil, fmt.Errorf("error parsing tool arguments: %w", err)
			}
			
			if c.debug {
				fmt.Printf("🔍 Parsed arguments: %+v\n", args)
			}
			
			result, err := c.callMCPTool(toolCall.Function.Name, args)
			if err != nil {
				if c.debug {
					fmt.Printf("❌ Tool call failed: %v\n", err)
				}
				result = fmt.Sprintf("Error calling tool: %v", err)
			} else if c.debug {
				fmt.Printf("✅ Tool result: %s\n", result)
			}
			
			result, err = c.callMCPTool(toolCall.Function.Name, args)
			if err != nil {
				result = fmt.Sprintf("Error calling tool: %v", err)
			}

			// Add tool result as a message
			toolResultMsg := Message{
				Role:    "tool",
				Content: result,
			}
			messages = append(messages, toolResultMsg)
		}
	}

	return messages, nil
}

func (c *OllamaClient) ListModels() ([]string, error) {
	resp, err := c.client.Get(c.baseURL + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("error fetching models: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding models response: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, model := range result.Models {
		models[i] = model.Name
	}

	return models, nil
}

func main() {
	var (
		model      = flag.String("model", defaultModel, "Ollama model to use")
		ollamaURL  = flag.String("url", defaultOllamaURL, "Ollama server URL")
		mcpURL     = flag.String("mcp", defaultMCPURL, "MCP server URL (empty to disable)")
		listModels = flag.Bool("list", false, "List available models")
		listTools  = flag.Bool("tools", false, "List available MCP tools")
		message    = flag.String("message", "", "Send a single message and exit")
		debug      = flag.Bool("debug", false, "Enable debug output")
	)
	flag.Parse()

	client := NewOllamaClient(*ollamaURL, *mcpURL, *debug)

	// Load MCP tools if MCP URL is provided
	if *mcpURL != "" {
		fmt.Printf("Loading MCP tools from %s...\n", *mcpURL)
		if err := client.loadMCPTools(); err != nil {
			fmt.Printf("Warning: Could not load MCP tools: %v\n", err)
			fmt.Println("Continuing without MCP integration...")
		} else {
			fmt.Printf("Loaded %d MCP tools\n", len(client.tools))
		}
	}

	// Handle list tools command
	if *listTools {
		if len(client.tools) == 0 {
			fmt.Println("No MCP tools available")
		} else {
			fmt.Println("Available MCP tools:")
			for _, tool := range client.tools {
				fmt.Printf("  - %s: %s\n", tool.Function.Name, tool.Function.Description)
			}
		}
		return
	}

	// Handle list models command
	if *listModels {
		models, err := client.ListModels()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing models: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Available models:")
		for _, model := range models {
			fmt.Printf("  - %s\n", model)
		}
		return
	}

	// Handle single message mode
	if *message != "" {
		messages := []Message{
			{Role: "user", Content: *message},
		}

		finalMessages, err := client.processChatWithTools(*model, messages)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Print the final assistant response
		for _, msg := range finalMessages {
			if msg.Role == "assistant" {
				fmt.Println(msg.Content)
			}
		}
		return
	}

	// Interactive chat mode
	fmt.Printf("Ollama CLI Chat with MCP Integration\n")
	fmt.Printf("Connected to Ollama: %s\n", *ollamaURL)
	if *mcpURL != "" {
		fmt.Printf("Connected to MCP: %s (%d tools available)\n", *mcpURL, len(client.tools))
	}
	fmt.Printf("Using model: %s\n", *model)
	fmt.Println("Type 'quit', 'exit', or press Ctrl+C to exit")
	fmt.Println("Type '/clear' to clear conversation history")
	fmt.Println("Type '/models' to list available models")
	fmt.Println("Type '/tools' to list available MCP tools")
	fmt.Println("Type '/model <name>' to switch models")
	fmt.Println("---")

	var conversation []Message
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle commands
		switch {
		case input == "quit" || input == "exit":
			fmt.Println("Goodbye!")
			return
		case input == "/clear":
			conversation = []Message{}
			fmt.Println("Conversation cleared.")
			continue
		case input == "/models":
			models, err := client.ListModels()
			if err != nil {
				fmt.Printf("Error listing models: %v\n", err)
				continue
			}
			fmt.Println("Available models:")
			for _, m := range models {
				if m == *model {
					fmt.Printf("  * %s (current)\n", m)
				} else {
					fmt.Printf("  - %s\n", m)
				}
			}
			continue
		case input == "/tools":
			if len(client.tools) == 0 {
				fmt.Println("No MCP tools available")
			} else {
				fmt.Println("Available MCP tools:")
				for _, tool := range client.tools {
					fmt.Printf("  - %s: %s\n", tool.Function.Name, tool.Function.Description)
				}
			}
			continue
		case strings.HasPrefix(input, "/model "):
			newModel := strings.TrimSpace(strings.TrimPrefix(input, "/model "))
			if newModel != "" {
				*model = newModel
				fmt.Printf("Switched to model: %s\n", *model)
			} else {
				fmt.Println("Usage: /model <model_name>")
			}
			continue
		}

		// Add user message to conversation
		conversation = append(conversation, Message{
			Role:    "user",
			Content: input,
		})

		// Send to Ollama with tool processing
		fmt.Print("Processing...")
		updatedConversation, err := client.processChatWithTools(*model, conversation)
		if err != nil {
			fmt.Printf("\nError: %v\n", err)
			// Remove the last message if there was an error
			conversation = conversation[:len(conversation)-1]
			continue
		}

		// Update conversation with all new messages
		conversation = updatedConversation

		// Clear the "Processing..." line and print the final assistant response
		fmt.Print("\r" + strings.Repeat(" ", 15) + "\r")
		
		// Find and print the last assistant message
		for i := len(conversation) - 1; i >= 0; i-- {
			if conversation[i].Role == "assistant" && conversation[i].Content != "" {
				fmt.Printf("Assistant: %s\n", conversation[i].Content)
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	}
}
