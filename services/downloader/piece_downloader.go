package downloader

import (
	"fmt"
	"github.com/hihoak/torrent-cli/client/torrent"
)

const (
	maxParallelRequests = 4
)

type pieceDownloader struct {
	client *torrent.Client

	piece workPiece
	buf   []byte

	bytesDownloaded int
	bytesRequested  int

	parallelRequests int
}

func NewPieceDownloader(client *torrent.Client, piece workPiece) *pieceDownloader {
	return &pieceDownloader{
		client:           client,
		piece:            piece,
		buf:              make([]byte, piece.SizeOfPiece),
		bytesRequested:   0,
		bytesDownloaded:  0,
		parallelRequests: 0,
	}
}

func (p *pieceDownloader) DownloadPiece() error {
	for p.bytesDownloaded < p.piece.SizeOfPiece {
		if !p.client.Chocked {
			for p.parallelRequests < maxParallelRequests && p.bytesRequested < p.piece.SizeOfPiece {
				blockSize := maxBlockSize
				if blockSize > p.piece.SizeOfPiece-p.bytesRequested {
					blockSize = p.piece.SizeOfPiece - p.bytesRequested
				}
				if err := p.client.SendRequest(p.piece.ID, p.bytesRequested, blockSize); err != nil {
					return fmt.Errorf("failed to send request for a block of a piece to peer %s: %w", p.client.PeerID, err)
				}
				p.bytesRequested += blockSize
				p.parallelRequests++
			}
		}

		if err := p.read(); err != nil {
			return fmt.Errorf("piece worker fails to read incoming message: %w", err)
		}
	}

	return nil
}

func (p *pieceDownloader) read() error {
	msg, err := p.client.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read message from peer %s: %w", p.client.PeerID, err)
	}

	if msg == nil {
		return nil
	}

	switch msg.ID {
	case torrent.MsgChoke:
		//fmt.Printf("received message choke from peer %s\n", p.client.PeerID)
		p.client.Chocked = true
	case torrent.MsgUnchoke:
		//fmt.Printf("received message unchoke from peer %s\n", p.client.PeerID)
		p.client.Chocked = false
	case torrent.MsgHave:
		//fmt.Printf("received message have from peer %s\n", p.client.PeerID)
		index, parseErr := msg.ParseHave()
		if parseErr != nil {
			return fmt.Errorf("failed to parse %d message received from peer %s: %w", torrent.MsgHave, p.client.PeerID, parseErr)
		}
		p.client.SetPieceToDownload(index)
	case torrent.MsgPiece:
		//fmt.Printf("received message piece from peer %s\n", p.client.PeerID)
		n, parseErr := msg.ParsePiece(p.piece.ID, p.buf)
		if parseErr != nil {
			return fmt.Errorf("failed to parse %d message received from peer %s: %w", torrent.MsgPiece, p.client.PeerID, parseErr)
		}
		p.bytesDownloaded += n
		p.parallelRequests--
	default:
		fmt.Println("some other status")
	}

	return nil
}
