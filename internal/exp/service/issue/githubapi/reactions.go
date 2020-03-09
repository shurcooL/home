package githubapi

import (
	"fmt"

	"github.com/shurcooL/githubv4"
	"github.com/shurcooL/reactions"
	"github.com/shurcooL/users"
)

type reactionGroups []struct {
	Content githubv4.ReactionContent
	Users   struct {
		Nodes      []*githubV4User
		TotalCount int
	} `graphql:"users(first:10)"`
	ViewerHasReacted bool
}

// ghReactions converts []githubv4.ReactionGroup to []reactions.Reaction.
func ghReactions(rgs reactionGroups, viewer users.User) []reactions.Reaction {
	var rs []reactions.Reaction
	for _, rg := range rgs {
		if rg.Users.TotalCount == 0 {
			continue
		}

		// Only return the details of first few users and viewer.
		var us []users.User
		addedViewer := false
		for i := 0; i < rg.Users.TotalCount; i++ {
			if i < len(rg.Users.Nodes) {
				user := ghUser(rg.Users.Nodes[i])
				us = append(us, user)
				if user.UserSpec == viewer.UserSpec {
					addedViewer = true
				}
			} else if i == len(rg.Users.Nodes) {
				// Add viewer last if they've reacted, but haven't been added already.
				if rg.ViewerHasReacted && !addedViewer {
					us = append(us, viewer)
				} else {
					us = append(us, users.User{})
				}
			} else {
				us = append(us, users.User{})
			}
		}

		rs = append(rs, reactions.Reaction{
			Reaction: internalizeReaction(rg.Content),
			Users:    us,
		})
	}
	return rs
}

// internalizeReaction converts githubv4.ReactionContent to reactions.EmojiID.
func internalizeReaction(reaction githubv4.ReactionContent) reactions.EmojiID {
	switch reaction {
	case githubv4.ReactionContentThumbsUp:
		return "+1"
	case githubv4.ReactionContentThumbsDown:
		return "-1"
	case githubv4.ReactionContentLaugh:
		return "smile"
	case githubv4.ReactionContentHooray:
		return "tada"
	case githubv4.ReactionContentConfused:
		return "confused"
	case githubv4.ReactionContentHeart:
		return "heart"
	case githubv4.ReactionContentRocket:
		return "rocket"
	case githubv4.ReactionContentEyes:
		return "eyes"
	default:
		panic("unreachable")
	}
}

// externalizeReaction converts reactions.EmojiID to githubv4.ReactionContent.
func externalizeReaction(reaction reactions.EmojiID) (githubv4.ReactionContent, error) {
	switch reaction {
	case "+1":
		return githubv4.ReactionContentThumbsUp, nil
	case "-1":
		return githubv4.ReactionContentThumbsDown, nil
	case "smile":
		return githubv4.ReactionContentLaugh, nil
	case "tada":
		return githubv4.ReactionContentHooray, nil
	case "confused":
		return githubv4.ReactionContentConfused, nil
	case "heart":
		return githubv4.ReactionContentHeart, nil
	case "rocket":
		return githubv4.ReactionContentRocket, nil
	case "eyes":
		return githubv4.ReactionContentEyes, nil
	default:
		return "", fmt.Errorf("%q is an unsupported reaction", reaction)
	}
}
