package resolver

import "time"

func NewGuestUser() *User {
	return &User{
		Id:        "0",
		Groups:    []string{"global/guest"},
		Email:     "",
		FirstName: "",
		LastName:  "Guest",
		HomeOrg:   "",
		Exp:       time.Now().Add(time.Hour * 24),
		LoggedIn:  false,
		LoggedOut: false,
	}
}

type User struct {
	Id        string    `json:"Id"`
	Groups    []string  `json:"Groups"`
	Email     string    `json:"email"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	HomeOrg   string    `json:"homeOrg"`
	Exp       time.Time `json:"exp"`
	LoggedIn  bool      `json:"loggedIn"`
	LoggedOut bool      `json:"loggedOut"`
}

func (u User) inGroup(grp string) bool {
	for _, g := range u.Groups {
		if g == grp {
			return true
		}
	}
	return false
}
