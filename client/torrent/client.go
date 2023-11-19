package torrent

import (
	"fmt"
	"github.com/hihoak/torrent-cli/services/peers"
	torrent_file_decoder "github.com/hihoak/torrent-cli/services/torrent-file-decoder"
	log "github.com/rs/zerolog/log"
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
	PeerID   string

	Chocked bool
}

func processHandshake(conn net.Conn, verifyHash [20]byte) (*torrentProtocolHandshake, error) {
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to send deadline timeout: %w", err)
	}
	defer func() {
		if err := conn.SetDeadline(time.Time{}); err != nil {
			log.Error().Err(err).Msg("failed to set deadline")
		}
	}()

	if handshakeErr := SendHandshake(conn, verifyHash, peers.MyPeerID); handshakeErr != nil {
		return nil, fmt.Errorf("failed to send handshake: %w", handshakeErr)
	}

	handshake, handshakeErr := RecvHandshake(conn, verifyHash)
	if handshakeErr != nil {
		return nil, fmt.Errorf("failed to recv handshake: %w", handshakeErr)
	}

	return handshake, nil
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
	fmt.Println("start initializing connect to:", peer.IP.String())
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", peer.IP.String(), peer.Port), 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to init connection to peer %v: %w", peer, err)
	}

	handshake, handshakeErr := processHandshake(conn, torrentFile.VerifyHash)
	if handshakeErr != nil {
		return nil, fmt.Errorf("failed to process handshake: %w", handshakeErr)
	}

	bitField, bitFieldErr := getPeersBitfield(conn)
	if bitFieldErr != nil {
		return nil, fmt.Errorf("failed to retrieve bitfield: %w", bitFieldErr)
	}

	fmt.Println("successfully established connection to:", peer.IP.String())
	return &Client{
		conn:     conn,
		bitfield: bitField,
		PeerID:   string(handshake.PeerID[:]),
		Chocked:  true,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) HasPieceToDownload(id int) bool {
	return c.bitfield.HasPiece(id)
}

func (c *Client) SetPieceToDownload(id int) {
	c.bitfield.SetPiece(id)
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

func (c *Client) SendRequest(index, begin, length int) error {
	message := CreateRequestMessage(index, begin, length)
	_, err := c.conn.Write(Marshall(message))
	return err
}

func (c *Client) ReadMessage() (*Message, error) {
	return UnmarshallMessage(c.conn)
}
