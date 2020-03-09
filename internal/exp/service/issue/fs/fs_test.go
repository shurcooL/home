package fs

import (
	"reflect"
	"testing"

	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

func TestToggleReaction(t *testing.T) {
	c := comment{
		Reactions: []reaction{
			{EmojiID: reactions.EmojiID("bar"), Authors: []userSpec{{ID: 1}, {ID: 2}}},
			{EmojiID: reactions.EmojiID("baz"), Authors: []userSpec{{ID: 3}}},
		},
	}

	toggleReaction(&c, users.UserSpec{ID: 1}, reactions.EmojiID("foo"))
	toggleReaction(&c, users.UserSpec{ID: 1}, reactions.EmojiID("bar"))
	toggleReaction(&c, users.UserSpec{ID: 1}, reactions.EmojiID("baz"))
	toggleReaction(&c, users.UserSpec{ID: 2}, reactions.EmojiID("bar"))

	want := comment{
		Reactions: []reaction{
			{EmojiID: reactions.EmojiID("baz"), Authors: []userSpec{{ID: 3}, {ID: 1}}},
			{EmojiID: reactions.EmojiID("foo"), Authors: []userSpec{{ID: 1}}},
		},
	}

	if got := c; !reflect.DeepEqual(got, want) {
		t.Errorf("\ngot  %+v\nwant %+v", got.Reactions, want.Reactions)
	}
}
