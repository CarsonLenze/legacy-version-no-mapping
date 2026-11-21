package legacypacket

import (
	"github.com/akmalfairuz/legacy-version/legacyver/proto"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// EducationSettings writes the layout used before 1.26.60, which carried CanModifyBlocks directly instead of
// inside AgentCapabilities.
func EducationSettings(io protocol.IO, pk *packet.EducationSettings) {
	io.String(&pk.CodeBuilderDefaultURI)
	io.String(&pk.CodeBuilderTitle)
	io.Bool(&pk.CanResizeCodeBuilder)
	io.Bool(&pk.DisableLegacyTitleBar)
	io.String(&pk.PostProcessFilter)
	io.String(&pk.ScreenshotBorderPath)
	capabilities, _ := pk.AgentCapabilities.Value()
	canModify := capabilities.CanModifyBlocks
	protocol.OptionalFunc(io, &canModify, io.Bool)
	if proto.IsReader(io) {
		pk.AgentCapabilities = protocol.Optional[protocol.EducationAgentCapabilities]{}
		if _, ok := canModify.Value(); ok {
			pk.AgentCapabilities = protocol.Option(protocol.EducationAgentCapabilities{CanModifyBlocks: canModify})
		}
	}
	protocol.OptionalFunc(io, &pk.OverrideURI, io.String)
	io.Bool(&pk.HasQuiz)
	protocol.OptionalMarshaler(io, &pk.ExternalLinkSettings)
}
