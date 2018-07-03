package broker

import (
	"errors"
	"sync"
	"time"

	"github.com/256dpi/gomqtt/packet"
	"github.com/256dpi/gomqtt/session"
	"github.com/256dpi/gomqtt/topic"
)

// A Session is used to persist incoming/outgoing packets, subscriptions and the
// will.
type Session interface {
	// NextID should return the next id for outgoing packets.
	NextID() packet.ID

	// SavePacket should store a packet in the session. An eventual existing
	// packet with the same id should be quietly overwritten.
	SavePacket(session.Direction, packet.GenericPacket) error

	// LookupPacket should retrieve a packet from the session using the packet id.
	LookupPacket(session.Direction, packet.ID) (packet.GenericPacket, error)

	// DeletePacket should remove a packet from the session. The method should
	// not return an error if no packet with the specified id does exists.
	DeletePacket(session.Direction, packet.ID) error

	// AllPackets should return all packets currently saved in the session. This
	// method is used to resend stored packets when the session is resumed.
	AllPackets(session.Direction) ([]packet.GenericPacket, error)

	// SaveSubscription should store the subscription in the session. An eventual
	// subscription with the same topic should be quietly overwritten.
	SaveSubscription(*packet.Subscription) error

	// LookupSubscription should match a topic against the stored subscriptions
	// and eventually return the first found subscription.
	LookupSubscription(topic string) (*packet.Subscription, error)

	// DeleteSubscription should remove the subscription from the session. The
	// method should not return an error if no subscription with the specified
	// topic does exist.
	DeleteSubscription(topic string) error

	// AllSubscriptions should return all subscriptions currently saved in the
	// session. This method is used to restore a clients subscriptions when the
	// session is resumed.
	AllSubscriptions() ([]*packet.Subscription, error)

	// SaveWill should store the will message.
	SaveWill(*packet.Message) error

	// LookupWill should retrieve the will message.
	LookupWill() (*packet.Message, error)

	// ClearWill should remove the will message from the store.
	ClearWill() error

	// Reset should completely reset the session.
	Reset() error
}

// Ack is executed by the Backend or Client to signal that a message will be
// delivered under the selected qos level and is therefore safe to be deleted
// from either queue.
type Ack func(message *packet.Message)

// A Backend provides the effective brokering functionality to its clients.
type Backend interface {
	// Authenticate should authenticate the client using the user and password
	// values and return true if the client is eligible to continue or false
	// when the broker should terminate the connection.
	Authenticate(client *Client, user, password string) (bool, error)

	// Setup is called when a new client comes online and is successfully
	// authenticated. Setup should return the already stored session for the
	// supplied id or create and return a new one. If the supplied id has a zero
	// length, a new temporary session should returned that is not stored
	// further. The backend may also close any existing clients that use the
	// same client id.
	//
	// Note: In this call the Backend may also allocate other resources and
	// setup the client for further usage as the broker will acknowledge the
	// connection when the call returns.
	Setup(client *Client, id string) (Session, bool, error)

	// QueueOffline is called after the clients stored subscriptions have been
	// resubscribed. It should be used to trigger a background process that
	// forwards all missed messages.
	QueueOffline(*Client) error

	// TODO: Rename to restored?

	// Subscribe should subscribe the passed client to the specified topic and
	// call Publish with any incoming messages. The subscription will also be
	// added to the session if the call returns without an error.
	Subscribe(*Client, *packet.Subscription) error

	// Unsubscribe should unsubscribe the passed client from the specified topic.
	// The subscription will also be removed from the session if the call returns
	// without an error.
	Unsubscribe(client *Client, topic string) error

	// Dequeue is called by the Client repeatedly to obtain the next message.
	// The backend must return no message and no error if the supplied channel
	// is closed. The returned Ack is executed by the Backend to signal that the
	// message is being delivered under the selected qos level and is therefore
	// safe to be deleted from the queue.
	Dequeue(*Client, <-chan struct{}) (*packet.Message, Ack, error)

	// StoreRetained should store the specified message.
	StoreRetained(*Client, *packet.Message) error

	// ClearRetained should remove the stored messages for the given topic.
	ClearRetained(client *Client, topic string) error

	// QueueRetained is called after acknowledging a subscription and should be
	// used to trigger a background process that forwards all retained messages.
	QueueRetained(client *Client, topic string) error

	// Publish should forward the passed message to all other clients that hold
	// a subscription that matches the messages topic. It should also add the
	// message to all sessions that have a matching offline subscription.
	//
	// Note: If the backend does not return an error the message will be
	// immediately acknowledged by the client and removed from the session.
	Publish(*Client, *packet.Message) error

	// TODO: Publish with ack.

	// Terminate is called when the client goes offline. Terminate should
	// unsubscribe the passed client from all previously subscribed topics. The
	// backend may also convert a clients subscriptions to offline subscriptions.
	//
	// Note: The Backend may also cleanup previously allocated resources for
	// that client as the broker will close the connection when the call
	// returns.
	Terminate(*Client) error
}

