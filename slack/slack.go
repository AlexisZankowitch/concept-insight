package slack

import (
	"fmt"

	"github.com/AlexisZankowitch/concept-insight/config"
	"github.com/slack-go/slack"
)


type SlackService struct {
	client *slack.Client
}

type MessageInfo struct {
	Message string
	Author string
	Author_slack_id string
	Posted string
}

// NewSlackService creates a new Slack service
func NewSlackService() *SlackService {
	api := slack.New(config.AppConfig.SlackToken)
	return &SlackService{
		client: api,
	}
}

func (s *SlackService) GetTechonologyPost(tech string, channel string) ([]MessageInfo, error) {
	params := slack.SearchParameters{
		Sort:          "score",
		SortDirection: "desc",
		Highlight:     false,
		Count:         20,
		Page:          1,
	}
	searchTechno := fmt.Sprintf("has::%s: in:%s", tech, channel)
	fmt.Printf("Search: %s", searchTechno)
	searchResult, err := s.client.SearchMessages(searchTechno, params)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return []MessageInfo{}, err
	}

	fmt.Printf("Found %d\n", len(searchResult.Matches))
	results := []MessageInfo{}
	// Loop through each message in the Matches slice.
	for _, match := range searchResult.Matches {
		// Access the individual message fields.
		// For example, the message's text, the user who sent it, and the timestamp.
		fmt.Printf("Message Text: %s\n", match.Text)
		fmt.Printf("Sent by User ID: %s\n", match.User)
		fmt.Printf("Timestamp: %s\n", match.Timestamp)
		fmt.Printf("Username: %s\n", match.Username)
		fmt.Println("--------------------")
		fmt.Println("")

		results = append(results, MessageInfo{
			Message: match.Text,
			Author: match.Username,
			Author_slack_id: match.User,
			Posted: match.Timestamp,
		})
	}

	return results, nil
}
