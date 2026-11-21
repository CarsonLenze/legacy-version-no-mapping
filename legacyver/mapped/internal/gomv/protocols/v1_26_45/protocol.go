// Package v1_26_45 implements the outgoing Minecraft Bedrock 1.26.45
// protocol 2169 native release against the protocol-2193 native model.
package v1_26_45

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	ID      int32 = 2169
	Version       = "1.26.45"
)

var packetMarshals = map[uint32]packetMarshal{
	packet.IDBossEvent:                     marshalBossEvent,
	packet.IDCameraPresets:                 marshalCameraPresets,
	packet.IDClientBoundAttributeLayerSync: marshalClientBoundAttributeLayerSync,
	packet.IDDimensionData:                 marshalDimensionData,
	packet.IDInventoryTransaction:          marshalInventoryTransaction,
	packet.IDItemStackResponse:             marshalItemStackResponse,
	packet.IDMoveActorDelta:                marshalMoveActorDelta,
	packet.IDPlaySound:                     marshalPlaySound,
	packet.IDPlayerAuthInput:               marshalPlayerAuthInput,
	packet.IDPrimitiveShapes:               marshalPrimitiveShapes,
	packet.IDServerBoundDiagnostics:        marshalServerBoundDiagnostics,
	packet.IDSubChunk:                      marshalSubChunk,
}
