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
	"encoding/binary"
	"fmt"
)

// A SUBACK Packet is sent by the Server to the Client to confirm receipt and processing
// of a SUBSCRIBE Packet. The SUBACK Packet contains a list of return codes, that specify
// the maximum QOS level that have been granted.
type SubackMessage struct {
	// The granted QOS levels for the requested subscriptions.
	ReturnCodes []byte

	// Shared message identifier.
	PacketID uint16
}

var _ Message = (*SubackMessage)(nil)

// NewSubackMessage creates a new SUBACK message.
func NewSubackMessage() *SubackMessage {
	return &SubackMessage{}
}

// Type return the messages message type.
func (sm SubackMessage) Type() MessageType {
	return SUBACK
}

// String returns a string representation of the message.
func (sm SubackMessage) String() string {
	return fmt.Sprintf("SUBACK: PacketID=%d ReturnCodes=%v", sm.PacketID, sm.ReturnCodes)
}

// Len returns the byte length of the message.
func (sm *SubackMessage) Len() int {
	ml := sm.len()
	return headerLen(ml) + ml
}

// Decode reads from the byte slice argument. It returns the total number of bytes
// decoded, and whether there have been any errors during the process.
// The byte slice MUST NOT be modified during the duration of this
// message being available since the byte slice never gets copied.
func (sm *SubackMessage) Decode(src []byte) (int, error) {
	total := 0

	// decode header
	hl, _, rl, err := headerDecode(src[total:], SUBACK)
	total += hl
	if err != nil {
		return total, err
	}

	// check buffer length
	if len(src) < total+2 {
		return total, fmt.Errorf("SUBACK/Decode: Insufficient buffer size. Expecting %d, got %d", total+2, len(src))
	}

	// check remaining length
	if rl <= 2 {
		return total, fmt.Errorf("SUBACK/Decode: Expected remaining length to be greater than 2, got %d", rl)
	}

	// read packet id
	sm.PacketID = binary.BigEndian.Uint16(src[total:])
	total += 2

	// calculate number of return codes
	rcl := int(rl) - 2

	// read return codes
	sm.ReturnCodes = src[total : total+rcl]
	total += len(sm.ReturnCodes)

	// validate return codes
	for i, code := range sm.ReturnCodes {
		if !validQOS(code) && code != QOSFailure {
			return total, fmt.Errorf("SUBACK/Decode: Invalid return code %d for topic %d", code, i)
		}
	}

	return total, nil
}

// Encode writes the message bytes into the byte array from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (sm *SubackMessage) Encode(dst []byte) (int, error) {
	total := 0

	// check return codes
	for i, code := range sm.ReturnCodes {
		if !validQOS(code) && code != QOSFailure {
			return total, fmt.Errorf("SUBACK/Encode: Invalid return code %d for topic %d", code, i)
		}
	}

	// encode header
	n, err := headerEncode(dst[total:], 0, sm.len(), sm.Len(), SUBACK)
	total += n
	if err != nil {
		return total, err
	}

	// write packet id
	binary.BigEndian.PutUint16(dst[total:], sm.PacketID)
	total += 2

	// write return codes
	copy(dst[total:], sm.ReturnCodes)
	total += len(sm.ReturnCodes)

	return total, nil
}

// Returns the payload length.
func (sm *SubackMessage) len() int {
	return 2 + len(sm.ReturnCodes)
}
