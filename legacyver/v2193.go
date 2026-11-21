package legacyver

import "github.com/akmalfairuz/legacy-version/legacyver/proto"

// New2193 returns the protocol used by Minecraft 1.26.50. It shares the 1.26.50 layouts (proto.ID2192 and up);
// 1.26.60 (protocol 2225) is native.
func New2193() *Protocol {
	return &Protocol{
		ver: "1.26.50",
		id:  proto.ID2193,
	}
}
