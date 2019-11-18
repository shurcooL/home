package github_test

import (
	"testing"

	"github.com/shurcooL/home/internal/exp/service/activity"
	"github.com/shurcooL/home/internal/exp/service/activity/github"
)

func TestImplementActivityService(t *testing.T) {
	var s interface{} = (*github.Service)(nil)
	if _, ok := s.(activity.Service); !ok {
		t.Error("*github.Service does not implement activity.Service")
	}
}
