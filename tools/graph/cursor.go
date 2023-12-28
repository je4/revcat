package graph

import (
	emperrors "emperror.dev/errors"
	"encoding/base64"
	"encoding/json"
)

func NewCursor(from, size int) *cursor {
	return &cursor{
		From: from,
		Size: size,
	}
}

func DecodeCursor(s string) (*cursor, error) {
	cCursor, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, emperrors.Wrap(err, "cannot decode cursor")
	}
	c := &cursor{}
	if err := json.Unmarshal(cCursor, c); err != nil {
		return nil, emperrors.Wrap(err, "cannot unmarshal cursor")
	}
	return c, nil
}

type cursor struct {
	From int `json:"from"`
	Size int `json:"size"`
}

func (c *cursor) Encode() (string, error) {
	jCursor, err := json.Marshal(c)
	if err != nil {
		return "", emperrors.Wrap(err, "cannot marshal cursor")
	}
	return base64.StdEncoding.EncodeToString(jCursor), nil
}
