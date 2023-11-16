package main

import (
	"fmt"
	"github.com/hihoak/torrent-cli/client/torrent"
	"github.com/hihoak/torrent-cli/services/peers"
	torrent_decoder "github.com/hihoak/torrent-cli/services/torrent-file-decoder"
	"log"
	"os"
)

func main() {
	data, _ := os.Open("fifa14.torrent")
	file, err := torrent_decoder.Unmarshall(data)
	if err != nil {
		log.Fatal("failed to create torrent file:", err)
	}

	peers, err := peers.GetPeers(file)
	if err != nil {
		log.Fatal("failed to get peers: ", err)
	}

	clients := make([]*torrent.Client, 0)
	for _, peer := range peers {
		client, handshakeErr := torrent.NewClient(file, peer)
		if handshakeErr != nil {
			log.Println("failed to proccess handshale:", handshakeErr)
			continue
		}
		clients = append(clients, client)
	}

	fmt.Println(clients)
}
