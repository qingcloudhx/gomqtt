package spec

import (
	"testing"
	"time"

	"github.com/qingcloudhx/gomqtt/client"
	"github.com/qingcloudhx/gomqtt/packet"
	"github.com/qingcloudhx/gomqtt/transport"
	"github.com/qingcloudhx/gomqtt/transport/flow"

	"github.com/stretchr/testify/assert"
)

// PublishResendQOS1Test tests the broker for properly retrying QOS1 publish
// packets.
func PublishResendQOS1Test(t *testing.T, config *Config, topic string) {
	id := config.clientID()

	assert.NoError(t, client.ClearSession(client.NewConfigWithClientID(config.URL, id), 10*time.Second))

	username, password := config.usernamePassword()

	connect := packet.NewConnect()
	connect.CleanSession = false
	connect.ClientID = id
	connect.Username = username
	connect.Password = password

	subscribe := packet.NewSubscribe()
	subscribe.ID = 1
	subscribe.Subscriptions = []packet.Subscription{
		{Topic: topic, QOS: 1},
	}

	publishOut := packet.NewPublish()
	publishOut.ID = 2
	publishOut.Message.Topic = topic
	publishOut.Message.QOS = 1

	pubackOut := packet.NewPuback()
	pubackOut.ID = 2

	publishIn := packet.NewPublish()
	publishIn.ID = 1
	publishIn.Message.Topic = topic
	publishIn.Message.QOS = 1

	pubackIn := packet.NewPuback()
	pubackIn.ID = 1

	disconnect := packet.NewDisconnect()

	conn1, err := transport.Dial(config.URL)
	assert.NoError(t, err)
	assert.NotNil(t, conn1)

	err = flow.New().
		Send(connect).
		Skip(&packet.Connack{}).
		Send(subscribe).
		Skip(&packet.Suback{}).
		Send(publishOut).
		Receive(pubackOut, publishIn).
		Close().
		Test(conn1)
	assert.NoError(t, err)

	conn2, err := transport.Dial(config.URL)
	assert.NoError(t, err)
	assert.NotNil(t, conn2)

	publishIn.Dup = true

	err = flow.New().
		Send(connect).
		Skip(&packet.Connack{}).
		Receive(publishIn).
		Send(pubackIn).
		Send(disconnect).
		Close().
		Test(conn2)
	assert.NoError(t, err)
}

// PublishResendQOS2Test tests the broker for properly retrying QOS2 Publish
// packets.
func PublishResendQOS2Test(t *testing.T, config *Config, topic string) {
	id := config.clientID()

	assert.NoError(t, client.ClearSession(client.NewConfigWithClientID(config.URL, id), 10*time.Second))

	username, password := config.usernamePassword()

	connect := packet.NewConnect()
	connect.CleanSession = false
	connect.ClientID = id
	connect.Username = username
	connect.Password = password

	subscribe := packet.NewSubscribe()
	subscribe.ID = 1
	subscribe.Subscriptions = []packet.Subscription{
		{Topic: topic, QOS: 2},
	}

	publishOut := packet.NewPublish()
	publishOut.ID = 2
	publishOut.Message.Topic = topic
	publishOut.Message.QOS = 2

	pubrelOut := packet.NewPubrel()
	pubrelOut.ID = 2

	pubcompOut := packet.NewPubcomp()
	pubcompOut.ID = 2

	publishIn := packet.NewPublish()
	publishIn.ID = 1
	publishIn.Message.Topic = topic
	publishIn.Message.QOS = 2

	pubrecIn := packet.NewPubrec()
	pubrecIn.ID = 1

	pubcompIn := packet.NewPubcomp()
	pubcompIn.ID = 1

	disconnect := packet.NewDisconnect()

	conn1, err := transport.Dial(config.URL)
	assert.NoError(t, err)
	assert.NotNil(t, conn1)

	err = flow.New().
		Send(connect).
		Skip(&packet.Connack{}).
		Send(subscribe).
		Skip(&packet.Suback{}).
		Send(publishOut).
		Skip(&packet.Pubrec{}).
		Send(pubrelOut).
		Receive(pubcompOut, publishIn).
		Close().
		Test(conn1)
	assert.NoError(t, err)

	time.Sleep(config.ProcessWait)

	conn2, err := transport.Dial(config.URL)
	assert.NoError(t, err)
	assert.NotNil(t, conn2)

	publishIn.Dup = true

	err = flow.New().
		Send(connect).
		Skip(&packet.Connack{}).
		Receive(publishIn).
		Send(pubrecIn).
		Skip(&packet.Pubrel{}).
		Send(pubcompIn).
		Send(disconnect).
		Close().
		Test(conn2)
	assert.NoError(t, err)
}

