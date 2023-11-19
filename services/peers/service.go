package peers

import (
	"encoding/binary"
	"fmt"
	torrent_decoder "github.com/hihoak/torrent-cli/services/torrent-file-decoder"
	"github.com/jackpal/bencode-go"
	log "github.com/rs/zerolog/log"
	"io"
	"math/rand"
	"net"
	"net/http"
	"time"
)

const (
	peerIPLengthBytes   = 4
	peerPortLengthBytes = 2
	peerAddressLength   = 6
)

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

var MyPeerID = getID()

func getID() [20]byte {
	id := make([]byte, 20)
	_, err := rand.Read(id[:])
	if err != nil {
		log.Fatal().Err(err).Msg("failed to generate ID")
	}
	return [20]byte(id)
}

type bencodePeers struct {
	Peers string `bencode:"peers"`
}

func (b *bencodePeers) convertToPeers() ([]*Peer, error) {
	if len(b.Peers)%peerAddressLength != 0 {
		return nil, fmt.Errorf("invalid peers length %d: length must deviding by %d", len(b.Peers), peerAddressLength)
	}
	res := make([]*Peer, 0, len(b.Peers)/peerAddressLength)
	for left := 0; left < len(b.Peers); left += peerAddressLength {
		ipRaw := []byte(b.Peers[left : left+peerIPLengthBytes])
		port := binary.BigEndian.Uint16([]byte(b.Peers[left+peerIPLengthBytes : left+peerIPLengthBytes+peerPortLengthBytes]))
		res = append(res, &Peer{
			IP:   net.IPv4(ipRaw[0], ipRaw[1], ipRaw[2], ipRaw[3]),
			Port: port,
		})
	}
	return res, nil
}

type Peer struct {
	IP   net.IP
	Port uint16
}

func GetPeers(torrentFile *torrent_decoder.TorrentFile) ([]*Peer, error) {
	trackerURL, err := torrentFile.BuildTrackerURL(MyPeerID)
	if err != nil {
		return nil, fmt.Errorf("failed to build tracker URL for find PEERS: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, trackerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request to get a peers: %w", err)
	}

	client := http.Client{
		Timeout: time.Second * 30,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get peers: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		responseErr := fmt.Errorf("failed to get peers: status code %d", resp.StatusCode)
		if resp.Body != nil {
			additionalInfo, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				log.Error().Err(readErr).Msg("failed to read body of response")
			}
			responseErr = fmt.Errorf("%w: %s", responseErr, string(additionalInfo))
		}
		return nil, responseErr
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("failed to close connection")
		}
	}()

	bencodePeersData := bencodePeers{}
	unmarshallErr := bencode.Unmarshal(resp.Body, &bencodePeersData)
	if unmarshallErr != nil {
		return nil, fmt.Errorf("failed to unmarshall peers response from bencode encoding to Peers structure: %w", unmarshallErr)
	}

	peers, convertToPeers := bencodePeersData.convertToPeers()
	if convertToPeers != nil {
		return nil, fmt.Errorf("failed to convert encoded peers to peers structure: %w", convertToPeers)
	}

	return peers, nil
}
