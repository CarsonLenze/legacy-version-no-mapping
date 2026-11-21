package mapped

import (
	"maps"
	"slices"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Block ID conversion between network hashes and dense runtime IDs, for every packet that carries a block ID
// outside chunk payloads. Adapted from shawtymarco/go-multiversion dragonfly/native_protocol.go (MIT). Packets
// are copied before they are changed, since the proxy may send the same packet to other clients.
type blockConversion struct {
	id func(uint32) uint32
}

func (m blockConversion) stack(stack protocol.ItemStack) protocol.ItemStack {
	if stack.NetworkID != 0 && stack.BlockRuntimeID != 0 {
		stack.BlockRuntimeID = int32(m.id(uint32(stack.BlockRuntimeID)))
	}
	return stack
}

func (m blockConversion) instance(it protocol.ItemInstance) protocol.ItemInstance {
	it.Stack = m.stack(it.Stack)
	return it
}

func (m blockConversion) instances(items []protocol.ItemInstance) []protocol.ItemInstance {
	cloned := slices.Clone(items)
	for i := range cloned {
		cloned[i] = m.instance(cloned[i])
	}
	return cloned
}

func (m blockConversion) stacks(items []protocol.ItemStack) []protocol.ItemStack {
	cloned := slices.Clone(items)
	for i := range cloned {
		cloned[i] = m.stack(cloned[i])
	}
	return cloned
}

func (m blockConversion) changes(entries []protocol.BlockChangeEntry) []protocol.BlockChangeEntry {
	cloned := slices.Clone(entries)
	for i := range cloned {
		cloned[i].BlockRuntimeID = m.id(cloned[i].BlockRuntimeID)
	}
	return cloned
}

func (m blockConversion) actions(actions []protocol.InventoryAction) []protocol.InventoryAction {
	cloned := slices.Clone(actions)
	for i := range cloned {
		cloned[i].OldItem = m.instance(cloned[i].OldItem)
		cloned[i].NewItem = m.instance(cloned[i].NewItem)
	}
	return cloned
}

func (m blockConversion) transaction(data protocol.InventoryTransactionData) protocol.InventoryTransactionData {
	switch source := data.(type) {
	case *protocol.UseItemTransactionData:
		cloned := *source
		cloned.Actions = m.actions(source.Actions)
		cloned.HeldItem = m.instance(source.HeldItem)
		if source.BlockRuntimeID != 0 || source.ActionType != protocol.UseItemActionClickAir {
			cloned.BlockRuntimeID = m.id(source.BlockRuntimeID)
		}
		return &cloned
	case *protocol.UseItemOnEntityTransactionData:
		cloned := *source
		cloned.HeldItem = m.instance(source.HeldItem)
		return &cloned
	case *protocol.ReleaseItemTransactionData:
		cloned := *source
		cloned.HeldItem = m.instance(source.HeldItem)
		return &cloned
	}
	return data
}

func (m blockConversion) packet(pk packet.Packet) packet.Packet {
	switch source := pk.(type) {
	case *packet.UpdateBlock:
		cloned := *source
		cloned.NewBlockRuntimeID = m.id(source.NewBlockRuntimeID)
		return &cloned
	case *packet.UpdateBlockSynced:
		cloned := *source
		cloned.NewBlockRuntimeID = m.id(source.NewBlockRuntimeID)
		return &cloned
	case *packet.UpdateSubChunkBlocks:
		cloned := *source
		cloned.Blocks = m.changes(source.Blocks)
		cloned.Extra = m.changes(source.Extra)
		return &cloned
	case *packet.InventoryContent:
		cloned := *source
		cloned.Content = m.instances(source.Content)
		cloned.StorageItem = m.instance(source.StorageItem)
		return &cloned
	case *packet.InventorySlot:
		cloned := *source
		cloned.NewItem = m.instance(source.NewItem)
		if storage, ok := source.StorageItem.Value(); ok {
			cloned.StorageItem = protocol.Option(m.instance(storage))
		}
		return &cloned
	case *packet.MobEquipment:
		cloned := *source
		cloned.NewItem = m.instance(source.NewItem)
		return &cloned
	case *packet.MobArmourEquipment:
		cloned := *source
		cloned.Helmet, cloned.Chestplate, cloned.Leggings, cloned.Boots, cloned.Body = m.instance(source.Helmet), m.instance(source.Chestplate), m.instance(source.Leggings), m.instance(source.Boots), m.instance(source.Body)
		return &cloned
	case *packet.AddItemActor:
		cloned := *source
		cloned.Item = m.instance(source.Item)
		return &cloned
	case *packet.AddPlayer:
		cloned := *source
		cloned.HeldItem = m.instance(source.HeldItem)
		return &cloned
	case *packet.AddActor:
		if source.EntityType != "minecraft:falling_block" {
			return pk
		}
		cloned := *source
		if id, ok := source.EntityMetadata[protocol.EntityDataKeyVariant].(int32); ok {
			cloned.EntityMetadata = maps.Clone(source.EntityMetadata)
			cloned.EntityMetadata[protocol.EntityDataKeyVariant] = int32(m.id(uint32(id)))
		}
		return &cloned
	case *packet.CreativeContent:
		cloned := *source
		cloned.Groups = slices.Clone(source.Groups)
		cloned.Items = slices.Clone(source.Items)
		for i := range cloned.Groups {
			cloned.Groups[i].Icon = m.stack(cloned.Groups[i].Icon)
		}
		for i := range cloned.Items {
			cloned.Items[i].Item = m.stack(cloned.Items[i].Item)
		}
		return &cloned
	case *packet.CraftingData:
		cloned := *source
		cloned.ShapedRecipes = slices.Clone(source.ShapedRecipes)
		for i := range cloned.ShapedRecipes {
			cloned.ShapedRecipes[i].Output = m.stacks(cloned.ShapedRecipes[i].Output)
		}
		cloned.ShapelessRecipes = slices.Clone(source.ShapelessRecipes)
		for i := range cloned.ShapelessRecipes {
			cloned.ShapelessRecipes[i].Output = m.stacks(cloned.ShapelessRecipes[i].Output)
		}
		cloned.UserDataShapelessRecipes = slices.Clone(source.UserDataShapelessRecipes)
		for i := range cloned.UserDataShapelessRecipes {
			cloned.UserDataShapelessRecipes[i].Output = m.stacks(cloned.UserDataShapelessRecipes[i].Output)
		}
		cloned.ShapedChemistryRecipes = slices.Clone(source.ShapedChemistryRecipes)
		for i := range cloned.ShapedChemistryRecipes {
			cloned.ShapedChemistryRecipes[i].Output = m.stacks(cloned.ShapedChemistryRecipes[i].Output)
		}
		cloned.ShapelessChemistryRecipes = slices.Clone(source.ShapelessChemistryRecipes)
		for i := range cloned.ShapelessChemistryRecipes {
			cloned.ShapelessChemistryRecipes[i].Output = m.stacks(cloned.ShapelessChemistryRecipes[i].Output)
		}
		cloned.SmithingTransformRecipes = slices.Clone(source.SmithingTransformRecipes)
		for i := range cloned.SmithingTransformRecipes {
			cloned.SmithingTransformRecipes[i].Result = m.stack(cloned.SmithingTransformRecipes[i].Result)
		}
		return &cloned
	case *packet.InventoryTransaction:
		cloned := *source
		cloned.Actions = m.actions(source.Actions)
		cloned.TransactionData = m.transaction(source.TransactionData)
		return &cloned
	case *packet.PlayerAuthInput:
		data, ok := source.ItemInteractionData.Value()
		if !ok {
			return pk
		}
		cloned := *source
		cloned.ItemInteractionData = protocol.Option(*m.transaction(&data).(*protocol.UseItemTransactionData))
		return &cloned
	case *packet.LevelSoundEvent:
		switch source.SoundType {
		case packet.SoundEventPlace, packet.SoundEventHit, packet.SoundEventItemUseOn, packet.SoundEventBreak,
			packet.SoundEventDoorOpen, packet.SoundEventDoorClose, packet.SoundEventTrapdoorOpen,
			packet.SoundEventTrapdoorClose, packet.SoundEventFenceGateOpen, packet.SoundEventFenceGateClose:
			if source.ExtraData == -1 {
				return pk
			}
			cloned := *source
			cloned.ExtraData = int32(m.id(uint32(source.ExtraData)))
			return &cloned
		}
	case *packet.LevelEvent:
		switch source.EventType {
		case packet.LevelEventParticlesDestroyBlock, packet.LevelEventParticlesDestroyBlockNoSound, packet.LevelEventParticleLegacyEvent | 21,
			packet.LevelEventParticlesCrackBlockDown, packet.LevelEventParticlesCrackBlockUp, packet.LevelEventParticlesCrackBlockNorth,
			packet.LevelEventParticlesCrackBlockSouth, packet.LevelEventParticlesCrackBlockWest, packet.LevelEventParticlesCrackBlockEast:
			cloned := *source
			cloned.EventData = int32(m.id(uint32(source.EventData)))
			return &cloned
		}
	}
	return pk
}
