package torrent

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

type torrentProtocolHandshake struct {
	fileVerifyHash    [fileVerifyHashLength]byte
	PeerID            [peerIDLength]byte
	additionalOptions [additionalOptionsLength]byte
}

func MarshallHandshake(t torrentProtocolHandshake) []byte {
	lenOfMessage := 1 + len(torrentIdentifier) + additionalOptionsLength + fileVerifyHashLength + peerIDLength
	res := make([]byte, lenOfMessage)
	res[0] = byte(len(torrentIdentifier))
	length := 1
	length += copy(res[length:], torrentIdentifier)
	length += copy(res[length:], t.additionalOptions[:])
	length += copy(res[length:], t.fileVerifyHash[:])
	length += copy(res[length:], t.PeerID[:])
	return res
}

func SendHandshake(conn net.Conn, fileVerifyHash, peerID [20]byte) error {
	if _, handshakeErr := conn.Write(MarshallHandshake(torrentProtocolHandshake{
		fileVerifyHash: fileVerifyHash,
		PeerID:         peerID,
	})); handshakeErr != nil {
		return fmt.Errorf("failed to start handshake: %w", handshakeErr)
	}
	return nil
}

func RecvHandshake(reader io.Reader, fileVerifyHash [20]byte) (*torrentProtocolHandshake, error) {
	torrentHandshakeRaw := make([]byte, 1)
	n, err := io.ReadFull(reader, torrentHandshakeRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to read handshake response: %w", err)
	}

	if n == 0 {
		return nil, fmt.Errorf("got empty data in handshake response")
	}

	protocolIdentifierLength := int(torrentHandshakeRaw[0])
	if protocolIdentifierLength != len(torrentIdentifier) {
		return nil, fmt.Errorf("too short response: can't even find torrent protocol identifier")
	}

	responseLength := len(torrentIdentifier) + additionalOptionsLength + fileVerifyHashLength + peerIDLength
	torrentHandshakeRaw = make([]byte, responseLength)
	if _, err = io.ReadFull(reader, torrentHandshakeRaw); err != nil {
		return nil, fmt.Errorf("failed to read other data from client: %w", err)
	}

	resHandshake := torrentProtocolHandshake{}
	respTorrentIdentifier := make([]byte, len(torrentIdentifier))
	idx := copy(respTorrentIdentifier, torrentHandshakeRaw[:len(torrentIdentifier)])
	idx += copy(resHandshake.additionalOptions[:], torrentHandshakeRaw[idx:idx+additionalOptionsLength])
	idx += copy(resHandshake.fileVerifyHash[:], torrentHandshakeRaw[idx:idx+fileVerifyHashLength])
	copy(resHandshake.PeerID[:], torrentHandshakeRaw[idx:])

	if !bytes.Equal(respTorrentIdentifier, []byte(torrentIdentifier)) {
		return nil, fmt.Errorf("wrong protocol identifier got %q expect %q", string(respTorrentIdentifier), torrentIdentifier)
	}

	if !bytes.Equal(resHandshake.fileVerifyHash[:], fileVerifyHash[:]) {
		return nil, fmt.Errorf("got wrong file hash in handshake response: got %q expect %q", string(resHandshake.fileVerifyHash[:]), string(fileVerifyHash[:]))
	}

	return &resHandshake, nil
}
