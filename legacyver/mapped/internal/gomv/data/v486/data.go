// Package v486 exposes the exact historical Dragonfly and gophertunnel data
// snapshots for Minecraft 1.18.1x/protocol 486.
package v486

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

var (
	//go:embed block_states.nbt
	blockStateData []byte
	//go:embed item_runtime_ids.nbt
	itemRuntimeIDData []byte
	//go:embed creative_items.nbt
	creativeItemData []byte
	//go:embed biome_definitions.nbt
	biomeDefinitionData []byte
)

// BlockState is one exact state in the historical network block registry.
type BlockState struct {
	Name       string         `nbt:"name"`
	Properties map[string]any `nbt:"states"`
	Version    int32          `nbt:"version"`
}

// CreativeItem is one entry in Dragonfly's historical creative snapshot.
type CreativeItem struct {
	Name  string `nbt:"name"`
	Meta  int16  `nbt:"meta"`
	NBT   string `nbt:"nbt"`
	Block struct {
		Name       string         `nbt:"name"`
		Properties map[string]any `nbt:"states"`
		Version    int32          `nbt:"version"`
	} `nbt:"block"`
}

// BlockStates decodes every concatenated root compound in block_states.nbt.
func BlockStates() ([]BlockState, error) {
	reader := bytes.NewReader(blockStateData)
	decoder := nbt.NewDecoder(reader)
	states := make([]BlockState, 0, 1<<14)
	for {
		var state BlockState
		if err := decoder.Decode(&state); err != nil {
			if reader.Len() == 0 {
				return states, nil
			}
			return nil, fmt.Errorf("decode block state %d: %w", len(states), err)
		}
		states = append(states, state)
	}
}

// Items decodes the exact identifier-to-runtime-ID map used by Dragonfly.
func Items() (map[string]int32, error) {
	items := map[string]int32{}
	if err := nbt.Unmarshal(itemRuntimeIDData, &items); err != nil {
		return nil, fmt.Errorf("decode item runtime IDs: %w", err)
	}
	return items, nil
}

// Creative decodes the historical ungrouped creative inventory snapshot.
func Creative() ([]CreativeItem, error) {
	var items []CreativeItem
	if err := nbt.Unmarshal(creativeItemData, &items); err != nil {
		return nil, fmt.Errorf("decode creative items: %w", err)
	}
	return items, nil
}

// BiomeDefinitions returns the exact default network-NBT biome definitions
// embedded in gophertunnel's final protocol-486 connection implementation.
func BiomeDefinitions() []byte { return bytes.Clone(biomeDefinitionData) }
