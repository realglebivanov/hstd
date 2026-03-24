package store

import "net/url"

type Link struct {
	ID     string `json:"id"`
	Link   string `json:"link"`
	Rotate bool   `json:"rotate,omitempty"`
}

func (l *Link) Summary() string {
	u, err := url.Parse(l.Link)
	if err != nil || u.User == nil {
		return l.Link
	}
	uuid := u.User.Username()
	if len(uuid) > 13 {
		uuid = uuid[:13]
	}
	return uuid + "@" + u.Host
}
