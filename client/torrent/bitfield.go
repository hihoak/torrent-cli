package torrent

type Bitfield []byte

func (b Bitfield) HasPiece(id int) bool {
	byteID := id / 8
	bitOffset := id % 8
	return b[byteID]>>(7-bitOffset)&1 != 0
}

func (b Bitfield) SetPiece(id int) {
	byteID := id / 8
	bitOffset := id % 8
	b[byteID] |= 1 << (7 - bitOffset)
}