type memorySession struct {
	*session.MemorySession

	queue chan *packet.Message

	owner *Client
	kill  chan struct{}
	done  chan struct{}
}

func newMemorySession(backlog int) *memorySession {
	return &memorySession{
		MemorySession: session.NewMemorySession(),
		queue:         make(chan *packet.Message, backlog),
		kill:          make(chan struct{}, 1),
		done:          make(chan struct{}, 1),
	}
}

func (s *memorySession) reuse() {
	s.kill = make(chan struct{}, 1)
	s.done = make(chan struct{}, 1)
}

// ErrQueueFull is returned to a client that attempts two write to its own full
// queue, which would result in a deadlock.
var ErrQueueFull = errors.New("queue full")

// ErrKilled is returned to a client that is killed by the broker.
var ErrKilled = errors.New("killed")

// ErrClosing is returned to a client if the backend is closing.
var ErrClosing = errors.New("closing")

// ErrKillTimeout is returned to a client if the killed existing client does not
// close in time.
var ErrKillTimeout = errors.New("kill timeout")

// A MemoryBackend stores everything in memory.
type MemoryBackend struct {
	// The maximal size of the session queue.
	SessionQueueSize int

	// The time after an error is returned while waiting on an killed existing
	// client to exit.
	KillTimeout time.Duration

	// A map of username and passwords that grant read and write access.
	Credentials map[string]string

	activeClients     map[string]*Client
	storedSessions    map[string]*memorySession
	temporarySessions map[*Client]*memorySession
	retainedMessages  *topic.Tree

	globalMutex sync.Mutex
	setupMutex  sync.Mutex
	closing     bool
}

// NewMemoryBackend returns a new MemoryBackend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		SessionQueueSize:  100,
		KillTimeout:       5 * time.Second,
		activeClients:     make(map[string]*Client),
		storedSessions:    make(map[string]*memorySession),
		temporarySessions: make(map[*Client]*memorySession),
		retainedMessages:  topic.NewTree(),
	}
}

// Authenticate authenticates a clients credentials by matching them to the
// saved Credentials map.
func (m *MemoryBackend) Authenticate(client *Client, user, password string) (bool, error) {
	// acquire global mutex
	m.globalMutex.Lock()
	defer m.globalMutex.Unlock()

	// return error if closing
	if m.closing {
		return false, ErrClosing
	}

	// allow all if there are no credentials
	if m.Credentials == nil {
		return true, nil
	}

	// check login
	if pw, ok := m.Credentials[user]; ok && pw == password {
		return true, nil
	}

	return false, nil
}

// Setup returns the already stored session for the supplied id or creates and
// returns a new one. If the supplied id has a zero length, a new session is
// returned that is not stored further. Furthermore, it will disconnect any client
// connected with the same client id.
func (m *MemoryBackend) Setup(client *Client, id string) (Session, bool, error) {
	// acquire global mutex
	m.globalMutex.Lock()
	defer m.globalMutex.Unlock()

	// acquire setup mutex
	m.setupMutex.Lock()
	defer m.setupMutex.Unlock()

	// return error if closing
	if m.closing {
		return nil, false, ErrClosing
	}

	// return a new temporary session if id is zero
	if len(id) == 0 {
		// create session
		sess := newMemorySession(m.SessionQueueSize)
		sess.owner = client

		// save session
		m.temporarySessions[client] = sess

		return sess, false, nil
	}

	// client id is available

	// retrieve existing client
	existingSession, ok := m.storedSessions[id]
	if !ok {
		if existingClient, ok2 := m.activeClients[id]; ok2 {
			existingSession, ok = m.temporarySessions[existingClient]
		}
	}

	// kill existing client if session is taken
	if ok && existingSession.owner != nil {
		// send signal
		close(existingSession.kill)

		// release global mutex to allow publish and termination, but leave the
		// setup mutex to prevent setups
		m.globalMutex.Unlock()

		// wait for client to close
		select {
		case <-existingSession.done:
			// continue
		case <-time.After(m.KillTimeout):
			return nil, false, ErrKillTimeout
		}

		// acquire mutex again
		m.globalMutex.Lock()
	}

	// delete any stored session and return temporary if requested
	if client.CleanSession() {
		// delete any stored session
		delete(m.storedSessions, id)

		// create new session
		sess := newMemorySession(m.SessionQueueSize)
		sess.owner = client

		// save session
		m.temporarySessions[client] = sess

		// save client
		m.activeClients[id] = client

		return sess, false, nil
	}

	// attempt to reuse a stored session
	storedSession, ok := m.storedSessions[id]
	if ok {
		// reuse session
		storedSession.reuse()
		storedSession.owner = client

		// save client
		m.activeClients[id] = client

		return storedSession, true, nil
	}

	// otherwise create fresh session
	storedSession = newMemorySession(m.SessionQueueSize)
	storedSession.owner = client

	// save session
	m.storedSessions[id] = storedSession

	// save client
	m.activeClients[id] = client

	return storedSession, false, nil
}