// PubrelResendQOS2Test tests the broker for properly retrying QOS2 Pubrel
// packets.
func PubrelResendQOS2Test(t *testing.T, config *Config, topic string) {
	id := config.clientID()

	assert.NoError(t, client.ClearSession(client.NewConfigWithClientID(config.URL, id), 10*time.Second))

	username, password := config.usernamePassword()

	connect := packet.NewConnect()
	connect.CleanSession = false
	connect.ClientID = id
	connect.Username = username
	connect.Password = password

	subscribe := packet.NewSubscribe()
	subscribe.ID = 1
	subscribe.Subscriptions = []packet.Subscription{
		{Topic: topic, QOS: 2},
	}

	publishOut := packet.NewPublish()
	publishOut.ID = 2
	publishOut.Message.Topic = topic
	publishOut.Message.QOS = 2

	pubrelOut := packet.NewPubrel()
	pubrelOut.ID = 2

	pubcompOut := packet.NewPubcomp()
	pubcompOut.ID = 2

	publishIn := packet.NewPublish()
	publishIn.ID = 1
	publishIn.Message.Topic = topic
	publishIn.Message.QOS = 2

	pubrecIn := packet.NewPubrec()
	pubrecIn.ID = 1

	pubrelIn := packet.NewPubrel()
	pubrelIn.ID = 1

	pubcompIn := packet.NewPubcomp()
	pubcompIn.ID = 1

	disconnect := packet.NewDisconnect()

	conn1, err := transport.Dial(config.URL)
	assert.NoError(t, err)
	assert.NotNil(t, conn1)

	err = flow.New().
		Send(connect).
		Skip(&packet.Connack{}).
		Send(subscribe).
		Skip(&packet.Suback{}).
		Send(publishOut).
		Skip(&packet.Pubrec{}).
		Send(pubrelOut).
		Receive(pubcompOut, publishIn).
		Send(pubrecIn).
		Close().
		Test(conn1)
	assert.NoError(t, err)

	time.Sleep(config.ProcessWait)

	conn2, err := transport.Dial(config.URL)
	assert.NoError(t, err)
	assert.NotNil(t, conn2)

	publishIn.Dup = true

	err = flow.New().
		Send(connect).
		Skip(&packet.Connack{}).
		Receive(pubrelIn).
		Send(pubcompIn).
		Send(disconnect).
		Close().
		Test(conn2)
	assert.NoError(t, err)
}

// StoredSubscriptionsTest tests the broker for properly handling stored
// subscriptions.
func StoredSubscriptionsTest(t *testing.T, config *Config, topic string, qos packet.QOS) {
	id := config.clientID()

	options := client.NewConfigWithClientID(config.URL, id)
	options.CleanSession = false

	assert.NoError(t, client.ClearSession(options, 10*time.Second))

	subscriber := client.New()

	cf, err := subscriber.Connect(options)
	assert.NoError(t, err)
	assert.NoError(t, cf.Wait(10*time.Second))
	assert.Equal(t, packet.ConnectionAccepted, cf.ReturnCode())
	assert.False(t, cf.SessionPresent())

	sf, err := subscriber.Subscribe(topic, qos)
	assert.NoError(t, err)
	assert.NoError(t, sf.Wait(10*time.Second))
	assert.Equal(t, []packet.QOS{qos}, sf.ReturnCodes())

	err = subscriber.Disconnect()
	assert.NoError(t, err)

	receiver := client.New()

	wait := make(chan struct{})

	receiver.Callback = func(msg *packet.Message, err error) error {
		assert.NoError(t, err)
		assert.Equal(t, topic, msg.Topic)
		assert.Equal(t, testPayload, msg.Payload)
		assert.Equal(t, packet.QOS(qos), msg.QOS)
		assert.False(t, msg.Retain)

		close(wait)
		return nil
	}

	cf, err = receiver.Connect(options)
	assert.NoError(t, err)
	assert.NoError(t, cf.Wait(10*time.Second))
	assert.Equal(t, packet.ConnectionAccepted, cf.ReturnCode())
	assert.True(t, cf.SessionPresent())

	pf, err := receiver.Publish(topic, testPayload, qos, false)
	assert.NoError(t, err)
	assert.NoError(t, pf.Wait(10*time.Second))

	safeReceive(wait)

	time.Sleep(config.NoMessageWait)

	err = receiver.Disconnect()
	assert.NoError(t, err)
}

