package slack

import (
	"context"
	"fmt"
	"strings"

	"github.com/AlexisZankowitch/concept-insight/utils"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
)

func NewGetConceptUserDetails(slack *SlackService) fxctx.Tool {

	return fxctx.NewTool(
		&mcp.Tool{
			Name: "Get user details",
			Description: utils.Ptr("Get the user details of a Concept employee using its slack id"),
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"search": {
						"type": "string",
						"description": "search parameter, could be slack_id, part of the name of the user you are looking for",
					},
				},
				Required: []string{"search"},
			},
		},

		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			search, ok := args["search"].(string)
			if !ok || search == "" {
				return &mcp.CallToolResult{
					IsError: utils.Ptr(true),
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: "Error: 'search' parameter is required and must be a string",
						},
					},
				}

			}
			users, err := slack.ListUsers()
			if err != nil {
				return &mcp.CallToolResult{
					IsError: utils.Ptr(true),
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Error fetching users: %v", err),
						},
					},
				}	
			}

			// lowercase search for case-insensitive match
			searchLower := strings.ToLower(search)
			var matches []ConceptUser
			for _, u := range users {
				if strings.Contains(strings.ToLower(u.Slack_id), searchLower) ||
					strings.Contains(strings.ToLower(u.Slack_Name), searchLower) ||
					strings.Contains(strings.ToLower(u.Real_Name), searchLower) {
					matches = append(matches, u)
				}
			}

			if len(matches) == 0 {
				return &mcp.CallToolResult{
					IsError: utils.Ptr(false),
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("No users found matching '%s'", search),
						},
					},
				}
			}


			// return results
			return &mcp.CallToolResult{
				IsError: utils.Ptr(false),
				Content: []interface{}{
					matches,
				},
			}


		},
	)
}

func NewFindTechnologyPost(slackService *SlackService) fxctx.Tool {
	return fxctx.NewTool(
		// Tool definition for MCP
		&mcp.Tool{
			Name:        "find-technology-posts",
			Description: utils.Ptr("Find posts from a specific technology. Returns an array containing the post, the slack id of the author, the timestamp of the message."),
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

			// // Build response content
			// var contentParts []interface{}
			//
			// // Add summary
			// contentParts = append(contentParts, mcp.TextContent{
			// 	Type: "text",
			// 	Text: fmt.Sprintf("Found %d messages tagged with '%s' across channels 'concept-tech' and 'today-I-learned':", len(allMessages), tech),
			// })
			//
			// // If we have search errors but some results, mention them
			// if len(searchErrors) > 0 {
			// 	contentParts = append(contentParts, mcp.TextContent{
			// 		Type: contentParts"text",
			// 		Text: fmt.Sprintf("Note: Some searches had errors: %s", strings.Join(searchErrors, "; ")),
			// 	})
			// }

			// Add message details for LLM analysis
			// if len(allMessages) > 0 {
			// 	contentParts = append(contentParts, mcp.TextContent{
			// 		Type: "text",
			// 		Text: "Message details for expert analysis:",
			// 	})
			//
			// 	for i, msg := range allMessages {
			// 		contentParts = append(contentParts, mcp.TextContent{
			// 			Type: "text",
			// 			Text: fmt.Sprintf("Message %d:\n- Author: %s (ID: %s)\n- Posted: %s\n- Content: %s\n",
			// 				i+1, msg.Slack_Author_Name, msg.Slack_id, msg.Posted, msg.Message),
			// 		})
			// 	}
			//
			// 	// Add analysis prompt for the LLM
			// 	contentParts = append(contentParts, mcp.TextContent{
			// 		Type: "text",
			// 		Text: fmt.Sprintf("Based on these messages about '%s', please analyze:\n1. Who appears to be the most knowledgeable experts?\n2. Who would you recommend contacting for questions about %s?", tech, tech),
			// 	})
			// } else {
			// 	contentParts = append(contentParts, mcp.TextContent{
			// 		Type: "text",
			// 		Text: fmt.Sprintf("No messages found tagged with '%s' in the searched channels.", tech),
			// 	})
			// }

			return &mcp.CallToolResult{
				Content: []interface{}{
					allMessages,
				}, 
				IsError: utils.Ptr(false),
			}
		},
	)
}
