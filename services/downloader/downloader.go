package downloader

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/hihoak/torrent-cli/client/torrent"
	"github.com/hihoak/torrent-cli/services/peers"
	"github.com/hihoak/torrent-cli/services/torrent-file-decoder"
	log "github.com/rs/zerolog/log"
	"os"
	"sync"
)

type PieceDownloader interface {
	DownloadPiece() (bool, error)
}

const (
	maxBlockSize = 16384
)

type workPiece struct {
	ID          int
	SizeOfPiece int
	Hash        [20]byte
}

type Downloader struct {
	torrentFile *torrent_file_decoder.TorrentFile

	peers []*peers.Peer

	todoChan chan workPiece
	doneChan chan workPiece

	buf []byte
}

func NewDownloader(torrentFile *torrent_file_decoder.TorrentFile, peers []*peers.Peer) *Downloader {
	return &Downloader{
		torrentFile: torrentFile,
		peers:       peers,
		todoChan:    make(chan workPiece, len(torrentFile.PieceHashes)),
		doneChan:    make(chan workPiece),
		buf:         make([]byte, torrentFile.Length),
	}
}

func (d *Downloader) calculateLengthForPiece(id int, sizeOfPiece int, totalLength int) int {
	start := id * sizeOfPiece
	end := start + sizeOfPiece
	if end > totalLength {
		end = totalLength
	}
	return end - start
}

func (d *Downloader) Download() error {
	for idx, hash := range d.torrentFile.PieceHashes {
		d.todoChan <- workPiece{
			ID:          idx,
			SizeOfPiece: d.calculateLengthForPiece(idx, d.torrentFile.PieceLength, d.torrentFile.Length),
			Hash:        hash,
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(d.peers))
	for _, peer := range d.peers {
		go func(peer *peers.Peer) {
			defer wg.Done()
			if err := d.downloadWorkerFunc(peer); err != nil {
				log.Error().Err(err).Msgf("stop downloading pieces from peer %v", peer)
			}
		}(peer)
	}

	go func() {
		wg.Wait()
		close(d.doneChan)
		close(d.todoChan)
	}()

	var countOfDonePieces int
	for ; countOfDonePieces < len(d.torrentFile.PieceHashes); countOfDonePieces++ {
		piece, ok := <-d.doneChan
		if !ok {
			break
		}
		log.Info().Msgf("piece %d/%d downloaded...", piece.ID, len(d.torrentFile.PieceHashes))
	}

	if countOfDonePieces == len(d.torrentFile.PieceHashes) {
		if err := d.saveFile(); err != nil {
			return fmt.Errorf("failed to save file to filesystem: %w", err)
		}
		log.Info().Msg("file is fully downloaded!")
		return nil
	}

	return fmt.Errorf("failed to download file: downloaded %d/%d of all pieces", countOfDonePieces, len(d.torrentFile.PieceHashes))
}

func (d *Downloader) savePiece(piece workPiece, downloader *pieceDownloader) {
	start := piece.ID * piece.SizeOfPiece
	end := start + piece.SizeOfPiece
	copy(d.buf[start:end], downloader.buf)
}

func (d *Downloader) saveFile() error {
	file, err := os.OpenFile("saved_file", os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("failed to close file")
		}
	}()
	if _, writeErr := file.Write(d.buf); writeErr != nil {
		return fmt.Errorf("failed to write data to file: %w", writeErr)
	}
	return nil
}

func isValidPieceHash(buf []byte, piece workPiece) bool {
	pieceHash := sha1.Sum(buf)
	return bytes.Equal(pieceHash[:], piece.Hash[:])
}

func (d *Downloader) downloadWorkerFunc(peer *peers.Peer) error {
	client, err := torrent.NewClient(d.torrentFile, peer)
	if err != nil {
		return fmt.Errorf("failed to init client from peer %v: %w", peer, err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			fmt.Println("failed to close connection to peer:", closeErr)
		}
	}()

	if unchokeErr := client.SendUnchoke(); unchokeErr != nil {
		return fmt.Errorf("failed to unchoke client: %w", unchokeErr)
	}
	if interestedErr := client.SendInterested(); interestedErr != nil {
		return fmt.Errorf("failed to send interest to client: %w", interestedErr)
	}

	for piece := range d.todoChan {
		if !client.HasPieceToDownload(piece.ID) {
			d.todoChan <- piece
			continue
		}
		downloader := NewPieceDownloader(client, piece)
		downloadErr := downloader.DownloadPiece()
		if downloadErr != nil {
			d.todoChan <- piece
			return fmt.Errorf("failed to download piece %v: %w", piece, downloadErr)
		}
		if !isValidPieceHash(downloader.buf, piece) {
			log.Error().Msgf("failed to download piece %v because hash is not equal to expected", piece)
			d.todoChan <- piece
			continue
		}
		log.Debug().Msgf("successfully download piece: %v", piece)
		d.doneChan <- piece
		d.savePiece(piece, downloader)
	}

	return nil
}
