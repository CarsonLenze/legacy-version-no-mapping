package v1_26_20

import (
	"fmt"
	"strings"
	"sync"

	"github.com/akmalfairuz/legacy-version/legacyver/mapped/internal/gomv/mapping"
	"github.com/df-mc/worldupgrader/itemupgrader"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type runtimeData struct {
	blocks      *mapping.BlockMapper
	targetItems map[string]mapping.TargetItem
	creative    struct{ Groups, Items []any }

	itemsMu sync.RWMutex
	items   *mapping.ItemMapper
}

func (data *runtimeData) itemMapper(native []protocol.ItemEntry) (*mapping.ItemMapper, error) {
	if data == nil {
		return nil, fmt.Errorf("protocol 975 runtime data is not configured")
	}
	data.itemsMu.RLock()
	items := data.items
	data.itemsMu.RUnlock()
	if items != nil {
		return items, items.ValidateNativeEntries(native)
	}
	if len(native) == 0 {
		return nil, fmt.Errorf("native item registry is empty")
	}
	created, err := mapping.NewItemMapperWithResolver(native, data.targetItems, resolveTargetItemName)
	if err != nil {
		return nil, err
	}
	data.itemsMu.Lock()
	if data.items == nil {
		data.items = created
	}
	items = data.items
	data.itemsMu.Unlock()
	return items, nil
}

func resolveTargetItemName(name string) string {
	resolved := itemupgrader.Upgrade(itemupgrader.ItemMeta{Name: name}).Name
	if strings.HasPrefix(resolved, "minecraft:item.") {
		resolved = "minecraft:" + strings.TrimPrefix(resolved, "minecraft:item.")
	}
	if resolved == "minecraft:reeds" {
		resolved = "minecraft:sugar_cane"
	}
	return resolved
}

func (data *runtimeData) currentItemMapper() *mapping.ItemMapper {
	if data == nil {
		return nil
	}
	data.itemsMu.RLock()
	defer data.itemsMu.RUnlock()
	return data.items
}
