package entity

type Message struct {
	To   string `json:"to"`
	Type string `json:"type"` // user, group
	Body string `json:"body"`
}
