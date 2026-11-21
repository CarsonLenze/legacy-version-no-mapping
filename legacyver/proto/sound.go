package proto

import "github.com/sandertv/gophertunnel/minecraft/protocol"

func MarshalSoundDataUpdate(io protocol.IO, x *protocol.SoundDataUpdate) {
	kind := uint32(x.Type)
	io.Varuint32(&kind)
	x.Type = uint8(kind)
	switch x.Type {
	case protocol.SoundDataUpdateStop, protocol.SoundDataUpdatePause, protocol.SoundDataUpdateResume:
	case protocol.SoundDataUpdateSetVolume:
		io.Float32(&x.Volume)
	case protocol.SoundDataUpdateSetPitch:
		io.Float32(&x.Pitch)
	case protocol.SoundDataUpdateFade:
		io.Float32(&x.Duration)
		io.Float32(&x.TargetVolume)
	case protocol.SoundDataUpdateSeekTo:
		io.Float32(&x.Seconds)
	default:
		io.UnknownEnumOption(x.Type, "sound data update type")
	}
}
