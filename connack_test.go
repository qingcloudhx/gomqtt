// Copyright (c) 2014 The gomqtt Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package message

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnackReturnCodes(t *testing.T) {
	require.Equal(t, ErrInvalidProtocolVersion.Error(), ConnackCode(1).Error())
	require.Equal(t, ErrIdentifierRejected.Error(), ConnackCode(2).Error())
	require.Equal(t, ErrServerUnavailable.Error(), ConnackCode(3).Error())
	require.Equal(t, ErrBadUsernameOrPassword.Error(), ConnackCode(4).Error())
	require.Equal(t, ErrNotAuthorized.Error(), ConnackCode(5).Error())
	require.Equal(t, "Unknown error", ConnackCode(6).Error())
}

func TestConnackInterface(t *testing.T) {
	msg := NewConnackMessage()

	require.Equal(t, msg.Type(), CONNACK)
	require.NotNil(t, msg.String())
}

func TestConnackMessageDecode(t *testing.T) {
	msgBytes := []byte{
		byte(CONNACK << 4),
		2,
		0, // session not present
		0, // connection accepted
	}

	msg := NewConnackMessage()

	n, err := msg.Decode(msgBytes)

	require.NoError(t, err)
	require.Equal(t, 4, n)
	require.False(t, msg.SessionPresent)
	require.Equal(t, ConnectionAccepted, msg.ReturnCode)
}

func TestConnackMessageDecodeError1(t *testing.T) {
	msgBytes := []byte{
		byte(CONNACK << 4),
		3, // <- wrong size
		0, // session not present
		0, // connection accepted
	}

	msg := NewConnackMessage()

	_, err := msg.Decode(msgBytes)
	require.Error(t, err)
}

func TestConnackMessageDecodeError2(t *testing.T) {
	msgBytes := []byte{
		byte(CONNACK << 4),
		2,
		0, // session not present
		// <- wrong message size
	}

	msg := NewConnackMessage()

	_, err := msg.Decode(msgBytes)
	require.Error(t, err)
}

func TestConnackMessageDecodeError3(t *testing.T) {
	msgBytes := []byte{
		byte(CONNACK << 4),
		2,
		64, // <- wrong value
		0,  // connection accepted
	}

	msg := NewConnackMessage()

	_, err := msg.Decode(msgBytes)
	require.Error(t, err)
}

func TestConnackMessageDecodeError4(t *testing.T) {
	msgBytes := []byte{
		byte(CONNACK << 4),
		2,
		0,
		6, // <- wrong code
	}

	msg := NewConnackMessage()

	_, err := msg.Decode(msgBytes)
	require.Error(t, err)
}

func TestConnackMessageDecodeError5(t *testing.T) {
	msgBytes := []byte{
		byte(CONNACK << 4),
		1, // <- wrong remaining length
		0,
		6,
	}

	msg := NewConnackMessage()

	_, err := msg.Decode(msgBytes)
	require.Error(t, err)
}

func TestConnackMessageEncode(t *testing.T) {
	msgBytes := []byte{
		byte(CONNACK << 4),
		2,
		1, // session present
		0, // connection accepted
	}

	msg := NewConnackMessage()
	msg.ReturnCode = ConnectionAccepted
	msg.SessionPresent = true

	dst := make([]byte, msg.Len())
	n, err := msg.Encode(dst)

	require.NoError(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, msgBytes, dst[:n])
}

func TestConnackMessageEncodeError1(t *testing.T) {
	msg := NewConnackMessage()

	dst := make([]byte, 3) // <- wrong buffer size
	n, err := msg.Encode(dst)

	require.Error(t, err)
	require.Equal(t, 0, n)
}

func TestConnackMessageEncodeError2(t *testing.T) {
	msg := NewConnackMessage()
	msg.ReturnCode = 11 // <- wrong return code

	dst := make([]byte, msg.Len())
	n, err := msg.Encode(dst)

	require.Error(t, err)
	require.Equal(t, 3, n)
}

func TestConnackEqualDecodeEncode(t *testing.T) {
	msgBytes := []byte{
		byte(CONNACK << 4),
		2,
		0, // session not present
		0, // connection accepted
	}

	msg := NewConnackMessage()
	n, err := msg.Decode(msgBytes)

	require.NoError(t, err)
	require.Equal(t, 4, n)

	dst := make([]byte, msg.Len())
	n2, err := msg.Encode(dst)

	require.NoError(t, err)
	require.Equal(t, 4, n2)
	require.Equal(t, msgBytes, dst[:n2])

	n3, err := msg.Decode(dst)

	require.NoError(t, err)
	require.Equal(t, 4, n3)
}

func BenchmarkConnackEncode(b *testing.B) {
	msg := NewConnackMessage()
	msg.ReturnCode = ConnectionAccepted
	msg.SessionPresent = true

	buf := make([]byte, msg.Len())

	for i := 0; i < b.N; i++ {
		_, err := msg.Encode(buf)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkConnackDecode(b *testing.B) {
	msgBytes := []byte{
		byte(CONNACK << 4),
		2,
		0, // session not present
		0, // connection accepted
	}

	msg := NewConnackMessage()

	for i := 0; i < b.N; i++ {
		_, err := msg.Decode(msgBytes)
		if err != nil {
			panic(err)
		}
	}
}
