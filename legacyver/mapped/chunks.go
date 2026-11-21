package mapped

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Chunk payload conversion. The backend sends version-9 sub-chunks whose palettes hold block hashes, followed
// by one 3D biome storage per section. 1.18 keeps that layout with its own runtime IDs. 1.16 needs version-8
// sub-chunks for Y 0-255 only, and one biome byte per column.

const storageCells = 4096

// mapID maps one palette value. It always produces a value: unknown blocks have already become air.
type mapID func(uint32) uint32

// storage is one decoded paletted storage: the value of each of its 4096 cells.
type storage [storageCells]uint32

func readStorage(r *bytes.Reader, previous *storage) (*storage, error) {
	header, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if header == 0xff {
		// Biome sections may reuse the previous section's storage.
		if previous == nil {
			return nil, fmt.Errorf("storage reuse marker without a previous storage")
		}
		copied := *previous
		return &copied, nil
	}
	bits := int(header >> 1)
	if header&1 == 0 || !validBits(bits) {
		return nil, fmt.Errorf("invalid network storage header %d", header)
	}
	var words []uint32
	if bits != 0 {
		perWord := 32 / bits
		words = make([]uint32, (storageCells+perWord-1)/perWord)
		if err := binary.Read(r, binary.LittleEndian, words); err != nil {
			return nil, err
		}
	}
	count := int32(1)
	if bits != 0 {
		if err := protocol.Varint32(r, &count); err != nil {
			return nil, err
		}
	}
	if count < 1 || count > storageCells || (bits != 0 && count > 1<<bits) {
		return nil, fmt.Errorf("invalid palette length %d", count)
	}
	palette := make([]uint32, count)
	for i := range palette {
		var value int32
		if err := protocol.Varint32(r, &value); err != nil {
			return nil, err
		}
		palette[i] = uint32(value)
	}
	s := new(storage)
	for cell := range storageCells {
		index := uint32(0)
		if bits != 0 {
			perWord := 32 / bits
			index = (words[cell/perWord] >> (uint(cell%perWord) * uint(bits))) & (1<<bits - 1)
		}
		if index >= uint32(len(palette)) {
			return nil, fmt.Errorf("palette index %d outside length %d", index, len(palette))
		}
		s[cell] = palette[index]
	}
	return s, nil
}

// writeStorage writes s with every value passed through mapValue, using the smallest valid bit width.
func writeStorage(w *bytes.Buffer, s *storage, mapValue mapID) {
	var palette []uint32
	indices := make(map[uint32]uint32)
	cells := make([]uint32, storageCells)
	for cell, value := range s {
		mapped := mapValue(value)
		index, ok := indices[mapped]
		if !ok {
			index = uint32(len(palette))
			indices[mapped] = index
			palette = append(palette, mapped)
		}
		cells[cell] = index
	}
	bits := 0
	for _, candidate := range []int{0, 1, 2, 3, 4, 5, 6, 8, 16} {
		if len(palette) <= 1<<candidate {
			bits = candidate
			break
		}
	}
	w.WriteByte(byte(bits<<1 | 1))
	if bits != 0 {
		perWord := 32 / bits
		words := make([]uint32, (storageCells+perWord-1)/perWord)
		for cell, index := range cells {
			words[cell/perWord] |= index << (uint(cell%perWord) * uint(bits))
		}
		_ = binary.Write(w, binary.LittleEndian, words)
		_ = protocol.WriteVarint32(w, int32(len(palette)))
	}
	for _, value := range palette {
		_ = protocol.WriteVarint32(w, int32(value))
	}
}

func validBits(bits int) bool {
	switch bits {
	case 0, 1, 2, 3, 4, 5, 6, 8, 16:
		return true
	}
	return false
}

// subChunk is one decoded sub-chunk.
type subChunk struct {
	y        int8
	indexed  bool // false for version 8, whose position is its order in the chunk
	storages []*storage
}

func readSubChunk(r *bytes.Reader) (subChunk, error) {
	var header [2]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return subChunk{}, err
	}
	if header[0] != 9 && header[0] != 8 {
		return subChunk{}, fmt.Errorf("unsupported sub-chunk version %d", header[0])
	}
	sub := subChunk{}
	if header[0] == 9 {
		y, err := r.ReadByte()
		if err != nil {
			return subChunk{}, err
		}
		sub.y, sub.indexed = int8(y), true
	}
	for range header[1] {
		s, err := readStorage(r, nil)
		if err != nil {
			return subChunk{}, err
		}
		sub.storages = append(sub.storages, s)
	}
	return sub, nil
}

