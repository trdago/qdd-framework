<template>
  <div class="tree-item">
    <div
      class="tree-item-label"
      :class="{ 'is-folder': isFolder, 'is-active': isActive }"
      @click="toggle"
    >
      <span class="tree-icon">
        <template v-if="isFolder">
          <svg v-if="isOpen" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>
          <svg v-if="!isOpen" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>
        </template>
        <svg v-if="!isFolder" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="16" y1="13" x2="8" y2="13"></line><line x1="16" y1="17" x2="8" y2="17"></line><polyline points="10 9 9 9 8 9"></polyline></svg>
      </span>
      <span class="tree-name">{{ name }}</span>
    </div>
    
    <div v-show="isOpen" v-if="isFolder" class="tree-children">
      <TreeItem
        v-for="(childModel, childName) in folderChildren"
        :key="childName"
        :name="childName"
        :model="childModel"
        @select="$emit('select', $event)"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  name: String,
  model: Object,
  activeNodeId: String
})

const emit = defineEmits(['select'])

const isOpen = ref(false)

const isFolder = computed(() => {
  return !props.model._node
})

const folderChildren = computed(() => {
  if (isFolder.value) return props.model
  return {}
})

const isActive = computed(() => {
  return !isFolder.value && props.model._node && props.model._node.id === props.activeNodeId
})

function toggle() {
  if (isFolder.value) {
    isOpen.value = !isOpen.value
    return
  }
  emit('select', props.model._node)
}
</script>

<style scoped>
.tree-item {
  text-align: left;
}
.tree-item-label {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  cursor: pointer;
  border-radius: 4px;
  transition: background-color 0.2s;
  font-size: 14px;
  color: #d1d5db;
}
.tree-item-label:hover {
  background-color: rgba(255, 255, 255, 0.05);
}
.tree-item-label.is-active {
  background-color: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
}
.tree-icon {
  width: 20px;
  display: inline-flex;
  justify-content: center;
  align-items: center;
  margin-right: 4px;
  color: #9ca3af;
}
.tree-children {
  padding-left: 20px;
  border-left: 1px solid rgba(255, 255, 255, 0.1);
  margin-left: 10px;
}
</style>
