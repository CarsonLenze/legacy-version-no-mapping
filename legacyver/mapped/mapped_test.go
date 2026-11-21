package mapped

import (
	"bytes"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func hashOf(t *testing.T, name string, properties map[string]any) uint32 {
	t.Helper()
	h, err := networkBlockHash(name, properties)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// The block network hash of minecraft:air is a well-known constant (-604749536).
func TestAirHash(t *testing.T) {
	if got := int32(hashOf(t, "minecraft:air", map[string]any{})); got != -604749536 {
		t.Fatalf("air hash %d", got)
	}
	r, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if r.toHash(r.airID) != r.airHash || r.fromHash(r.airHash) != r.airID {
		t.Fatal("air does not round trip")
	}
	if r.fromHash(12345) != r.airID {
		t.Fatal("unknown hashes must become air")
	}
	t.Logf("%d native block states", r.BlockCount())
}

func protocols(t *testing.T) []*Protocol {
	t.Helper()
	var out []*Protocol
	for _, construct := range []func() (*Protocol, error){New486, New475, New419} {
		p, err := construct()
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

func stoneHash(t *testing.T) uint32 { return hashOf(t, "minecraft:stone", map[string]any{}) }

// encodeChunk builds a backend overworld chunk: sub-chunks at the given indices, all one block, then 24 biome
// sections (the first explicit, the rest reuse markers).
func encodeChunk(blocks map[int8]uint32, biome uint32) ([]byte, uint32) {
	w := new(bytes.Buffer)
	var count uint32
	for y := int8(-4); y < 20; y++ {
		block, ok := blocks[y]
		if !ok {
			continue
		}
		s := new(storage)
		for i := range s {
			s[i] = block
		}
		w.Write([]byte{9, 1, byte(y)})
		writeStorage(w, s, func(v uint32) uint32 { return v })
		count++
	}
	s := new(storage)
	for i := range s {
		s[i] = biome
	}
	writeStorage(w, s, func(v uint32) uint32 { return v })
	for range 23 {
		w.WriteByte(0xff)
	}
	w.WriteByte(0) // border blocks
	return w.Bytes(), count
}

func TestChunks(t *testing.T) {
	r, _ := loadRegistry()
	stone := stoneHash(t)
	payload, count := encodeChunk(map[int8]uint32{-4: stone, 0: stone, 2: stone}, 1)
	for _, p := range protocols(t) {
		pk, err := p.chunk(&packet.LevelChunk{Dimension: packet.DimensionOverworld, SubChunkCount: count, RawPayload: payload})
		if err != nil {
			t.Fatalf("%d: %v", p.ID(), err)
		}
		chunk := pk.(*packet.LevelChunk)
		wantStone, _ := p.inner.(blockMapper).MapBlockRuntimeID(r.fromHash(stone))
		rd := bytes.NewReader(chunk.RawPayload)
		if p.legacyChunks {
			// Index 0 is stone, 1 is a filled gap, 2 is stone; index -4 is below 1.16's world.
			if chunk.SubChunkCount != 3 {
				t.Fatalf("1.16 sub-chunk count %d", chunk.SubChunkCount)
			}
			for i, want := range []uint32{wantStone, p.targetAir(), wantStone} {
				sub, err := readSubChunk(rd)
				if err != nil {
					t.Fatal(err)
				}
				if sub.indexed || sub.storages[0][0] != want {
					t.Fatalf("1.16 sub-chunk %d: %+v want %d", i, sub.storages[0][0], want)
				}
			}
			biomes := make([]byte, 256)
			if _, err := rd.Read(biomes); err != nil || biomes[0] != 1 || biomes[255] != 1 {
				t.Fatalf("1.16 biomes %v %v", biomes[:4], err)
			}
		} else {
			for i := range count {
				sub, err := readSubChunk(rd)
				if err != nil {
					t.Fatal(err)
				}
				if sub.storages[0][4095] != wantStone {
					t.Fatalf("%d sub-chunk %d: %d want %d", p.ID(), i, sub.storages[0][4095], wantStone)
				}
			}
			for i := range 24 {
				header, _ := rd.ReadByte()
				if header == 0xff {
					t.Fatalf("%d biome section %d uses the reuse marker", p.ID(), i)
				}
				_ = rd.UnreadByte()
				if _, err := readStorage(rd, nil); err != nil {
					t.Fatal(err)
				}
			}
		}
		if rest, _ := rd.ReadByte(); rest != 0 || rd.Len() != 0 {
			t.Fatalf("%d: trailing data %d, %d bytes left", p.ID(), rest, rd.Len())
		}
		t.Logf("protocol %d: stone -> %d, chunk %d -> %d bytes", p.ID(), wantStone, len(payload), len(chunk.RawPayload))
	}
}

// An UpdateBlock is written in the target's runtime IDs and read back as the original hash.
func TestUpdateBlockRoundTrip(t *testing.T) {
	stone := stoneHash(t)
	for _, p := range protocols(t) {
		in := &packet.UpdateBlock{Position: protocol.BlockPos{1, 64, 2}, NewBlockRuntimeID: stone, Flags: packet.BlockUpdateNetwork}
		out := p.ConvertFromLatest(in, nil)
		if len(out) != 1 {
			t.Fatalf("%d: %d packets", p.ID(), len(out))
		}
		buf := new(bytes.Buffer)
		out[0].Marshal(p.NewWriter(buf, 0))
		decoded := p.Packets(false)[packet.IDUpdateBlock]()
		decoded.Marshal(p.NewReader(bytes.NewReader(buf.Bytes()), 0, false))
		back := p.ConvertToLatest(decoded, nil)
		got := back[0].(*packet.UpdateBlock)
		if got.NewBlockRuntimeID != stone || got.Position != in.Position {
			t.Fatalf("%d: got %+v", p.ID(), got)
		}
		if in.NewBlockRuntimeID != stone {
			t.Fatal("input packet was modified")
		}
	}
}
