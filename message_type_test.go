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

func TestMessageTypes(t *testing.T) {
	if CONNECT != 1 ||
		CONNACK != 2 ||
		PUBLISH != 3 ||
		PUBACK != 4 ||
		PUBREC != 5 ||
		PUBREL != 6 ||
		PUBCOMP != 7 ||
		SUBSCRIBE != 8 ||
		SUBACK != 9 ||
		UNSUBSCRIBE != 10 ||
		UNSUBACK != 11 ||
		PINGREQ != 12 ||
		PINGRESP != 13 ||
		DISCONNECT != 14 {

		t.Errorf("Message types have invalid code")
	}
}

func TestMessageTypeString(t *testing.T) {
	require.Equal(t, "UNKNOWN", MessageType(99).String())
}

func TestMessageTypeValid(t *testing.T) {
	require.True(t, CONNECT.Valid())
}

func TestMessageTypeNew(t *testing.T) {
	list := []MessageType{
		CONNECT,
		CONNACK,
		PUBLISH,
		PUBACK,
		PUBREC,
		PUBREL,
		PUBCOMP,
		SUBSCRIBE,
		SUBACK,
		UNSUBSCRIBE,
		UNSUBACK,
		PINGREQ,
		PINGRESP,
		DISCONNECT,
	}

	for _, mt := range list {
		m, err := mt.New()
		require.NotNil(t, m)
		require.NoError(t, err)
	}
}
