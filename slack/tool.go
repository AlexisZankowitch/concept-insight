package slack

import (
	"context"
	"fmt"
	"strings"

	"github.com/AlexisZankowitch/concept-insight/utils"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
)


func NewGetLastestPostsByUserId(slack *SlackService) fxctx.Tool {
	return fxctx.NewTool(
		&mcp.Tool{
			Name: "Get the latest 200 posts by slack user id",
			Description: utils.Ptr("Retrieve the lastest 200 posts of a user ifentified by its slack user id"),
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"slack_user_id": {
						"type": "string",
						"description": "slack user id of the user we want to list the post from",
					},
				},
				Required: []string{"slack_user_id"},
			},
		},
		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			slackUserId, ok := args["slack_user_id"].(string)
			if !ok || slackUserId == "" {
				return &mcp.CallToolResult{
					IsError: utils.Ptr(true),
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: "Error: slack user id is required and must be a string",
						},
					},
				}
			}

			posts, err := slack.GetPostByUser(slackUserId)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: utils.Ptr(true),
					Content: []interface{}{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Error fetching user's post: %v", err),
						},
					},
				}	
			}

			return &mcp.CallToolResult{
				IsError: utils.Ptr(false),
				Content: []interface{}{
					posts,
				},
			}
		},
	)
}			
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


			return &mcp.CallToolResult{
				Content: []interface{}{
					allMessages,
				}, 
				IsError: utils.Ptr(false),
			}
		},
	)
}
