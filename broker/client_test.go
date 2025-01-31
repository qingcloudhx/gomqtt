package broker

import (
	"testing"
	"time"

	"github.com/qingcloudhx/gomqtt/client"
	"github.com/qingcloudhx/gomqtt/packet"
	"github.com/qingcloudhx/gomqtt/transport"
	"github.com/qingcloudhx/gomqtt/transport/flow"

	"github.com/stretchr/testify/assert"
)

type testMemoryBackend struct {
	MemoryBackend

	packets []packet.Generic
}

func (b *testMemoryBackend) Setup(client *Client, id string, clean bool) (Session, bool, error) {
	client.PacketCallback = func(pkt packet.Generic) error {
		b.packets = append(b.packets, pkt)
		return nil
	}

	return b.MemoryBackend.Setup(client, id, clean)
}

func TestClientMaximumKeepAlive(t *testing.T) {
	backend := &testMemoryBackend{
		MemoryBackend: *NewMemoryBackend(),
	}

	backend.MemoryBackend.ClientMaximumKeepAlive = 10 * time.Millisecond

	port, quit, done := Run(NewEngine(backend), "tcp")

	conn, err := transport.Dial("tcp://localhost:" + port)
	assert.NoError(t, err)

	f := flow.New().
		Send(packet.NewConnect()).
		Receive(packet.NewConnack()).
		End()

	err = f.Test(conn)
	assert.NoError(t, err)

	ret := backend.Close(5 * time.Second)
	assert.True(t, ret)

	close(quit)

	safeReceive(done)
}

func TestClientPacketCallback(t *testing.T) {
	backend := &testMemoryBackend{
		MemoryBackend: *NewMemoryBackend(),
	}

	port, quit, done := Run(NewEngine(backend), "tcp")

	options := client.NewConfig("tcp://localhost:" + port)

	client1 := client.New()

	cf, err := client1.Connect(options)
	assert.NoError(t, err)
	assert.NoError(t, cf.Wait(10*time.Second))

	sf, err := client1.Subscribe("pc", 0)
	assert.NoError(t, err)
	assert.NoError(t, sf.Wait(10*time.Second))

	pf, err := client1.Publish("pc", nil, 1, false)
	assert.NoError(t, err)
	assert.NoError(t, pf.Wait(10*time.Second))

	err = client1.Disconnect()
	assert.NoError(t, err)

	ret := backend.Close(5 * time.Second)
	assert.True(t, ret)

	close(quit)

	safeReceive(done)

	assert.Len(t, backend.packets, 2)
	assert.Equal(t, packet.SUBSCRIBE, backend.packets[0].Type())
	assert.Equal(t, packet.PUBLISH, backend.packets[1].Type())
}

func TestClientTokenTimeoutPublish(t *testing.T) {
	backend := &testMemoryBackend{
		MemoryBackend: *NewMemoryBackend(),
	}

	backend.MemoryBackend.ClientParallelPublishes = 1
	backend.MemoryBackend.ClientTokenTimeout = 10 * time.Millisecond

	port, quit, done := Run(NewEngine(backend), "tcp")

	conn, err := transport.Dial("tcp://localhost:" + port)
	assert.NoError(t, err)

	f := flow.New().
		Send(packet.NewConnect()).
		Receive(packet.NewConnack()).
		Send(&packet.Publish{Message: packet.Message{Topic: "ttp", QOS: 2}, ID: 1}).
		Receive(&packet.Pubrec{ID: 1}).
		Send(&packet.Publish{Message: packet.Message{Topic: "ttp", QOS: 2}, ID: 2}).
		End()

	err = f.Test(conn)
	assert.NoError(t, err)

	ret := backend.Close(5 * time.Second)
	assert.True(t, ret)

	close(quit)

	safeReceive(done)
}

func TestClientTokenTimeoutDequeue(t *testing.T) {
	backend := &testMemoryBackend{
		MemoryBackend: *NewMemoryBackend(),
	}

	backend.MemoryBackend.ClientInflightMessages = 1
	backend.MemoryBackend.ClientTokenTimeout = 10 * time.Millisecond

	port, quit, done := Run(NewEngine(backend), "tcp")

	options := client.NewConfig("tcp://localhost:" + port)

	client1 := client.New()

	cf, err := client1.Connect(options)
	assert.NoError(t, err)
	assert.NoError(t, cf.Wait(10*time.Second))

	conn, err := transport.Dial("tcp://localhost:" + port)
	assert.NoError(t, err)

	f := flow.New().
		Send(packet.NewConnect()).
		Receive(packet.NewConnack()).
		Send(&packet.Subscribe{Subscriptions: []packet.Subscription{{Topic: "ttd", QOS: 2}}, ID: 1}).
		Receive(&packet.Suback{ID: 1, ReturnCodes: []packet.QOS{2}}).
		Run(func() {
			pf, err := client1.Publish("ttd", nil, 2, false)
			assert.NoError(t, err)
			assert.NoError(t, pf.Wait(10*time.Second))

			pf, err = client1.Publish("ttd", nil, 2, false)
			assert.NoError(t, err)
			assert.NoError(t, pf.Wait(10*time.Second))
		}).
		Receive(&packet.Publish{Message: packet.Message{Topic: "ttd", QOS: 2}, ID: 1}).
		End()

	err = f.Test(conn)
	assert.NoError(t, err)

	err = client1.Disconnect()
	assert.NoError(t, err)

	ret := backend.Close(5 * time.Second)
	assert.True(t, ret)

	close(quit)

	safeReceive(done)
}
