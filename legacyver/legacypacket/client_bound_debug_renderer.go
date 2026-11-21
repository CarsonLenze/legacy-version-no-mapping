package legacypacket

import (
	"image/color"

	"github.com/akmalfairuz/legacy-version/legacyver/proto"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func ClientBoundDebugRenderer(io protocol.IO, pk *packet.ClientBoundDebugRenderer) {
	if proto.IsProtoGTE(io, proto.ID898) {
		typeName := "cleardebugmarkers"
		if pk.Type == packet.ClientBoundDebugRendererAddCube {
			typeName = "adddebugmarkercube"
		}
		io.String(&typeName)
		switch typeName {
		case "cleardebugmarkers":
			pk.Type = packet.ClientBoundDebugRendererClear
		case "adddebugmarkercube":
			pk.Type = packet.ClientBoundDebugRendererAddCube
		default:
			io.UnknownEnumOption(typeName, "client bound debug renderer type")
			return
		}
	} else {
		legacyType := pk.Type + 1
		io.Uint32(&legacyType)
		if legacyType < 1 || legacyType > 2 {
			io.UnknownEnumOption(legacyType, "client bound debug renderer type")
			return
		}
		pk.Type = legacyType - 1
	}
	if proto.IsProtoGTE(io, proto.ID898) {
		protocol.OptionalMarshaler(io, &pk.Data)
		return
	}
	if pk.Type != packet.ClientBoundDebugRendererAddCube {
		if proto.IsReader(io) {
			pk.Data = protocol.Optional[packet.DebugMarkerData]{}
		}
		return
	}
	// Before 1.21.130 the colour was four floats between zero and one.
	data, _ := pk.Data.Value()
	io.String(&data.Text)
	io.Vec3(&data.Position)
	channels := [4]float32{float32(data.Colour.R) / 255, float32(data.Colour.G) / 255, float32(data.Colour.B) / 255, float32(data.Colour.A) / 255}
	for i := range channels {
		io.Float32(&channels[i])
	}
	io.Uint64(&data.Duration)
	if proto.IsReader(io) {
		toByte := func(f float32) uint8 { return uint8(min(max(f, 0), 1)*255 + 0.5) }
		data.Colour = color.RGBA{R: toByte(channels[0]), G: toByte(channels[1]), B: toByte(channels[2]), A: toByte(channels[3])}
		pk.Data = protocol.Option(data)
	}
}
