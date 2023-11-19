package torrent

import (
	"encoding/binary"
	"fmt"
	"io"
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
	buf := make([]byte, messageBytesSizeLength)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, fmt.Errorf("failed to read length data: %w", err)
	}

	dataLength := binary.BigEndian.Uint32(buf)

	if dataLength == 0 {
		return nil, nil
	}

	buf = make([]byte, dataLength)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, fmt.Errorf("failed to read payload data: %w", err)
	}

	return &Message{
		ID:      messageID(buf[0]),
		Payload: buf[1:],
	}, nil
}

func Marshall(message *Message) []byte {
	messageSize := messageBytesSizeLength + messageIDLength + len(message.Payload)
	usefulInfoSize := messageIDLength + len(message.Payload)
	res := make([]byte, messageSize)
	binary.BigEndian.PutUint32(res[:messageBytesSizeLength], uint32(usefulInfoSize))
	res[messageBytesSizeLength+messageIDLength-1] = byte(message.ID)
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

func (m *Message) ParseHave() (int, error) {
	if m.ID != MsgHave {
		return 0, fmt.Errorf("ParseHave: failed to parse message expected type %q current %q", MsgHave, m.ID)
	}
	if len(m.Payload) != 4 {
		return 0, fmt.Errorf("ParseHave: expected payload of length 4 of type %q", MsgHave)
	}
	return int(binary.BigEndian.Uint32(m.Payload)), nil
}

func (m *Message) ParsePiece(expectedPieceIndex int, buf []byte) (int, error) {
	if m.ID != MsgPiece {
		return 0, fmt.Errorf("ParsePiece: failed to parse message expected type %q current %q", MsgPiece, m.ID)
	}
	if len(m.Payload) < 8 {
		return 0, fmt.Errorf("ParsePiece: payload length must be more than 8 for type %q", MsgPiece)
	}
	parsedIndex := int(binary.BigEndian.Uint32(m.Payload[:4]))
	if parsedIndex != expectedPieceIndex {
		return 0, fmt.Errorf("ParsePiece: got wrong piece index %q expected %q", parsedIndex, expectedPieceIndex)
	}
	begin := int(binary.BigEndian.Uint32(m.Payload[4:8]))
	if begin >= len(buf) {
		return 0, fmt.Errorf("ParsePiece: buffer is too short to start reading message: needs at least %d size or more: current buffer length %d", begin, len(buf))
	}
	usefulData := m.Payload[8:]
	if begin+len(usefulData) > len(buf) {
		return 0, fmt.Errorf("ParsePiece: buffer is too short to read all useful data of message: needs at least %d size or more: current buffer length %d", begin+len(usefulData), len(buf))
	}

	return copy(buf[begin:], usefulData), nil
}
