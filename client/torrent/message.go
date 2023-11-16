package torrent

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
)

type messageID byte

const (
	MsgChoke messageID = iota
	MsgUnchoke
	MsgInterested
	MsgNotInterested
	MsgHave
	MsgBitfield
	MsgRequest
	MsgPiece
	MsgCancel

	messageBytesSizeLength = 4
	messageIDLength        = 1
)

type Message struct {
	ID      messageID
	Payload []byte
}

func UnmarshallMessage(reader io.Reader) (*Message, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	if len(data) < messageBytesSizeLength+messageIDLength {
		return nil, fmt.Errorf("too short response message must be at least %d bytes long", messageBytesSizeLength+messageIDLength)
	}
	dataLength := binary.BigEndian.Uint32(data[0:messageBytesSizeLength])

	msgID, err := strconv.Atoi(string(data[messageBytesSizeLength+messageIDLength]))
	if err != nil {
		return nil, fmt.Errorf("message id must be integer")
	}

	return &Message{
		ID:      messageID(msgID),
		Payload: data[messageBytesSizeLength+messageIDLength : dataLength],
	}, nil
}

func Marshall(message *Message) []byte {
	messageSize := messageBytesSizeLength + messageIDLength + len(message.Payload)
	res := make([]byte, messageSize)
	binary.BigEndian.PutUint32(res[:messageBytesSizeLength], uint32(messageSize))
	copy(res[messageBytesSizeLength:messageBytesSizeLength+messageIDLength], []byte{byte(message.ID)})
	copy(res[messageBytesSizeLength+messageIDLength:], message.Payload)
	return res
}

// FormatRequest creates a REQUEST message
func CreateRequestMessage(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &Message{ID: MsgRequest, Payload: payload}
}

func CreateBitfieldMessage() *Message {
	return &Message{ID: MsgBitfield}
}

func CreateInterestedMessage() *Message {
	return &Message{ID: MsgInterested}
}

func CreateUnchokeMessage() *Message {
	return &Message{ID: MsgUnchoke}
}

func CreateHaveMessage(pieceID int) *Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(pieceID))
	return &Message{ID: MsgHave, Payload: payload}
}
