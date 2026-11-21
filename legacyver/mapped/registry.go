// Package mapped supports the client versions that predate block network hashes (1.16.100, 1.18.0-1.18.2 and
// 1.18.10-1.18.12) using shawtymarco/go-multiversion's adapters. Those versions need full block, item and chunk
// mapping; every newer version is handled by the parent package without any.
//
// The backend (and the 1.26.60 client) identify blocks by network hash, while the adapters work on dense
// runtime IDs. This package numbers the vanilla 1.26.60 block states, converts hashes to and from those
// numbers around the adapters, and rewrites chunk palettes into the target version's runtime IDs.
package mapped

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// blockStateData is the vanilla 1.26.60 block state list (from go-multiversion data/native2225).
//
//go:embed data/block_states_1_26_60.nbt
var blockStateData []byte

type blockState struct {
	Name       string         `nbt:"name"`
	Properties map[string]any `nbt:"states"`
	Version    int32          `nbt:"version"`
}

func blockStates() ([]blockState, error) {
	buf := bytes.NewBuffer(blockStateData)
	decoder := nbt.NewDecoder(buf)
	var states []blockState
	for buf.Len() > 0 {
		var state blockState
		if err := decoder.Decode(&state); err != nil {
			return nil, fmt.Errorf("block %d: %w", len(states), err)
		}
		states = append(states, state)
	}
	return states, nil
}

// nativeRegistry numbers the vanilla 1.26.60 block states and maps their network hashes to those numbers.
type nativeRegistry struct {
	states  []blockState
	hashes  []uint32
	byHash  map[uint32]uint32
	airID   uint32
	airHash uint32
}

func (r *nativeRegistry) BlockCount() int      { return len(r.states) }
func (r *nativeRegistry) AirRuntimeID() uint32 { return r.airID }
func (r *nativeRegistry) RuntimeIDToState(id uint32) (string, map[string]any, bool) {
	if id >= uint32(len(r.states)) {
		return "", nil, false
	}
	return r.states[id].Name, r.states[id].Properties, true
}

// fromHash returns the dense ID of a block hash. Unknown hashes, such as custom blocks, become air.
func (r *nativeRegistry) fromHash(hash uint32) uint32 {
	if id, ok := r.byHash[hash]; ok {
		return id
	}
	return r.airID
}

// toHash returns the network hash of a dense ID.
func (r *nativeRegistry) toHash(id uint32) uint32 {
	if id < uint32(len(r.hashes)) {
		return r.hashes[id]
	}
	return r.airHash
}

var (
	registryOnce sync.Once
	registry     *nativeRegistry
	registryErr  error
)

func loadRegistry() (*nativeRegistry, error) {
	registryOnce.Do(func() {
		states, err := blockStates()
		if err != nil {
			registryErr = fmt.Errorf("load 1.26.60 block states: %w", err)
			return
		}
		r := &nativeRegistry{states: states, hashes: make([]uint32, len(states)), byHash: make(map[uint32]uint32, len(states))}
		foundAir := false
		for i, state := range states {
			hash, err := networkBlockHash(state.Name, state.Properties)
			if err != nil {
				registryErr = err
				return
			}
			if previous, exists := r.byHash[hash]; exists {
				registryErr = fmt.Errorf("block hash collision between states %d and %d", previous, i)
				return
			}
			r.hashes[i], r.byHash[hash] = hash, uint32(i)
			if state.Name == "minecraft:air" {
				r.airID, r.airHash, foundAir = uint32(i), hash, true
			}
		}
		if !foundAir {
			registryErr = fmt.Errorf("1.26.60 block states have no minecraft:air")
			return
		}
		registry = r
	})
	return registry, registryErr
}

// networkBlockHash returns the FNV-1a hash of the block state's little-endian NBT ({name, states}), which is
// the block network ID used when StartGame.UseBlockNetworkIDHashes is set.
func networkBlockHash(name string, properties map[string]any) (uint32, error) {
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	data := []byte{10, 0, 0}
	writeString := func(value string) {
		data = binary.LittleEndian.AppendUint16(data, uint16(len(value)))
		data = append(data, value...)
	}
	data = append(data, 8)
	writeString("name")
	writeString(name)
	data = append(data, 10)
	writeString("states")
	for _, key := range keys {
		switch value := properties[key].(type) {
		case string:
			data = append(data, 8)
			writeString(key)
			writeString(value)
		case uint8:
			data = append(data, 1)
			writeString(key)
			data = append(data, value)
		case int8:
			data = append(data, 1)
			writeString(key)
			data = append(data, byte(value))
		case bool:
			data = append(data, 1)
			writeString(key)
			if value {
				data = append(data, 1)
			} else {
				data = append(data, 0)
			}
		case int32:
			data = append(data, 3)
			writeString(key)
			data = binary.LittleEndian.AppendUint32(data, uint32(value))
		default:
			return 0, fmt.Errorf("unsupported block state type %T for %s %s", value, name, key)
		}
	}
	data = append(data, 0, 0)
	hash := fnv.New32a()
	_, _ = hash.Write(data)
	return hash.Sum32(), nil
}
