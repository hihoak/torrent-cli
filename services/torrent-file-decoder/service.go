package torrent_file_decoder

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/jackpal/bencode-go"
	"io"
	"net/url"
	"strconv"
)

type bencodeTorrentFile struct {
	Announce string             `bencode:"announce"`
	Info     bencodeTorrentInfo `bencode:"info"`
}

type bencodeTorrentInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

func (b *bencodeTorrentFile) CalculateVerifyHash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, b.Info)
	if err != nil {
		return [20]byte{}, fmt.Errorf("failed to marshall torrent info into bencode: %w", err)
	}
	return sha1.Sum(buf.Bytes()), nil
}

func (b *bencodeTorrentFile) convertPiecesToSlice() ([][20]byte, error) {
	if len(b.Info.Pieces)%20 != 0 {
		return nil, fmt.Errorf("wrong hash pieces length: must deleting on 20")
	}
	res := make([][20]byte, 0, len(b.Info.Pieces)/20)
	for left := 0; left < len(b.Info.Pieces); left += 20 {
		right := left + 20
		res = append(res, [20]byte([]byte(b.Info.Pieces[left:right])))
	}
	return res, nil
}

func (b *bencodeTorrentFile) toTorrentFile() (*TorrentFile, error) {
	pieceHashes, err := b.convertPiecesToSlice()
	if err != nil {
		return nil, fmt.Errorf("failed convert bencode torrent file to torrent file struct: %w", err)
	}
	verifyHash, err := b.CalculateVerifyHash()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate verify hash: %w", err)
	}
	res := TorrentFile{
		Announce:    b.Announce,
		VerifyHash:  verifyHash,
		PieceHashes: pieceHashes,
		PieceLength: b.Info.PieceLength,
		Length:      b.Info.Length,
		Name:        b.Info.Name,
	}

	return &res, nil
}

type TorrentFile struct {
	Announce    string
	VerifyHash  [20]byte
	PieceHashes [][20]byte
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

func Unmarshall(data io.Reader) (*TorrentFile, error) {
	var res bencodeTorrentFile
	if err := bencode.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshall data from bencode encoding: %w", err)
	}
	return res.toTorrentFile()
}

func (t *TorrentFile) BuildTrackerURL(peerID [20]byte) (string, error) {
	base, err := url.Parse(t.Announce)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL %q: %w", t.Announce, err)
	}

	port := base.Port()
	if port == "" {
		switch base.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			port = "80"
		}
	}
	params := url.Values{
		"info_hash":  []string{string(t.VerifyHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{port},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.Length)},
	}
	base.RawQuery = params.Encode()
	return base.String(), nil
}
