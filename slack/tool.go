package slack

import (
	"context"
	"fmt"
	"strings"

	"github.com/AlexisZankowitch/concept-insight/utils"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
)

func NewFindTechExpertTool(slackService *SlackService) fxctx.Tool {
	return fxctx.NewTool(
		// Tool definition for MCP
		&mcp.Tool{
			Name:        "find-tech-expert",
			Description: utils.Ptr("Find experts for a specific technology by analyzing tagged messages from Slack channels"),
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"technology": {
						"type":        "string",
						"description": "The technology to search for (e.g., python, react, golang)",
					},
				},
				Required: []string{"technology"},
			},
		},

		// Tool execution callback
		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			// Extract technology from arguments
			tech, ok := args["technology"].(string)
			if !ok || tech == "" {
				return &mcp.CallToolResult{
					IsError: utils.Ptr(true),
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: "Error: 'technology' parameter is required and must be a string",
						},
					},
				}
			}

			// Search in both channels
			channels := []string{"concept-tech", "today-I-learned"}
			var allMessages []MessageInfo
			var searchErrors []string

			for _, channel := range channels {
				messages, err := slackService.GetTechonologyPost(tech, channel)
				if err != nil {
					searchErrors = append(searchErrors, fmt.Sprintf("Error searching in %s: %v", channel, err))
					continue
				}
				allMessages = append(allMessages, messages...)
			}

			// If we have errors but no messages, return error
			if len(allMessages) == 0 && len(searchErrors) > 0 {
				return &mcp.CallToolResult{
					IsError: utils.Ptr(true),
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Failed to retrieve messages: %s", strings.Join(searchErrors, "; ")),
						},
					},
				}
			}

			// Build response content
			var contentParts []interface{}

			// Add summary
			contentParts = append(contentParts, mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Found %d messages tagged with '%s' across channels 'concept-tech' and 'today-I-learned':", len(allMessages), tech),
			})

			// If we have search errors but some results, mention them
			if len(searchErrors) > 0 {
				contentParts = append(contentParts, mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Note: Some searches had errors: %s", strings.Join(searchErrors, "; ")),
				})
			}

			// Add message details for LLM analysis
			if len(allMessages) > 0 {
				contentParts = append(contentParts, mcp.TextContent{
					Type: "text",
					Text: "Message details for expert analysis:",
				})

				for i, msg := range allMessages {
					contentParts = append(contentParts, mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Message %d:\n- Author: %s (ID: %s)\n- Posted: %s\n- Content: %s\n",
							i+1, msg.Author, msg.Author_slack_id, msg.Posted, msg.Message),
					})
				}

				// Add analysis prompt for the LLM
				contentParts = append(contentParts, mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Based on these messages about '%s', please analyze:\n1. Who appears to be the most knowledgeable experts?\n2. Who would you recommend contacting for questions about %s?", tech, tech),
				})
			} else {
				contentParts = append(contentParts, mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("No messages found tagged with '%s' in the searched channels.", tech),
				})
			}

			return &mcp.CallToolResult{
				Content: contentParts,
				IsError: utils.Ptr(false),
			}
		},
	)
}
