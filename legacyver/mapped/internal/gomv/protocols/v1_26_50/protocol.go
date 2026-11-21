// Package v1_26_50 freezes the outgoing native protocol 2193 wire against the
// stable 1.26.60 native wire model (2225), with direct registry mapping.
package v1_26_50

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	ID      int32 = 2193
	Version       = "1.26.50"
)

type wireMarshal func(protocol.IO, packet.Packet, bool)
type wirePacket struct {
	inner   packet.Packet
	marshal wireMarshal
}

func (p *wirePacket) ID() uint32 { return p.inner.ID() }
func (p *wirePacket) Marshal(io protocol.IO) {
	_, reading := io.(interface{ SliceLength(uint32, uint32) })
	p.marshal(io, p.inner, reading)
}

// WrapWirePacket applies only the frozen 2193 wire differences. Concrete
// historical wrappers keep their own layout and direct registry conversion.
func WrapWirePacket(pk packet.Packet) packet.Packet {
	var marshal wireMarshal
	switch pk.(type) {
	case *packet.InventoryTransaction:
		marshal = marshalInventoryTransaction
	case *packet.StartGame:
		marshal = marshalStartGame
	case *packet.AddActor:
		marshal = marshalAddActor
	case *packet.AddPlayer:
		marshal = marshalAddPlayer
	case *packet.Animate:
		marshal = marshalAnimate
	case *packet.LevelChunk:
		marshal = marshalLevelChunk
	case *packet.PlayerList:
		marshal = marshalPlayerList
	case *packet.PlayerSkin:
		marshal = marshalPlayerSkin
	case *packet.DimensionData:
		marshal = marshalDimensionData
	case *packet.EducationSettings:
		marshal = marshalEducationSettings
	case *packet.ServerBoundDiagnostics:
		marshal = marshalServerBoundDiagnostics
	case *packet.ClientboundUpdateSoundData:
		marshal = marshalClientboundUpdateSoundData
	case *packet.ClientBoundAttributeLayerSync:
		marshal = marshalClientBoundAttributeLayerSync
	}
	if marshal == nil {
		return pk
	}
	return &wirePacket{pk, marshal}
}
func UnwrapWirePacket(pk packet.Packet) packet.Packet {
	if p, ok := pk.(*wirePacket); ok {
		return p.inner
	}
	return pk
}
