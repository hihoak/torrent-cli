package torrent

import (
	"fmt"
	"github.com/hihoak/torrent-cli/services/peers"
	torrent_file_decoder "github.com/hihoak/torrent-cli/services/torrent-file-decoder"
	"net"
	"time"
)

const (
	torrentIdentifier       = "BitTorrent protocol"
	additionalOptionsLength = 8
	fileVerifyHashLength    = 20
	peerIDLength            = 20
)

type Client struct {
	conn     net.Conn
	bitfield Bitfield
}

func processHandshake(conn net.Conn, verifyHash [20]byte) error {
	//if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
	//	return fmt.Errorf("failed to send deadline timeout: %w", err)
	//}
	//defer conn.SetDeadline(time.Time{})

	torrentHandShake := TorrentProtocolHandshake{
		FileVerifyHash: verifyHash,
		PeerID:         peers.MyPeerID,
	}
	if _, handshakeErr := conn.Write(MarshallHandshake(torrentHandShake)); handshakeErr != nil {
		return fmt.Errorf("failed to start handshake: %w", handshakeErr)
	}

	gotHandshake, unmarshallErr := UnmarshallHandshake(conn)
	if unmarshallErr != nil {
		return fmt.Errorf("failed to unmarshall got handshake: %w", unmarshallErr)
	}

	if torrentHandShake.FileVerifyHash != gotHandshake.FileVerifyHash {
		return fmt.Errorf("handshake is failed: file hashed are not equal")
	}

	if torrentHandShake.PeerID != gotHandshake.PeerID {
		return fmt.Errorf("handshake is failed: peer IDs is not equal")
	}

	return nil
}

func getPeersBitfield(conn net.Conn) (Bitfield, error) {
	message, err := UnmarshallMessage(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read bitfield message: %w", err)
	}

	if message.ID != MsgBitfield {
		return nil, fmt.Errorf("wrong message ID %s expect %s", message.ID, MsgBitfield)
	}

	return message.Payload, nil
}

func NewClient(torrentFile *torrent_file_decoder.TorrentFile, peer *peers.Peer) (*Client, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", peer.IP.String(), peer.Port), time.Second*2)
	if err != nil {
		return nil, fmt.Errorf("failed to init connection to peer %v: %w", peer, err)
	}

	if handshakeErr := processHandshake(conn, torrentFile.VerifyHash); handshakeErr != nil {
		return nil, fmt.Errorf("failed to process handshake: %w", handshakeErr)
	}

	bitField, bitFieldErr := getPeersBitfield(conn)
	if bitFieldErr != nil {
		return nil, fmt.Errorf("failed to retrieve bitfield: %w", bitFieldErr)
	}

	return &Client{
		conn:     conn,
		bitfield: bitField,
	}, nil
}

func (c *Client) SendHave(pieceID int) error {
	msg := CreateHaveMessage(pieceID)
	if _, err := c.conn.Write(Marshall(msg)); err != nil {
		return fmt.Errorf("failed to send %q message to client: %w", MsgHave, err)
	}
	return nil
}

func (c *Client) SendUnchoke() error {
	msg := CreateUnchokeMessage()
	if _, err := c.conn.Write(Marshall(msg)); err != nil {
		return fmt.Errorf("failed to send %q message to client: %w", MsgUnchoke, err)
	}
	return nil
}

func (c *Client) SendInterested() error {
	msg := CreateInterestedMessage()
	if _, err := c.conn.Write(Marshall(msg)); err != nil {
		return fmt.Errorf("failed to send %q message to client: %w", MsgInterested, err)
	}
	return nil
}