func writeSubChunk(w *bytes.Buffer, sub subChunk, version byte, blocks mapID) {
	w.WriteByte(version)
	w.WriteByte(byte(len(sub.storages)))
	if version == 9 {
		w.WriteByte(byte(sub.y))
	}
	for _, s := range sub.storages {
		writeStorage(w, s, blocks)
	}
}

// dimensionRange returns the lowest sub-chunk index and the biome section count of a vanilla dimension.
func dimensionRange(dimension int32) (minIndex int8, sections int, err error) {
	switch dimension {
	case packet.DimensionOverworld:
		return -4, 24, nil
	case packet.DimensionNether:
		return 0, 8, nil
	case packet.DimensionEnd:
		return 0, 16, nil
	}
	return 0, 0, fmt.Errorf("unknown chunk dimension %d", dimension)
}

// chunk18 maps the palettes of a LevelChunk payload, keeping the version-9 layout and 3D biomes. Biome
// storages are always written in full: 1.18 clients do not understand the reuse marker.
func chunk18(data []byte, count uint32, sections int, blocks, biomes mapID) ([]byte, error) {
	r, w := bytes.NewReader(data), new(bytes.Buffer)
	for range count {
		sub, err := readSubChunk(r)
		if err != nil {
			return nil, err
		}
		writeSubChunk(w, sub, 9, blocks)
	}
	var previous *storage
	for i := range sections {
		s, err := readStorage(r, previous)
		if err != nil {
			return nil, fmt.Errorf("biome section %d: %w", i, err)
		}
		writeStorage(w, s, biomes)
		previous = s
	}
	_, _ = io.Copy(w, r)
	return w.Bytes(), nil
}

// subChunk18 maps one sub-chunk of a SubChunk packet and keeps its trailing block entity NBT.
func subChunk18(data []byte, blocks mapID) ([]byte, error) {
	r, w := bytes.NewReader(data), new(bytes.Buffer)
	sub, err := readSubChunk(r)
	if err != nil {
		return nil, err
	}
	writeSubChunk(w, sub, 9, blocks)
	_, _ = io.Copy(w, r)
	return w.Bytes(), nil
}

// chunk16 rebuilds a LevelChunk payload for 1.16: version-8 sub-chunks for Y 0-255 only (sub-chunk index 0
// first, as 1.16 has no index byte), a 256-byte 2D biome map, then the border blocks and block entities.
// It returns the new payload and sub-chunk count.
func chunk16(data []byte, count uint32, dimension int32, blocks, biomes mapID, air uint32) ([]byte, uint32, error) {
	minIndex, sections, err := dimensionRange(dimension)
	if err != nil {
		return nil, 0, err
	}
	r := bytes.NewReader(data)
	byIndex := make(map[int8]subChunk, count)
	highest := int8(-1)
	for i := range count {
		sub, err := readSubChunk(r)
		if err != nil {
			return nil, 0, err
		}
		if !sub.indexed {
			sub.y = minIndex + int8(i)
		}
		if sub.y >= 0 && sub.y < 16 {
			byIndex[sub.y] = sub
			highest = max(highest, sub.y)
		}
	}
	var biomeStorage *storage
	// Use the biomes at Y=64, close to the surface in all three dimensions.
	surface := int((64 - int(minIndex)*16) / 16)
	var previous *storage
	for i := range sections {
		s, err := readStorage(r, previous)
		if err != nil {
			return nil, 0, fmt.Errorf("biome section %d: %w", i, err)
		}
		if i == surface || (biomeStorage == nil && i == sections-1) {
			biomeStorage = s
		}
		previous = s
	}

	w := new(bytes.Buffer)
	emptyStorage := new(storage)
	for i := range emptyStorage {
		emptyStorage[i] = air
	}
	for y := int8(0); y <= highest; y++ {
		sub, ok := byIndex[y]
		if !ok {
			// Fill gaps with air; air is already a target runtime ID, so it is written unmapped.
			w.WriteByte(8)
			w.WriteByte(1)
			writeStorage(w, emptyStorage, func(v uint32) uint32 { return v })
			continue
		}
		writeSubChunk(w, sub, 8, blocks)
	}
	columns := make([]byte, 256)
	if biomeStorage != nil {
		for x := range 16 {
			for z := range 16 {
				id := biomes(biomeStorage[x<<8|z<<4])
				if id > math.MaxUint8 {
					id = 1 // plains
				}
				columns[z<<4|x] = byte(id)
			}
		}
	}
	w.Write(columns)
	_, _ = io.Copy(w, r)
	return w.Bytes(), uint32(highest + 1), nil
}
