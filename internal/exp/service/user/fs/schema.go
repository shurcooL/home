package fs

import "github.com/shurcooL/users"

// Tree layout:
//
// 	root
// 	└── users (newline separated JSON stream of user objects)

// user is an on-disk representation of users.User.
type user struct {
	UserSpec  userSpec
	Elsewhere []userSpec `json:",omitempty"`

	Login     string
	Name      string `json:",omitempty"`
	Email     string `json:",omitempty"`
	AvatarURL string `json:",omitempty"`
	HTMLURL   string `json:",omitempty"`

	SiteAdmin bool `json:",omitempty"`
}

func fromUser(u users.User) user {
	var elsewhere []userSpec
	for _, us := range u.Elsewhere {
		elsewhere = append(elsewhere, fromUserSpec(us))
	}
	return user{
		UserSpec:  fromUserSpec(u.UserSpec),
		Elsewhere: elsewhere,

		Login:     u.Login,
		Name:      u.Name,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		HTMLURL:   u.HTMLURL,

		SiteAdmin: u.SiteAdmin,
	}
}

func (u user) User() users.User {
	var elsewhere []users.UserSpec
	for _, us := range u.Elsewhere {
		elsewhere = append(elsewhere, us.UserSpec())
	}
	return users.User{
		UserSpec:  u.UserSpec.UserSpec(),
		Elsewhere: elsewhere,

		Login:     u.Login,
		Name:      u.Name,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		HTMLURL:   u.HTMLURL,

		SiteAdmin: u.SiteAdmin,
	}
}

// userSpec is an on-disk representation of users.UserSpec.
type userSpec struct {
	ID     uint64
	Domain string `json:",omitempty"`
}

func fromUserSpec(us users.UserSpec) userSpec {
	return userSpec(us)
}

func (us userSpec) UserSpec() users.UserSpec {
	return users.UserSpec(us)
}