// CleanStoredSubscriptionsTest tests the broker for properly clearing stored
// subscriptions.
func CleanStoredSubscriptionsTest(t *testing.T, config *Config, topic string) {
	id := config.clientID()

	options := client.NewConfigWithClientID(config.URL, id)
	options.CleanSession = false

	assert.NoError(t, client.ClearSession(options, 10*time.Second))

	subscriber := client.New()

	cf, err := subscriber.Connect(options)
	assert.NoError(t, err)
	assert.NoError(t, cf.Wait(10*time.Second))
	assert.Equal(t, packet.ConnectionAccepted, cf.ReturnCode())
	assert.False(t, cf.SessionPresent())

	sf, err := subscriber.Subscribe(topic, 0)
	assert.NoError(t, err)
	assert.NoError(t, sf.Wait(10*time.Second))
	assert.Equal(t, []packet.QOS{0}, sf.ReturnCodes())

	err = subscriber.Disconnect()
	assert.NoError(t, err)

	nonReceiver := client.New()
	nonReceiver.Callback = func(msg *packet.Message, err error) error {
		assert.Fail(t, "should not be called")
		return nil
	}

	options.CleanSession = true

	cf, err = nonReceiver.Connect(options)
	assert.NoError(t, err)
	assert.NoError(t, cf.Wait(10*time.Second))
	assert.Equal(t, packet.ConnectionAccepted, cf.ReturnCode())
	assert.False(t, cf.SessionPresent())

	pf, err := nonReceiver.Publish(topic, testPayload, 0, false)
	assert.NoError(t, err)
	assert.NoError(t, pf.Wait(10*time.Second))

	time.Sleep(config.NoMessageWait)

	err = nonReceiver.Disconnect()
	assert.NoError(t, err)
}

// RemoveStoredSubscriptionTest tests the broker for properly removing stored
// subscriptions.
func RemoveStoredSubscriptionTest(t *testing.T, config *Config, topic string) {
	id := config.clientID()

	options := client.NewConfigWithClientID(config.URL, id)
	options.CleanSession = false

	assert.NoError(t, client.ClearSession(options, 10*time.Second))

	subscriberAndUnsubscriber := client.New()

	cf, err := subscriberAndUnsubscriber.Connect(options)
	assert.NoError(t, err)
	assert.NoError(t, cf.Wait(10*time.Second))
	assert.Equal(t, packet.ConnectionAccepted, cf.ReturnCode())
	assert.False(t, cf.SessionPresent())

	sf, err := subscriberAndUnsubscriber.Subscribe(topic, 0)
	assert.NoError(t, err)
	assert.NoError(t, sf.Wait(10*time.Second))
	assert.Equal(t, []packet.QOS{0}, sf.ReturnCodes())

	unsf, err := subscriberAndUnsubscriber.Unsubscribe(topic)
	assert.NoError(t, err)
	assert.NoError(t, unsf.Wait(10*time.Second))

	err = subscriberAndUnsubscriber.Disconnect()
	assert.NoError(t, err)

	nonReceiver := client.New()
	nonReceiver.Callback = func(msg *packet.Message, err error) error {
		assert.Fail(t, "should not be called")
		return nil
	}

	cf, err = nonReceiver.Connect(client.NewConfig(config.URL))
	assert.NoError(t, err)
	assert.NoError(t, cf.Wait(10*time.Second))
	assert.Equal(t, packet.ConnectionAccepted, cf.ReturnCode())
	assert.False(t, cf.SessionPresent())

	pf, err := nonReceiver.Publish(topic, testPayload, 0, false)
	assert.NoError(t, err)
	assert.NoError(t, pf.Wait(10*time.Second))

	time.Sleep(config.NoMessageWait)

	err = nonReceiver.Disconnect()
	assert.NoError(t, err)
}
