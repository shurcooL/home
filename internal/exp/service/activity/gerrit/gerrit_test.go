package gerrit_test

import (
	"testing"

	"github.com/shurcooL/home/internal/exp/service/activity"
	"github.com/shurcooL/home/internal/exp/service/activity/gerrit"
)

func TestImplementActivityService(t *testing.T) {
	var s interface{} = (*gerrit.Service)(nil)
	if _, ok := s.(activity.Service); !ok {
		t.Error("*gerrit.Service does not implement activity.Service")
	}
}
