package torrent

import (
	"fmt"
	"io"
	"strconv"
)

type TorrentProtocolHandshake struct {
	FileVerifyHash    [fileVerifyHashLength]byte
	PeerID            [peerIDLength]byte
	AdditionalOptions [additionalOptionsLength]byte
}

func MarshallHandshake(t TorrentProtocolHandshake) []byte {
	//lenOfMessage := 1 + len(torrentIdentifier) + additionalOptionsLength + fileVerifyHashLength + peerIDLength
	//res := make([]byte, lenOfMessage)
	//res[0] = byte(lenOfMessage)
	//length := 1
	//length += copy(res[length:], torrentIdentifier)
	//length += copy(res[length:], t.AdditionalOptions[:])
	//length += copy(res[length:], t.FileVerifyHash[:])
	//length += copy(res[length:], t.PeerID[:])
	buf := make([]byte, len(torrentIdentifier)+49)
	buf[0] = byte(len(torrentIdentifier))
	curr := 1
	curr += copy(buf[curr:], torrentIdentifier)
	curr += copy(buf[curr:], make([]byte, 8)) // 8 reserved bytes
	curr += copy(buf[curr:], t.FileVerifyHash[:])
	curr += copy(buf[curr:], t.PeerID[:])
	return buf
}

func UnmarshallHandshake(reader io.Reader) (*TorrentProtocolHandshake, error) {
	torrentHandshakeRaw := make([]byte, 1)
	var n int
	var err error
	for idx := 0; idx < 3; idx++ {
		n, err = io.ReadFull(reader, torrentHandshakeRaw)
		if err != nil {
			continue
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read handshake response: %w", err)
	}
	torrentHandshakeRaw = torrentHandshakeRaw[:n]
	if len(torrentHandshakeRaw) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	responseLength, err := strconv.Atoi(string(torrentHandshakeRaw[0]))
	if err != nil {
		return nil, fmt.Errorf("first byte must be integer because it is define length of message: %w", err)
	}
	if responseLength < len(torrentIdentifier) {
		return nil, fmt.Errorf("too short response: can't even find torrent protocol identifier")
	}
	if string(torrentHandshakeRaw[1:1+len(torrentIdentifier)]) != torrentIdentifier {
		return nil, fmt.Errorf("wrong protocol identifier got %q expect %q", string(torrentHandshakeRaw[1:1+len(torrentIdentifier)]), torrentIdentifier)
	}

	// validate additional options
	if responseLength < len(torrentIdentifier)+additionalOptionsLength {
		return nil, fmt.Errorf("too short response: can't even find torrent additional options")
	}
	if responseLength < len(torrentIdentifier)+additionalOptionsLength+fileVerifyHashLength {
		return nil, fmt.Errorf("too short response: can't even find torrent file to download verify hash")
	}
	if responseLength < len(torrentIdentifier)+additionalOptionsLength+fileVerifyHashLength+peerIDLength {
		return nil, fmt.Errorf("too short response: can't even find torrent peer ID")
	}

	return &TorrentProtocolHandshake{
		AdditionalOptions: [additionalOptionsLength]byte(torrentHandshakeRaw[1+len(torrentIdentifier) : 1+len(torrentIdentifier)+additionalOptionsLength]),
		FileVerifyHash:    [fileVerifyHashLength]byte(torrentHandshakeRaw[1+len(torrentIdentifier)+additionalOptionsLength : 1+len(torrentIdentifier)+additionalOptionsLength+fileVerifyHashLength]),
		PeerID:            [peerIDLength]byte(torrentHandshakeRaw[1+len(torrentIdentifier)+additionalOptionsLength+fileVerifyHashLength:]),
	}, nil
}
