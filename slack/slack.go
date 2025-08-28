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
	Slack_Author_Name string
	Slack_id string
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
		// fmt.Printf("Message Text: %s\n", match.Text)
		// fmt.Printf("Sent by User ID: %s\n", match.User)
		// fmt.Printf("Timestamp: %s\n", match.Timestamp)
		// fmt.Printf("Username: %s\n", match.Username)
		// fmt.Println("--------------------")
		// fmt.Println("")

		results = append(results, MessageInfo{
			Message: match.Text,
			Slack_Author_Name: match.Username,
			Slack_id: match.User,
			Posted: match.Timestamp,
		})
	}

	return results, nil
}

type ConceptUser struct {
	Slack_id string
	Slack_Name string
	Real_Name string
	// Email is not present in profile actually 
	// Email string
}

func (s *SlackService) ListUsers() ([]ConceptUser, error){
	users, err := s.client.GetUsers()
	if err != nil {
		fmt.Printf("error %v \n", err)
		return nil, err
	}

	results := []ConceptUser{}
	for _, user := range users {
		if !user.Deleted {
			results = append(results, ConceptUser{
				Slack_id: user.ID,
				Slack_Name: user.Name,
				Real_Name: user.Profile.RealName,
				// Email: user.Profile.Email,
			})
		} 
	}

	return results, nil
}

func (s *SlackService) GetPostByUser(userId string) ([]MessageInfo, error) {
	results := []MessageInfo{}

	params := slack.SearchParameters{
		Sort:          "timestamp",
		SortDirection: "desc",
		Highlight:     false,
		Count:         100,
		Page:          1,
	}
	search := fmt.Sprintf("from:%s in:#concept-tech in:#today-i-learned", userId)
	searchResults, err := s.client.SearchMessages(search, params)
	if err != nil {
		fmt.Printf("Error %v", err)
		return nil, err
	}

	for _, matche := range searchResults.Matches {
		results = append(results, MessageInfo{
			Message: matche.Text,
			Slack_Author_Name: matche.Username,
			Slack_id: matche.User,
			Posted: matche.Timestamp,
		})
	}

	fmt.Printf("Results get post by user %v", results)
	return results, nil
}
