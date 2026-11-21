package legacypacket

import (
	"github.com/akmalfairuz/legacy-version/legacyver/proto"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func ClientboundUpdateSoundData(io protocol.IO, pk *packet.ClientboundUpdateSoundData) {
	io.Uint64(&pk.ServerSoundHandle)
	if proto.IsProtoGTE(io, proto.ID2168) {
		// Before 1.26.60 the update was sent seven times; the client used the last one.
		for range 7 {
			proto.MarshalSoundDataUpdate(io, &pk.Update)
		}
		return
	}
	event := "Stop"
	io.String(&event)
	if proto.IsReader(io) {
		if event != "Stop" {
			io.UnknownEnumOption(event, "sound data event")
			return
		}
		pk.Update = protocol.SoundDataUpdate{Type: protocol.SoundDataUpdateStop}
	}
}
