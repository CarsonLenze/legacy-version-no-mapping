package mapped

import (
	"fmt"
	"slices"

	"github.com/akmalfairuz/legacy-version/legacyver/mapped/internal/gomv/mapping"
	"github.com/akmalfairuz/legacy-version/legacyver/mapped/internal/gomv/protocols/v1_16_100"
	"github.com/akmalfairuz/legacy-version/legacyver/mapped/internal/gomv/protocols/v1_18_0"
	"github.com/akmalfairuz/legacy-version/legacyver/mapped/internal/gomv/protocols/v1_18_10"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Protocol adapts a go-multiversion adapter to a backend that uses block network hashes.
type Protocol struct {
	inner    minecraft.Protocol
	registry *nativeRegistry
	// legacyChunks is set for 1.16, which needs version-8 sub-chunks and 2D biomes.
	legacyChunks bool
}

type blockMapper interface {
	MapBlockRuntimeID(uint32) (uint32, bool)
}

type biomeMapper interface {
	MapBiomeRuntimeID(uint32) (uint32, bool)
}

type chunkRadiusLimiter interface {
	NetworkChunkRadiusLimit() int
}

// New419 returns Minecraft 1.16.100 (protocol 419).
func New419() (*Protocol, error) { return newProtocol(v1_16_100.NewWithRegistries, true) }

// New475 returns Minecraft 1.18.0-1.18.2 (protocol 475).
func New475() (*Protocol, error) { return newProtocol(v1_18_0.NewWithRegistries, false) }

// New486 returns Minecraft 1.18.10-1.18.12 (protocol 486).
func New486() (*Protocol, error) { return newProtocol(v1_18_10.NewWithRegistries, false) }

// All returns 1.18.10, 1.18.0 and 1.16.100. These clients connect over RakNet v10 and send Login before
// RequestNetworkSettings, which the listener's transport must support; Unwrap exposes the adapter's transport
// hints (LegacyNetworkSettings, PreSpawnPackets) for that.
func All() ([]minecraft.Protocol, error) {
	var protocols []minecraft.Protocol
	for _, construct := range []func() (*Protocol, error){New486, New475, New419} {
		p, err := construct()
		if err != nil {
			return nil, err
		}
		protocols = append(protocols, p)
	}
	return protocols, nil
}

func newProtocol(construct func(mapping.BlockRegistry, []protocol.ItemEntry) (minecraft.Protocol, error), legacyChunks bool) (*Protocol, error) {
	registry, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	// Items come from the backend's item registry at runtime (conn.GameData().Items).
	inner, err := construct(registry, nil)
	if err != nil {
		return nil, err
	}
	if _, ok := inner.(blockMapper); !ok {
		return nil, fmt.Errorf("protocol %d adapter has no block mapping", inner.ID())
	}
	return &Protocol{inner: inner, registry: registry, legacyChunks: legacyChunks}, nil
}

// Unwrap returns the go-multiversion adapter.
func (p *Protocol) Unwrap() minecraft.Protocol { return p.inner }

func (p *Protocol) ID() int32                         { return p.inner.ID() }
func (p *Protocol) Ver() string                       { return p.inner.Ver() }
func (p *Protocol) Packets(listener bool) packet.Pool { return p.inner.Packets(listener) }
func (p *Protocol) NewReader(r minecraft.ByteReader, shieldID int32, enableLimits bool) protocol.IO {
	return p.inner.NewReader(r, shieldID, enableLimits)
}
func (p *Protocol) NewWriter(w minecraft.ByteWriter, shieldID int32) protocol.IO {
	return p.inner.NewWriter(w, shieldID)
}

// ConvertToLatest converts a client packet and turns the adapter's dense block IDs back into hashes.
func (p *Protocol) ConvertToLatest(pk packet.Packet, conn *minecraft.Conn) []packet.Packet {
	converted := p.inner.ConvertToLatest(pk, conn)
	toHash := blockConversion{id: p.registry.toHash}
	for i, current := range converted {
		converted[i] = toHash.packet(current)
	}
	return converted
}

// ConvertFromLatest turns block hashes into dense IDs, rewrites chunk payloads into the target's runtime IDs,
// then lets the adapter convert the packet.
func (p *Protocol) ConvertFromLatest(pk packet.Packet, conn *minecraft.Conn) []packet.Packet {
	switch source := pk.(type) {
	case *packet.LevelChunk, *packet.SubChunk:
		converted, err := p.chunk(pk)
		if err != nil {
			// A chunk that cannot be converted would show the client garbage; leave it out.
			return nil
		}
		pk = converted
	case *packet.ChunkRadiusUpdated:
		if limiter, ok := p.inner.(chunkRadiusLimiter); ok && int(source.ChunkRadius) > limiter.NetworkChunkRadiusLimit() {
			cloned := *source
			cloned.ChunkRadius = int32(limiter.NetworkChunkRadiusLimit())
			pk = &cloned
		}
	default:
		pk = blockConversion{id: p.registry.fromHash}.packet(pk)
	}
	return p.inner.ConvertFromLatest(pk, conn)
}

// targetBlock maps a block hash straight to the target's runtime ID.
func (p *Protocol) targetBlock(hash uint32) uint32 {
	mapper := p.inner.(blockMapper)
	if target, ok := mapper.MapBlockRuntimeID(p.registry.fromHash(hash)); ok {
		return target
	}
	target, _ := mapper.MapBlockRuntimeID(p.registry.airID)
	return target
}

func (p *Protocol) targetAir() uint32 {
	target, _ := p.inner.(blockMapper).MapBlockRuntimeID(p.registry.airID)
	return target
}

func (p *Protocol) targetBiome(id uint32) uint32 {
	if mapper, ok := p.inner.(biomeMapper); ok {
		if target, ok := mapper.MapBiomeRuntimeID(id); ok {
			return target
		}
		return 1 // plains
	}
	return id
}

func (p *Protocol) chunk(pk packet.Packet) (packet.Packet, error) {
	switch source := pk.(type) {
	case *packet.LevelChunk:
		if source.CacheEnabled || len(source.BlobHashes) != 0 {
			return nil, fmt.Errorf("cached chunks are not supported")
		}
		if _, ok := source.SubChunkLimit.Value(); ok || source.SubChunkCount >= 0xfffffffe {
			return nil, fmt.Errorf("sub-chunk request mode is not supported")
		}
		cloned := *source
		if p.legacyChunks {
			payload, count, err := chunk16(source.RawPayload, source.SubChunkCount, source.Dimension, p.targetBlock, p.targetBiome, p.targetAir())
			if err != nil {
				return nil, err
			}
			cloned.RawPayload, cloned.SubChunkCount = payload, count
			return &cloned, nil
		}
		_, sections, err := dimensionRange(source.Dimension)
		if err != nil {
			return nil, err
		}
		payload, err := chunk18(source.RawPayload, source.SubChunkCount, sections, p.targetBlock, p.targetBiome)
		if err != nil {
			return nil, err
		}
		cloned.RawPayload = payload
		return &cloned, nil
	case *packet.SubChunk:
		if p.legacyChunks {
			return nil, fmt.Errorf("1.16 has no sub-chunk requests")
		}
		if source.CacheEnabled {
			return nil, fmt.Errorf("cached sub-chunks are not supported")
		}
		cloned := *source
		cloned.SubChunkEntries = slices.Clone(source.SubChunkEntries)
		for i, entry := range cloned.SubChunkEntries {
			payload, ok := entry.RawPayload.Value()
			if entry.Result != protocol.SubChunkResultSuccess || !ok {
				continue
			}
			mapped, err := subChunk18(payload, p.targetBlock)
			if err != nil {
				return nil, err
			}
			cloned.SubChunkEntries[i].RawPayload = protocol.Option(mapped)
		}
		return &cloned, nil
	}
	return pk, nil
}
