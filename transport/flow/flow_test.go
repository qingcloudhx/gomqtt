package flow

import (
	"testing"
	"time"

	"github.com/qingcloudhx/gomqtt/packet"

	"github.com/stretchr/testify/assert"
)

func TestFlow(t *testing.T) {
	connect := packet.NewConnect()
	connack := packet.NewConnack()

	subscribe := packet.NewSubscribe()
	subscribe.Subscriptions = []packet.Subscription{
		{Topic: "test"},
	}
	subscribe.ID = 1

	publish1 := packet.NewPublish()
	publish1.ID = 2
	publish1.Message.Topic = "test"
	publish1.Message.QOS = 1

	publish2 := packet.NewPublish()
	publish2.ID = 3
	publish2.Message.Topic = "test"
	publish2.Message.QOS = 1

	wait := make(chan struct{})

	server := New().
		Receive(connect).
		Send(connack).
		Run(func() {
			close(wait)
		}).
		Skip(&packet.Subscribe{}).
		Receive(publish1, publish2).
		Close()

	client := New().
		Send(connect).
		Receive(connack).
		Run(func() {
			<-wait
		}).
		Send(subscribe).
		Send(publish2, publish1).
		End()

	pipe := NewPipe()

	errCh := server.TestAsync(pipe, 100*time.Millisecond)

	err := client.Test(pipe)
	assert.NoError(t, err)

	err = <-errCh
	assert.NoError(t, err)
}

func TestAlreadyClosedError(t *testing.T) {
	pipe := NewPipe()
	pipe.Close()

	err := pipe.Send(nil, false)
	assert.Error(t, err)
}
