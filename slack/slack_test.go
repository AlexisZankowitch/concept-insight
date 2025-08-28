package slack

import (
	"fmt"
	"testing"
)

func Test_Slack(t *testing.T) {
	s := NewSlackService()
	r, err :=	s.GetTechonologyPost("golang", "concept-tech")
	if err != nil {
		fmt.Printf("Err %v", err)
		return
	}
	fmt.Printf("Results: %d", len(r))
}

func Test_SlackUser(t *testing.T) {
	s := NewSlackService()

	r, err := s.ListUsers()
	if err != nil {
		fmt.Printf("err %v", err)
		return
	}

	fmt.Printf("Results size %v \n", len(r))
	fmt.Printf("Results %v", r)
}
