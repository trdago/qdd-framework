<template>
  <div class="node-container">
    <div class="node-item" :class="{ 'node-certified': node.certified, 'node-debt': !node.certified }" @click.stop="toggleExpand">
      <span class="icon" v-if="node.children && node.children.length > 0">
        {{ expanded ? '▼' : '▶' }}
      </span>
      <span class="icon" v-if="!node.children || node.children.length === 0">
        {{ node.certified ? '✅' : '❌' }}
      </span>
      
      <div class="node-content">
        <div class="node-title">
          <span class="node-type">{{ node.type }}:</span>
          <span class="node-name">{{ node.name }}</span>
        </div>
        <div class="node-path" v-if="node.path">{{ node.path }}</div>
        
        <div class="node-tags" v-if="!node.certified && node.missing_certs && node.missing_certs.length > 0">
          <span class="tag debt" v-for="cert in node.missing_certs" :key="cert">Falta: {{ cert }}</span>
        </div>
      </div>
    </div>
    
    <div class="node-children" v-if="expanded && node.children && node.children.length > 0">
      <TopologyNode v-for="child in node.children" :key="child.id" :node="child" />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const props = defineProps({
  node: {
    type: Object,
    required: true
  }
})

const expanded = ref(true)

function toggleExpand() {
  if (props.node.children && props.node.children.length > 0) {
    expanded.value = !expanded.value
  }
}
</script>

<style scoped>
.node-container {
  margin-left: 20px;
  position: relative;
}
.node-container::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: -10px;
  width: 1px;
  background: var(--border);
  opacity: 0.3;
}
.node-item {
  display: flex;
  align-items: flex-start;
  padding: 12px;
  margin: 8px 0;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.05);
  cursor: pointer;
  transition: all 0.2s ease;
}
.node-item:hover {
  background: rgba(255, 255, 255, 0.05);
}
.node-certified {
  border-left: 4px solid var(--accent-success);
}
.node-debt {
  border-left: 4px solid var(--accent-danger);
  background: rgba(255, 71, 87, 0.05);
}
.node-content {
  margin-left: 10px;
  flex: 1;
}
.node-type {
  font-size: 0.75rem;
  text-transform: uppercase;
  color: var(--text-muted);
  letter-spacing: 1px;
  margin-right: 8px;
}
.node-name {
  font-weight: 600;
  color: var(--text-primary);
}
.node-path {
  font-size: 0.8rem;
  color: var(--text-secondary);
  font-family: monospace;
  margin-top: 4px;
}
.node-tags {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}
.tag {
  font-size: 0.7rem;
  padding: 2px 8px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.1);
}
.tag.debt {
  background: rgba(255, 71, 87, 0.2);
  color: #ff6b81;
  border: 1px solid rgba(255, 71, 87, 0.4);
}
.icon {
  margin-top: 2px;
  font-size: 0.9rem;
  color: var(--text-muted);
}
</style>
