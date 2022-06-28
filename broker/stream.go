package broker

import "limq/message"

// stream describes a low-level interface to interact with a queue
type stream interface {
	// direct chan access is left here for easy and native reading via select statements
	//
	// you should never send to ch() directly, use publish
	ch() chan *message.Message

	publish(m *message.Message)

	subscribe()
	unsubscribe()
	online() uint32
	clear()
}
