package main

import "time"

// ErrorMsg is an error type that was recorded as an event.
type ErrorMsg struct {
	ID          string                 `json:"id"`
	App         string                 `json:"app"`
	CreatedAt   time.Time              `json:"created_at"`
	Payload     map[string]interface{} `json:"payload"`
	PayloadJSON string                 `json:"-"`
}

// ByCreatedAt allows errors to be sorted in the inverse order that they were
// recieved.
type ByCreatedAt []ErrorMsg

func (e ByCreatedAt) Len() int {
	return len(e)
}
func (e ByCreatedAt) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
func (e ByCreatedAt) Less(i, j int) bool {
	return e[i].CreatedAt.After(e[j].CreatedAt)
}