// QueueOffline will begin with forwarding all missed messages in a separate
// goroutine.
func (m *MemoryBackend) QueueOffline(client *Client) error {
	// not needed as missed messages are already added to the session queue

	return nil
}

// Subscribe will subscribe the passed client to the specified topic.
func (m *MemoryBackend) Subscribe(client *Client, sub *packet.Subscription) error {
	// the subscription will be added to the session by the broker

	return nil
}

// Unsubscribe will unsubscribe the passed client from the specified topic.
func (m *MemoryBackend) Unsubscribe(client *Client, topic string) error {
	// the subscription will be removed from the session by the broker

	return nil
}

// Dequeue will get the next message from the queue.
func (m *MemoryBackend) Dequeue(client *Client, close <-chan struct{}) (*packet.Message, Ack, error) {
	// mutex locking not needed

	// get session
	sess := client.Session().(*memorySession)

	// TODO: Add ack support.

	// get next message from queue
	select {
	case msg := <-sess.queue:
		return msg, nil, nil
	case <-close:
		return nil, nil, nil
	case <-sess.kill:
		return nil, nil, ErrKilled
	}
}

// StoreRetained will store the specified message.
func (m *MemoryBackend) StoreRetained(client *Client, msg *packet.Message) error {
	// mutex locking not needed

	// set retained message
	m.retainedMessages.Set(msg.Topic, msg.Copy())

	return nil
}

// ClearRetained will remove the stored messages for the given topic.
func (m *MemoryBackend) ClearRetained(client *Client, topic string) error {
	// mutex locking not needed

	// clear retained message
	m.retainedMessages.Empty(topic)

	return nil
}

// QueueRetained will queue all retained messages matching the given topic.
func (m *MemoryBackend) QueueRetained(client *Client, topic string) error {
	// get retained messages
	values := m.retainedMessages.Search(topic)

	// publish messages
	for _, value := range values {
		select {
		case client.Session().(*memorySession).queue <- value.(*packet.Message):
		default:
			return ErrQueueFull
		}
	}

	return nil
}

// Publish will forward the passed message to all other subscribed clients. It
// will also add the message to all sessions that have a matching offline
// subscription.
func (m *MemoryBackend) Publish(client *Client, msg *packet.Message) error {
	// acquire global mutex
	m.globalMutex.Lock()
	defer m.globalMutex.Unlock()

	// add message to temporary sessions
	for _, sess := range m.temporarySessions {
		if sub, _ := sess.LookupSubscription(msg.Topic); sub != nil {
			// detect deadlock when adding to own queue
			if sess.owner == client {
				select {
				case sess.queue <- msg:
				default:
					return ErrQueueFull
				}
			} else {
				sess.queue <- msg
			}
		}
	}

	// add message to stored sessions
	for _, sess := range m.storedSessions {
		if sub, _ := sess.LookupSubscription(msg.Topic); sub != nil {
			// detect deadlock when adding to own queue
			if sess.owner == client {
				select {
				case sess.queue <- msg:
				default:
					return ErrQueueFull
				}
			} else {
				sess.queue <- msg
			}
		}
	}

	return nil
}

// Terminate will unsubscribe the passed client from all previously subscribed
// topics. If the client connect with clean=true it will also clean the session.
// Otherwise it will create offline subscriptions for all QOS 1 and QOS 2
// subscriptions.
func (m *MemoryBackend) Terminate(client *Client) error {
	// acquire global mutex
	m.globalMutex.Lock()
	defer m.globalMutex.Unlock()

	// get session
	sess := client.Session().(*memorySession)

	// release session
	sess.owner = nil

	// remove any temporary session
	delete(m.temporarySessions, client)

	// remove any saved client
	delete(m.activeClients, client.ClientID())

	// signal exit
	close(sess.done)

	return nil
}

// Close will close all active clients and close the backend. The return value
// denotes if the timeout has been reached.
func (m *MemoryBackend) Close(timeout time.Duration) bool {
	// acquire global mutex
	m.globalMutex.Lock()

	// set closing
	m.closing = true

	// prepare channel list
	var list []chan struct{}

	// close temporary sessions
	for _, sess := range m.temporarySessions {
		close(sess.kill)
		list = append(list, sess.done)
	}

	// closed owned stored sessions
	for _, sess := range m.storedSessions {
		if sess.owner != nil {
			close(sess.kill)
			list = append(list, sess.done)
		}
	}

	// release mutex to allow termination
	m.globalMutex.Unlock()

	// return early if empty
	if len(list) == 0 {
		return true
	}

	// prepare timeout
	tm := time.After(timeout)

	// wait for clients to close
	for _, ch := range list {
		select {
		case <-ch:
			continue
		case <-tm:
			return false
		}
	}

	return true
}
