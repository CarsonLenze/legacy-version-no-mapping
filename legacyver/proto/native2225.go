// Conversions between the 1.26.60 (protocol 2225) native model and the layouts older clients use. Adapted from
// shawtymarco/go-multiversion internal/packetio (MIT).
package proto

import (
	"fmt"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const ID2193 = 2193 // v1.26.50; 1.26.60 (2225) is native.

// environment2193 is the environment attribute layout used before 1.26.60.
type environment2193 struct {
	AttributeName                                string
	FromAttribute                                protocol.Optional[protocol.AttributeData]
	Attribute                                    protocol.AttributeData
	ToAttribute                                  protocol.Optional[protocol.AttributeData]
	CurrentTransitionTicks, TotalTransitionTicks uint32
	EaseType                                     int32
	LocalTransitionTicks                         uint32
	NoiseTransition                              bool
	NoiseAlignment                               protocol.NoiseAlignment
}

func environmentFromNative(x *protocol.EnvironmentAttributeData) environment2193 {
	value := environment2193{AttributeName: x.AttributeName, Attribute: x.Attribute}
	switch x.PayloadType {
	case protocol.EnvironmentAttributePayloadTypeTransition:
		value.FromAttribute, value.ToAttribute = protocol.Option(x.FromAttribute), protocol.Option(x.ToAttribute)
		value.Attribute = x.FromAttribute
		value.CurrentTransitionTicks = x.TransitionSettings.CurrentTransitionTicks
		value.TotalTransitionTicks = x.TransitionSettings.TotalTransitionTicks
		value.EaseType = x.TransitionSettings.EaseType
	case protocol.EnvironmentAttributePayloadTypeNoiseTransition:
		value.FromAttribute, value.ToAttribute = protocol.Option(x.FromAttribute), protocol.Option(x.ToAttribute)
		value.Attribute = x.FromAttribute
		value.CurrentTransitionTicks = x.NoiseTransitionSettings.CurrentTransitionTicks
		value.TotalTransitionTicks = x.NoiseTransitionSettings.TotalTransitionTicks
		value.EaseType = x.NoiseTransitionSettings.EaseType
		value.LocalTransitionTicks = x.NoiseTransitionSettings.LocalTransitionTicks
		value.NoiseTransition = true
		value.NoiseAlignment = x.NoiseTransitionSettings.NoiseAlignment
	}
	return value
}

func environmentToNative(value environment2193, x *protocol.EnvironmentAttributeData) {
	*x = protocol.EnvironmentAttributeData{AttributeName: value.AttributeName, Attribute: value.Attribute}
	from, hasFrom := value.FromAttribute.Value()
	to, hasTo := value.ToAttribute.Value()
	if !hasFrom {
		from = value.Attribute
	}
	if !hasTo {
		to = value.Attribute
	}
	if value.NoiseTransition {
		x.PayloadType = protocol.EnvironmentAttributePayloadTypeNoiseTransition
		x.FromAttribute, x.ToAttribute = from, to
		x.NoiseTransitionSettings = protocol.AttributeNoiseTransitionSettings{CurrentTransitionTicks: value.CurrentTransitionTicks, TotalTransitionTicks: value.TotalTransitionTicks, EaseType: value.EaseType, LocalTransitionTicks: value.LocalTransitionTicks, NoiseAlignment: value.NoiseAlignment}
	} else if hasFrom || hasTo {
		x.PayloadType = protocol.EnvironmentAttributePayloadTypeTransition
		x.FromAttribute, x.ToAttribute = from, to
		x.TransitionSettings = protocol.AttributeTransitionSettings{CurrentTransitionTicks: value.CurrentTransitionTicks, TotalTransitionTicks: value.TotalTransitionTicks, EaseType: value.EaseType}
	}
}

// layerNoiseFromNative recovers the layer-level noise name older clients use. Different per-attribute noise
// names cannot be represented by that layout.
func layerNoiseFromNative(layer *protocol.AttributeLayerData) protocol.Optional[string] {
	for _, value := range layer.EnvironmentAttributes {
		if value.PayloadType == protocol.EnvironmentAttributePayloadTypeNoiseTransition && value.NoiseTransitionSettings.NoiseName != "" {
			return protocol.Option(value.NoiseTransitionSettings.NoiseName)
		}
	}
	return protocol.Optional[string]{}
}

func layerNoiseToNative(layer *protocol.AttributeLayerData, name protocol.Optional[string]) {
	value, _ := name.Value()
	for i := range layer.EnvironmentAttributes {
		if layer.EnvironmentAttributes[i].PayloadType == protocol.EnvironmentAttributePayloadTypeNoiseTransition {
			layer.EnvironmentAttributes[i].NoiseTransitionSettings.NoiseName = value
		}
	}
}

// stackRequestActions2193 is the stack request action numbering used before 1.26.60.
var stackRequestActions2193 = []func() protocol.StackRequestAction{
	func() protocol.StackRequestAction { return &protocol.TakeStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.PlaceStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.SwapStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.DropStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.DestroyStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.ConsumeStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.CreateStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.LabTableCombineStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.BeaconPaymentStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.MineBlockStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.CraftRecipeStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.AutoCraftRecipeStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.CraftCreativeStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.CraftRecipeOptionalStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.CraftGrindstoneRecipeStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.CraftLoomRecipeStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.CraftNonImplementedStackRequestAction{} },
	func() protocol.StackRequestAction { return &protocol.CraftResultsDeprecatedStackRequestAction{} },
}

func stackRequestActionVariant2193(x protocol.StackRequestAction) (uint32, bool) {
	for variant, construct := range stackRequestActions2193 {
		if fmt.Sprintf("%T", construct()) == fmt.Sprintf("%T", x) {
			return uint32(variant), true
		}
	}
	return 0, false
}
