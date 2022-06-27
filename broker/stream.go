package broker

// stream describes a low-level interface to interact with a queue
type stream interface {
	// direct chan access is left here for easy and native reading via select statements
	//
	// you should never send to ch() directly, use publish
	ch() chan *Message

	publish(m *Message)

	subscribe()
	unsubscribe()
	online() uint32
	clear()
}
