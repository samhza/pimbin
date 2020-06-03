package pimbin

import (
	"crypto/rand"
	"encoding/base64"
)

type user struct {
	User
	srv *Server
}

func (u *user) refreshToken(user *User) error {
	b := make([]byte, 24)
	rand.Read(b)
	u.Token = base64.URLEncoding.EncodeToString(b)
	_, err := u.srv.db.RefreshToken(&u.User)
	return err
}

func (u *user) updatePassword(hash string) error {
	u.User.Password = hash
	return u.srv.db.UpdatePassword(&u.User)
}
