package main

import (
	"limq/channel"
	"sync/atomic"
	"testing"
	"time"
)

func TestInMemory(t *testing.T) {
	var d channel.Descriptor = channel.NewInMemory()

	readersCount := int32(1000)
	readOccurs := int32(0)

	for i := int32(0); i < readersCount; i++ {
		go func(p int) {
			//t.Logf("spawned a waiter #%d...", p+1)

			data, err := d.Read(-1)
			if err != nil {
				t.Error(err)
			}

			_ = data
			//t.Logf("waiter %d received %s", p, string(data))
			atomic.AddInt32(&readOccurs, 1)
		}(int(i))
	}

	time.Sleep(5 * time.Second)
	t.Log("sending the text")

	d.Send([]byte("TEXT"))

	time.Sleep(3 * time.Second)

	if atomic.LoadInt32(&readOccurs) != readersCount {
		t.Error("occurs != 5")
	}
}
