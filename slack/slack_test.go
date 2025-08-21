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
