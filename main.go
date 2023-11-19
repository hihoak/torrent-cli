package main

import (
	"github.com/hihoak/torrent-cli/services/downloader"
	"github.com/hihoak/torrent-cli/services/peers"
	torrent_decoder "github.com/hihoak/torrent-cli/services/torrent-file-decoder"
	log "github.com/rs/zerolog/log"
	"os"
	"time"
)

func main() {
	startOfProgram := time.Now()
	data, _ := os.Open("RPG_End_of_Aspiration.rar.torrent")
	file, err := torrent_decoder.Unmarshall(data)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create torrent file")
	}

	torrentPeers, err := peers.GetPeers(file)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get peers")
	}

	download := downloader.NewDownloader(file, torrentPeers)
	if downloadErr := download.Download(); downloadErr != nil {
		log.Fatal().Err(downloadErr).Msg("failed to download file")
	}
	timeOfExecutionSeconds := time.Now().Sub(startOfProgram).Seconds()
	fileSizeMb := float64(file.Length) / 1024 / 1024
	averageSpeedMbPerSecond := fileSizeMb / timeOfExecutionSeconds
	log.Info().Msgf("program executed for %f second. Downloaded %f Mb. Average speed: %f Mb/second", timeOfExecutionSeconds, fileSizeMb, averageSpeedMbPerSecond)
}
