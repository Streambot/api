package streambot

import(
	"code.google.com/p/go-uuid/uuid"
)

type Channel struct {
	Id			string
	Name		string
}

func NewChannel(name string) (ch *Channel) { 
	// Create a new runtime Channel object
	ch = &Channel{uuid.New(), name}
	return
}