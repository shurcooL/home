package fs

import (
	"context"
	"fmt"

	"github.com/shurcooL/users"
)

func (s *service) user(ctx context.Context, user users.UserSpec) users.User {
	u, err := s.users.Get(ctx, user)
	if err != nil {
		return users.User{
			UserSpec:  user,
			Login:     fmt.Sprintf("%d@%s", user.ID, user.Domain),
			AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96",
			HTMLURL:   "",
		}
	}
	return u
}
