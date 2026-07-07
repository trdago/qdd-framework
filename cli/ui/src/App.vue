<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import mermaid from 'mermaid'
import MarkdownIt from 'markdown-it'
const md = new MarkdownIt()
mermaid.initialize({ startOnLoad: false, theme: 'dark' })

const state = ref(null)
const loading = ref(true)
const activeDetail = ref(null)
const activeTab = ref('overview')
// View Modes & Graph State
const knowledgeViewMode = ref('grid')
const knowledgeGraphSvg = ref('')
const topologyViewMode = ref('grid')
const topologyGraphSvg = ref('')
const topologySearchQuery = ref('')
const graphZoom = ref(1)
const graphPan = ref({ x: 0, y: 0 })
const isFullScreen = ref(false)
let isDragging = false
let startPan = { x: 0, y: 0 }

const zoomIn = () => { graphZoom.value = Math.min(graphZoom.value + 0.2, 5) }
const zoomOut = () => { graphZoom.value = Math.max(graphZoom.value - 0.2, 0.2) }
const resetZoom = () => { graphZoom.value = 1; graphPan.value = { x: 0, y: 0 } }
const toggleFullScreen = () => { isFullScreen.value = !isFullScreen.value; resetZoom(); }

const onGraphWheel = (e) => {
  e.preventDefault()
  if (e.deltaY > 0) { zoomOut(); return; }
  zoomIn()
}
const onGraphMouseDown = (e) => {
  isDragging = true
  startPan = { x: e.clientX - graphPan.value.x, y: e.clientY - graphPan.value.y }
}
const onGraphMouseMove = (e) => {
  if (!isDragging) return
  graphPan.value = { x: e.clientX - startPan.x, y: e.clientY - startPan.y }
}
const onGraphMouseUp = () => { isDragging = false }
const onGraphMouseLeave = () => { isDragging = false }
const omniInput = ref('')
const searchQuery = ref('')
const qclLoading = ref(false)
const lifecycleSvg = ref('')
const connectionStatus = ref('DISCONNECTED')
const syncingTabs = ref({
  overview: false,
  intelligence: false,
  sprints: false,
  findings: false,
  certifications: false,
  knowledge: false,
  lifecycle: false,
  topology: false,
  value: false,
  policies: false
})

const triggerSync = (tabName) => {
  syncingTabs.value[tabName] = true
  setTimeout(() => {
    syncingTabs.value[tabName] = false
  }, 2500)
}

let evtSource = null

const certSearchQuery = ref('')

const certStats = computed(() => {
  if (!state.value?.certifications) return { total: 0, pass: 0, fail: 0 }
  const certs = state.value.certifications
  return {
    total: certs.length,
    pass: certs.filter(c => c.status === 'PASS' || c.status.toLowerCase() === 'certified').length,
    fail: certs.filter(c => c.status !== 'PASS' && c.status.toLowerCase() !== 'certified').length
  }
})

const filteredCerts = computed(() => {
  if (!state.value?.certifications) return []
  let certs = state.value.certifications.map(c => {
    return { ...c, typeLabel: c.type || 'Proyecto' }
  })
  
  if (certSearchQuery.value) {
    const q = certSearchQuery.value.toLowerCase()
    certs = certs.filter(c => 
      c.id.toLowerCase().includes(q) || 
      (c.raw?.description || '').toLowerCase().includes(q) ||
      (c.raw?.title || '').toLowerCase().includes(q) ||
      (c.type || '').toLowerCase().includes(q)
    )
  }
  return certs
})

const extractKnowledgeMetadata = (k) => {
  const content = k.content || ""
  
  // Extract Title
  let title = k.path.split('/').pop()
  const titleMatch = content.match(/^#\s+(.+)$/m)
  if (titleMatch) {
    title = titleMatch[1].trim()
  }

  // Extract Snippet
  let snippet = ""
  const lines = content.split('\n')
  for (let line of lines) {
    const l = line.trim()
    if (l && !l.startsWith('#') && !l.startsWith('>') && !l.startsWith('![')) {
      snippet = l
      if (snippet.length > 120) snippet = snippet.substring(0, 117) + "..."
      break
    }
  }

  // Read Time
  const words = content.split(/\s+/).length
  const readTime = Math.max(1, Math.ceil(words / 200))
  
  // Extract Smart Tags
  const tags = []
  const c = content.toLowerCase()
  if (c.includes('security') || c.includes('owasp') || c.includes('auth')) tags.push('Security')
  if (c.includes('database') || c.includes('postgres') || c.includes('sql')) tags.push('Database')
  if (c.includes('architecture') || c.includes('adr')) tags.push('Architecture')
  if (c.includes('test') || c.includes('mock') || c.includes('jest')) tags.push('Testing')
  if (c.includes('api') || c.includes('rest') || c.includes('graphql')) tags.push('API')
  if (c.includes('frontend') || c.includes('vue') || c.includes('ui')) tags.push('Frontend')
  if (c.includes('deploy') || c.includes('ci/cd') || c.includes('docker')) tags.push('DevOps')
  if (tags.length === 0) tags.push('General')

  return { ...k, title, snippet, readTime, tags }
}

const knowledgeGroups = computed(() => {
  if (!state.value?.knowledge) return []
  
  let docs = state.value.knowledge
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    docs = docs.filter(k => {
      const meta = extractKnowledgeMetadata(k)
      return k.path.toLowerCase().includes(q) || 
             k.content.toLowerCase().includes(q) ||
             meta.tags.some(t => t.toLowerCase().includes(q))
    })
  }

  const groups = [
    { id: 'adr', name: '🏛️ Decisiones Arquitectónicas (ADRs)', items: [] },
    { id: 'guides', name: '📘 Guías y Estándares', items: [] },
    { id: 'workflows', name: '⚙️ Flujos de Trabajo', items: [] },
    { id: 'general', name: '📄 Documentación General', items: [] }
  ]

  docs.forEach(k => {
    const metaK = extractKnowledgeMetadata(k)
    const p = k.path.toLowerCase()
    if (p.includes('adr')) {
      groups[0].items.push(metaK)
      return
    }
    if (p.includes('guide') || p.includes('manual')) {
      groups[1].items.push(metaK)
      return
    }
    if (p.includes('workflow') || p.includes('flow')) {
      groups[2].items.push(metaK)
      return
    }
    groups[3].items.push(metaK)
  })

  return groups.filter(g => g.items.length > 0)
})

const findingsByPillar = computed(() => {
  if (!state.value?.findings) return {}
  const groups = {
    'Estructural': { title: 'Riesgo Estructural 🧠', items: [], passIcon: true },
    'Seguridad': { title: 'Seguridad Pragmática 🛡️', items: [], passIcon: true },
    'Estabilidad': { title: 'Estabilidad & Rendimiento ⚡', items: [], passIcon: true },
    'Certificación': { title: 'Brecha Certificación 📑', items: [], passIcon: true }
  }
  
  state.value.findings.forEach(f => {
    const p = f.pillar || 'Certificación'
    if (!groups[p]) groups[p] = { title: p, items: [], passIcon: true }
    groups[p].items.push(f)
    if (f.status.toUpperCase() !== 'RESOLVED') {
      groups[p].passIcon = false
    }
  })
  
  return groups
})

const openFindingsCount = computed(() => {
  if (!state.value?.findings) return 0
  return state.value.findings.filter(f => f.status.toUpperCase() !== 'RESOLVED').length
})


const renderLifecycle = async () => {
  if (!state.value?.knowledge) return
  const doc = state.value.knowledge.find(k => k.path === 'docs/command-reference.md')
  if (doc) {
    const match = doc.content.match(/```mermaid\n([\s\S]*?)```/)
    if (match && match[1]) {
      try {
        let mermaidSrc = match[1]
        
        // Append execution disabled style if policy says so
        if (state.value?.policies && state.value.policies.allow_execution === false) {
           mermaidSrc += "\n    classDef disabled fill:#333,stroke:#666,stroke-width:2px,color:#999,stroke-dasharray: 5 5;\n    class A,B,C,F,I disabled;"
        }
        
        const { svg } = await mermaid.render('mermaid-chart', mermaidSrc)
        lifecycleSvg.value = svg
      } catch (err) {
        console.error("Mermaid error:", err)
      }
    }
  }
}

const knowledgeGraphDefinition = computed(() => {
  if (!state.value?.graph_data?.nodes || state.value.graph_data.nodes.length === 0) {
    return 'graph TD\n  Empty[No data]'
  }
  
  let mm = `graph TD\n`
  const nodes = state.value.graph_data.nodes
  const edges = state.value.graph_data.edges

  // Map nodes
  nodes.forEach(n => {
    let cleanName = (n.name || 'Unnamed').replace(/[^\w\s-]/gi, '').substring(0, 40)
    let safeID = n.id.replace(/[^a-zA-Z0-9]/g, '_')
    let icon = ''
    let styleClass = ''
    
    switch(n.type) {
      case 'rule': icon = '🛡️'; styleClass = ':::ruleClass'; break;
      case 'feature': icon = '🧩'; styleClass = ':::featureClass'; break;
      case 'test': icon = '🧪'; styleClass = ':::testClass'; break;
      case 'finding': icon = '🐞'; styleClass = ':::findingClass'; break;
      case 'task': icon = '🏃'; styleClass = ':::taskClass'; break;
      case 'doc': icon = '📄'; styleClass = ':::docClass'; break;
      default: icon = '📄'; styleClass = ':::docClass'; break;
    }
    
    mm += `  ${safeID}["${icon} ${cleanName}"]${styleClass}\n`
  })

  // Map edges
  edges.forEach(e => {
    let sourceSafe = e.source.replace(/[^a-zA-Z0-9]/g, '_')
    let targetSafe = e.target.replace(/[^a-zA-Z0-9]/g, '_')
    // Filter relations if they clutter, but for now show all
    if (e.relation !== 'IMPORTS') {
      mm += `  ${sourceSafe} -->|${e.relation}| ${targetSafe}\n`
    }
  })

  // Classes
  mm += `\n`
  mm += `  classDef ruleClass fill:#10b98122,stroke:#10b981,color:#10b981;\n`
  mm += `  classDef featureClass fill:#3b82f622,stroke:#3b82f6,color:#3b82f6;\n`
  mm += `  classDef testClass fill:#f59e0b22,stroke:#f59e0b,color:#f59e0b;\n`
  mm += `  classDef findingClass fill:#ef444422,stroke:#ef4444,color:#ef4444;\n`
  mm += `  classDef taskClass fill:#8b5cf622,stroke:#8b5cf6,color:#8b5cf6;\n`
  mm += `  classDef docClass fill:#6b728022,stroke:#6b7280,color:#9ca3af;\n`

  console.log("GENERATED MERMAID GRAPH:\n" + mm)
  return mm
})

const flattenedTopology = computed(() => {
  if (!state.value?.topology?.application) return []
  const nodes = []
  const traverse = (node, parentPath = '') => {
    if (node) {
      nodes.push({ ...node, path: node.path || parentPath })
      if (node.children) {
        node.children.forEach(c => traverse(c, node.path || parentPath))
      }
    }
  }
  traverse(state.value.topology.application)
  
  if (topologySearchQuery.value) {
    const q = topologySearchQuery.value.toLowerCase()
    return nodes.filter(n => (n.name && n.name.toLowerCase().includes(q)) || (n.type && n.type.toLowerCase().includes(q)))
  }
  return nodes
})

const topologyGraphDefinition = computed(() => {
  if (!state.value?.topology?.application) return ''
  let mm = `mindmap\n  root((${state.value.topology.application.name}))\n`
  
  const traverseMindmap = (node, depth) => {
    if (!node.children || node.children.length === 0) return
    node.children.forEach(c => {
      const indent = ' '.repeat((depth + 1) * 2 + 2)
      let icon = c.certified ? '✅' : '❌'
      let cleanName = c.name.replace(/[^\w\s-]/gi, '').substring(0, 30)
      mm += `${indent}[${icon} ${cleanName}]\n`
      traverseMindmap(c, depth + 1)
    })
  }
  
  traverseMindmap(state.value.topology.application, 1)
  return mm
})

watch(activeTab, (newTab) => {
  if (newTab === 'lifecycle') {
    renderLifecycle()
  }
})

watch(knowledgeViewMode, async (newVal) => {
  if (newVal === 'graph') {
    try {
      const { svg } = await mermaid.render('mermaid-knowledge', knowledgeGraphDefinition.value)
      knowledgeGraphSvg.value = svg
    } catch (e) {
      console.error("Mermaid error (Mindmap):", e)
    }
  }
})

watch(topologyViewMode, async (newVal) => {
  if (newVal === 'graph') {
    try {
      const { svg } = await mermaid.render('mermaid-topology', topologyGraphDefinition.value)
      topologyGraphSvg.value = svg
    } catch (e) {
      console.error("Mermaid error (Topology Mindmap):", e)
    }
  }
})

watch(() => state.value?.certifications, (newVal, oldVal) => {
  if (oldVal) triggerSync('certifications')
}, { deep: true })

watch(() => state.value?.findings, (newVal, oldVal) => {
  if (oldVal) triggerSync('findings')
}, { deep: true })

watch(() => state.value?.sprints, (newVal, oldVal) => {
  if (oldVal) triggerSync('sprints')
}, { deep: true })

watch(() => state.value?.understanding, (newVal, oldVal) => {
  if (oldVal) triggerSync('intelligence')
}, { deep: true })

watch(() => state.value?.knowledge, (newVal, oldVal) => {
  if (oldVal) triggerSync('knowledge')
}, { deep: true })

watch(() => state.value?.score, (newVal, oldVal) => {
  if (oldVal !== undefined) triggerSync('overview')
})

watch(() => state.value?.topology, (newVal, oldVal) => {
  if (oldVal) triggerSync('topology')
}, { deep: true })

watch(() => state.value?.value_metrics, (newVal, oldVal) => {
  if (oldVal) triggerSync('overview')
}, { deep: true })

watch(() => state.value?.policies?.allow_execution, (newVal, oldVal) => {
  if (newVal !== oldVal && activeTab.value === 'lifecycle') {
    renderLifecycle()
  }
})

const executeOmni = async () => {
    if (!omniInput.value) return;
    const intent = omniInput.value;
    omniInput.value = '';
    qclLoading.value = true;
    
    try {
        const res = await fetch('/api/intent', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ input: intent })
        });
        
        if (!res.ok) {
            throw new Error(`Error ${res.status}: ${await res.text()}`);
        }
        
        const data = await res.json();
        
        activeDetail.value = {
            id: 'QCL Intent Result',
            type: 'AI Execution Plan',
            status: 'COMPLETED',
            raw: data,
        }
    } catch (err) {
        alert("Error procesando intent con QCL: " + err.message);
    } finally {
        qclLoading.value = false;
    }
}

const togglePolicy = async (key) => {
  const newPolicies = { ...state.value.policies, [key]: !state.value.policies[key] }
  // optimistically update UI
  state.value.policies = newPolicies
  
  try {
    const res = await fetch('/api/policies', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(newPolicies)
    })
    if (!res.ok) {
      alert("Error saving policies")
    }
  } catch (e) {
    alert("Error communicating with QDD Engine")
  }
}

const openDetail = (item, type) => {
  activeDetail.value = { ...item, type }
}

const renderedMarkdown = computed(() => {
  if (activeDetail.value?.type === 'Knowledge' && activeDetail.value?.content) {
    return md.render(activeDetail.value.content)
  }
  return ""
})
const closeDetail = () => {
  activeDetail.value = null
}

const initRealTime = () => {
  evtSource = new EventSource('/api/stream')
  
  evtSource.onopen = () => {
    connectionStatus.value = 'CONNECTED'
  }
  
  evtSource.onmessage = (e) => {
    state.value = JSON.parse(e.data)
    loading.value = false
    connectionStatus.value = 'CONNECTED'
    if (activeTab.value === 'lifecycle') {
      renderLifecycle()
    }
  }
  
  evtSource.onerror = (e) => {
    console.error("Error in SSE stream, trying to reconnect...", e)
    connectionStatus.value = 'DISCONNECTED'
  }
}

const handleKeydown = (e) => {
  if (e.key === 'Escape') closeDetail()
}

onMounted(() => {
  initRealTime()
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  if (evtSource) evtSource.close()
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div class="saas-layout" role="main">
    <div class="bg-orb orb-1"></div>
    <div class="bg-orb orb-2"></div>
    <div class="bg-orb orb-3"></div>
    
    <nav class="sidebar" role="navigation" aria-label="Main Sidebar">
      <div class="brand">
        <div class="brand-logo"><span class="icon" style="color:var(--accent-color)">◬</span></div>
        <div class="brand-text">
          <h1>QDD Framework</h1>
          <span class="version-badge">{{ state?.version || 'v1.1.0' }}</span>
        </div>
      </div>
      <div class="nav-section" role="tablist" aria-label="Main Tabs">
        <div class="nav-label">Governance</div>
        <a href="#" class="nav-item" role="tab" :aria-selected="activeTab==='overview'" :class="{active: activeTab==='overview'}" @click.prevent="activeTab='overview'"><span class="icon">◱</span> <span style="flex:1;">Overview</span><div v-if="syncingTabs['overview'] || state?.working_on === 'init'" class="sync-spinner" aria-hidden="true"></div></a>
        <a href="#" class="nav-item" role="tab" :aria-selected="activeTab==='cognitive'" :class="{active: activeTab==='cognitive'}" @click.prevent="activeTab='cognitive'"><span class="icon">🧠</span> <span style="flex:1;">Cognitive Core</span><div v-if="syncingTabs['intelligence'] || syncingTabs['knowledge'] || state?.working_on === 'learn' || state?.working_on === 'docs'" class="sync-spinner" aria-hidden="true"></div></a>
        <a href="#" class="nav-item" role="tab" :aria-selected="activeTab==='quality'" :class="{active: activeTab==='quality'}" @click.prevent="activeTab='quality'"><span class="icon">🛡️</span> <span style="flex:1;">Quality Gates</span><div v-if="syncingTabs['sprints'] || syncingTabs['findings'] || syncingTabs['certifications'] || state?.working_on === 'sprint' || state?.working_on === 'audit' || state?.working_on === 'certify'" class="sync-spinner" aria-hidden="true"></div></a>
        <a href="#" class="nav-item" role="tab" :aria-selected="activeTab==='map'" :class="{active: activeTab==='map'}" @click.prevent="activeTab='map'"><span class="icon">🗺️</span> <span style="flex:1;">Project Map</span><div v-if="syncingTabs['topology'] || syncingTabs['lifecycle'] || state?.working_on === 'map' || state?.working_on === 'release'" class="sync-spinner" aria-hidden="true"></div></a>
        <a href="#" class="nav-item" role="tab" :aria-selected="activeTab==='policies'" :class="{active: activeTab==='policies'}" @click.prevent="activeTab='policies'"><span class="icon">⚙️</span> <span style="flex:1;">Policy Control</span></a>
      </div>
      
      <div class="sidebar-footer">
        <div class="status-indicator" aria-live="polite">
          <div v-if="connectionStatus === 'CONNECTED'" class="pulse-dot" aria-hidden="true"></div>
          <div v-if="connectionStatus === 'DISCONNECTED'" class="pulse-dot disconnected" aria-hidden="true"></div>
          {{ connectionStatus === 'CONNECTED' ? 'Live Sync Active' : 'Desconectado (Reintentando...)' }}
        </div>
      </div>
    </nav>

    <div v-if="loading" class="loading-state" aria-busy="true" aria-live="polite">
      <div class="pulse-dot"></div> Cargando sistema cognitivo...
    </div>

    <main v-if="!loading" class="content-area" id="main-content" aria-live="polite">
      <header class="top-nav glassmorphism" role="banner" aria-label="App Header">
        <div class="breadcrumbs">
          <span style="text-transform: capitalize;">{{ state?.project_name || 'QDD' }}</span> <span class="separator">/</span> <span class="current" style="text-transform: capitalize;">{{ activeTab }}</span>
        </div>
        
        <div class="omnibar" :class="{ 'is-loading': qclLoading }">
            <input type="text" v-model="omniInput" @keyup.enter="executeOmni" :disabled="qclLoading" placeholder="QDD Intent (e.g. qdd sprint 3)..." class="omni-input" aria-label="QDD Intent Input" />
            <button @click="executeOmni" :disabled="qclLoading" class="omni-btn" aria-label="Ejecutar Intent">
                <span v-if="!qclLoading">✨</span>
                <span v-if="qclLoading" class="spinner">⏳</span>
            </button>
        </div>
        <div class="header-actions">
          <div class="audit-badge" :class="state?.audit_status.startsWith('PASS') ? 'pass' : 'fail'">
            <span class="indicator-dot"></span> {{ state?.audit_status }}
          </div>
        </div>
      </header>

      <div class="page-content">
        <!-- OVERVIEW TAB -->
        <section v-show="activeTab === 'overview'" aria-label="Overview Dashboard">
          <div class="grid-layout cols-4 mb-section">
            <div class="panel glass-panel score-panel fade-in stagger-1" role="region" aria-labelledby="score-title">
              <h3 id="score-title" class="panel-title">Quality Score</h3>
              <div class="score-row">
                <div class="score-container">
                  <div class="score-value">{{ state.score }}</div>
                  <div class="score-grade" :class="'grade-' + state.grade.charAt(0).toLowerCase()">Grade {{ state.grade }}</div>
                </div>
                <div class="qdd-ring-container">
                  <svg class="qdd-ring" viewBox="0 0 100 100">
                    <circle class="ring-bg" cx="50" cy="50" r="45"></circle>
                    <circle class="ring-progress" cx="50" cy="50" r="45" :style="{'stroke-dashoffset': 283 - (283 * state.score) / 100}"></circle>
                  </svg>
                </div>
              </div>
            </div>

            <div class="panel glass-panel fade-in stagger-2" role="region" style="grid-column: span 2;">
              <h3 class="panel-title">ROI & Value Realization</h3>
              <div class="stats-cards" style="margin-top: 1rem;">
                <div class="stat-card">
                  <span class="stat-value"><span v-if="state.usage_time">{{ state.usage_time }}</span><span v-if="!state.usage_time">&nbsp;</span></span>
                  <span class="stat-label">Tiempo de Uso QDD</span>
                </div>
                <div class="stat-card pass">
                  <span class="stat-value"><span v-if="state.value_metrics?.hours_saved !== undefined">{{ state.value_metrics.hours_saved }} hrs</span><span v-if="state.value_metrics?.hours_saved === undefined">&nbsp;</span></span>
                  <span class="stat-label">Horas Ahorradas (IA)</span>
                </div>
                <div class="stat-card">
                  <span class="stat-value"><span v-if="state.value_metrics?.debt_reduced !== undefined">{{ state.value_metrics.debt_reduced }}</span><span v-if="state.value_metrics?.debt_reduced === undefined">&nbsp;</span></span>
                  <span class="stat-label">Bugs Prevenidos</span>
                </div>
                <div class="stat-card" style="border-left: 2px solid #10b981;">
                  <span class="stat-value" style="color: #10b981;"><span v-if="state.value_metrics?.hours_saved !== undefined">${{ (state.value_metrics.hours_saved * 50).toLocaleString() }}</span><span v-if="state.value_metrics?.hours_saved === undefined">&nbsp;</span></span>
                  <span class="stat-label">ROI Estimado</span>
                </div>
              </div>
            </div>
            
            <div class="panel glass-panel fade-in stagger-3" role="region">
              <h3 id="telemetry-title" class="panel-title">System Telemetry</h3>
              <div class="telemetry-grid">
                <div class="telemetry-stat">
                  <span class="t-label">UPTIME</span>
                  <span class="t-value">{{ state.telemetry?.uptime || '0s' }}</span>
                </div>
                <div class="telemetry-stat">
                  <span class="t-label">MEM (SYS)</span>
                  <span class="t-value">{{ state.telemetry?.memory_sys || '0 MB' }}</span>
                </div>
                <div class="telemetry-stat">
                  <span class="t-label">GOROUTINES</span>
                  <span class="t-value">{{ state.telemetry?.goroutines || 0 }}</span>
                </div>
              </div>
            </div>
            
            <div class="panel glass-panel fade-in stagger-4" role="region">
               <h3 class="panel-title">Infrastructure Stack</h3>
               <div class="stack-grid" aria-label="Technology Stack" style="max-height: 100px; overflow-y: auto;">
                 <div class="stack-item" v-if="state.config?.architecture">
                   <span class="stack-icon glow-icon" style="color: #60a5fa;">🏗️</span>
                   <span class="stack-name">{{ typeof state.config.architecture === 'string' ? state.config.architecture : state.config.architecture[0] }}</span>
                 </div>
                 <div class="stack-item" v-for="lang in state.config?.languages" :key="lang">
                   <span class="stack-icon glow-icon" style="color: #f59e0b;">⚡</span>
                   <span class="stack-name">{{ lang }}</span>
                 </div>
                 <div class="stack-item" v-for="db in state.config?.databases" :key="db">
                   <span class="stack-icon glow-icon" style="color: #10b981;">🗄️</span>
                   <span class="stack-name">{{ db }}</span>
                 </div>
               </div>
            </div>
          </div>
          
          <div class="grid-layout cols-1">
             <div class="panel glass-panel" style="display: flex; flex-direction: column;">
                <h3 class="panel-title">Evolución de Uso QDD (30 Días)</h3>
                <div class="chart-container" style="flex: 1; min-height: 220px; position: relative; margin-top: 16px;">
                   <svg viewBox="0 0 100 100" preserveAspectRatio="none" style="width: 100%; height: 100%; overflow: visible; padding-bottom: 20px;">
                     <!-- Grid lines -->
                     <line x1="0" y1="20" x2="100" y2="20" stroke="rgba(255,255,255,0.05)" stroke-width="0.5" />
                     <line x1="0" y1="40" x2="100" y2="40" stroke="rgba(255,255,255,0.05)" stroke-width="0.5" />
                     <line x1="0" y1="60" x2="100" y2="60" stroke="rgba(255,255,255,0.05)" stroke-width="0.5" />
                     <line x1="0" y1="80" x2="100" y2="80" stroke="rgba(255,255,255,0.1)" stroke-width="0.5" />
                     
                     <!-- X Axis Labels -->
                     <text x="0" y="95" fill="var(--text-muted)" font-size="4">Sem 1</text>
                     <text x="33" y="95" fill="var(--text-muted)" font-size="4">Sem 2</text>
                     <text x="66" y="95" fill="var(--text-muted)" font-size="4">Sem 3</text>
                     <text x="100" y="95" fill="var(--text-muted)" font-size="4" text-anchor="end">Hoy</text>

                     <!-- Bugs (Deuda Técnica) Line - Red -->
                     <polyline points="0,50 33,60 66,40 100,70" fill="none" stroke="#ef4444" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                     
                     <!-- Sprints Line - Orange -->
                     <polyline points="0,80 33,70 66,50 100,30" fill="none" stroke="#f59e0b" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                     
                     <!-- Certifications Line - Green -->
                     <polyline points="0,75 33,65 66,45 100,20" fill="none" stroke="#10b981" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                   </svg>
                   
                   <!-- Legend -->
                   <div style="display: flex; justify-content: center; gap: 16px; position: absolute; bottom: -10px; width: 100%;">
                     <div style="display: flex; align-items: center; gap: 4px;">
                       <span style="width: 8px; height: 8px; border-radius: 50%; background: #f59e0b;"></span>
                       <span style="font-size: 11px; color: var(--text-secondary);">Sprints</span>
                     </div>
                     <div style="display: flex; align-items: center; gap: 4px;">
                       <span style="width: 8px; height: 8px; border-radius: 50%; background: #10b981;"></span>
                       <span style="font-size: 11px; color: var(--text-secondary);">Certs</span>
                     </div>
                     <div style="display: flex; align-items: center; gap: 4px;">
                       <span style="width: 8px; height: 8px; border-radius: 50%; background: #ef4444;"></span>
                       <span style="font-size: 11px; color: var(--text-secondary);">Bugs</span>
                     </div>
                   </div>
                </div>
             </div>
             
             <div class="panel glass-panel" style="display: flex; flex-direction: column; background: #000;">
                <h3 class="panel-title" style="display: flex; align-items: center; gap: 8px;">
                  <span class="pulse-dot"></span> Live MCP Log
                </h3>
                <div class="mcp-terminal" style="flex: 1; min-height: 200px; max-height: 300px; overflow-y: auto; background: #0c0c0c; border: 1px solid #222; border-radius: 8px; padding: 12px; font-family: monospace; font-size: 12px; color: #a1a1aa; line-height: 1.5;">
                   <div v-for="(log, idx) in state.mcp_logs" :key="idx" class="mcp-log-line" style="margin-bottom: 4px; word-break: break-all;">
                     <span style="color: #60a5fa;">></span> {{ log }}
                   </div>
                   <div v-if="!state.mcp_logs || state.mcp_logs.length === 0" style="color: #555;">Esperando actividad del agente...</div>
                </div>
             </div>
          </div>
        </section>

        <!-- COGNITIVE CORE TAB (Intelligence + Knowledge) -->
        <section v-show="activeTab === 'cognitive'" class="fade-in" role="region" aria-labelledby="cognitive-title">
          <div v-if="state?.understanding" class="panel glass-panel mb-section">
            <h2 id="cognitive-title" class="panel-title">🧠 Cognitive Core</h2>
            <div class="intel-summary" style="margin-bottom: 24px;">
              <p style="font-size: 16px; line-height: 1.6; color: var(--text-primary);">{{ state.understanding.summary }}</p>
            </div>
            
            <div class="stats-cards" style="grid-template-columns: repeat(3, 1fr); margin-bottom: 24px;">
                <div class="stat-card">
                  <span class="stat-value">{{ state.understanding.components?.length || 0 }}</span>
                  <span class="stat-label">Componentes</span>
                </div>
                <div class="stat-card pass">
                  <span class="stat-value">{{ state.understanding.objectives?.length || 0 }}</span>
                  <span class="stat-label">Objetivos</span>
                </div>
                <div class="stat-card">
                  <span class="stat-value">{{ state.understanding.guidelines?.length || 0 }}</span>
                  <span class="stat-label">Reglas</span>
                </div>
            </div>

            <div style="display: flex; gap: 12px; margin-bottom: 30px;">
                <button class="btn btn-outline" @click="openDetail({title: 'Componentes Core', content: state.understanding.components.map(c=>'- '+c).join('\n')}, 'Cognitive')" v-if="state.understanding.components?.length > 0">Ver Componentes</button>
                <button class="btn btn-outline" @click="openDetail({title: 'Objetivos de Negocio', content: state.understanding.objectives.map(c=>'- '+c).join('\n')}, 'Cognitive')" v-if="state.understanding.objectives?.length > 0">Ver Objetivos</button>
                <button class="btn btn-outline" @click="openDetail({title: 'Reglas y Guidelines', content: state.understanding.guidelines.map(c=>'- '+c).join('\n')}, 'Cognitive')" v-if="state.understanding.guidelines?.length > 0">Ver Reglas</button>
            </div>
          </div>
          <div v-if="!state.usage_time" class="empty-state">No hay reporte cognitivo. Ejecuta <code>qdd learn</code>.</div>

          <div class="panel glass-panel" v-if="state?.knowledge?.length > 0">
             <div class="panel-header" style="display: flex; justify-content: space-between; align-items: center; border-bottom: none; padding-bottom: 0;">
                <h3 class="panel-title" style="margin: 0;">Knowledge Base Domains</h3>
                <div style="display: flex; gap: 16px; align-items: center;">
                  <div class="segmented-control" style="display: flex; background: rgba(0,0,0,0.3); border-radius: 8px; padding: 4px; border: 1px solid var(--border-color);">
                    <button @click="knowledgeViewMode = 'grid'" :class="{ 'btn-active': knowledgeViewMode === 'grid' }" style="background: transparent; border: none; color: white; padding: 6px 12px; border-radius: 6px; cursor: pointer; font-size: 13px; transition: all 0.2s;">Grid</button>
                    <button @click="knowledgeViewMode = 'graph'" :class="{ 'btn-active': knowledgeViewMode === 'graph' }" style="background: transparent; border: none; color: white; padding: 6px 12px; border-radius: 6px; cursor: pointer; font-size: 13px; transition: all 0.2s;">Graph</button>
                  </div>
                  <input v-show="knowledgeViewMode === 'grid'" type="text" v-model="searchQuery" placeholder="🔍 Buscar documentos..." style="padding: 8px 16px; border-radius: 8px; border: 1px solid var(--border-color); background: rgba(0,0,0,0.3); color: white; font-size: 13px;">
                </div>
             </div>

             <div v-show="knowledgeViewMode === 'grid'" class="mt-4">
                <div v-for="group in knowledgeGroups" :key="group.id" style="margin-bottom: 24px;">
                   <h4 style="margin: 0 0 12px 0; font-size: 14px; color: var(--text-primary); border-bottom: 1px solid rgba(255,255,255,0.1); padding-bottom: 6px;">{{ group.name }}</h4>
                   <div class="knowledge-grid" style="display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px;">
                      <div v-for="k in group.items" :key="k.path" tabindex="0" class="card glass-panel clickable-card" role="button" @click="openDetail(k, 'Knowledge')" @keydown.enter="openDetail(k, 'Knowledge')" style="padding: 12px; transition: all 0.2s; border: 1px solid rgba(255,255,255,0.05); cursor: pointer; border-radius: 8px; display: flex; flex-direction: column; justify-content: space-between; background: rgba(0,0,0,0.2);">
                        <div>
                          <h5 style="margin: 0 0 6px 0; font-size: 14px; color: var(--accent-color);">{{ k.title }}</h5>
                          <div style="display: flex; gap: 4px; margin-bottom: 8px; flex-wrap: wrap;">
                            <span v-for="tag in k.tags" :key="tag" style="font-size: 9px; padding: 2px 6px; background: rgba(59, 130, 246, 0.1); color: #60a5fa; border-radius: 12px; border: 1px solid rgba(59, 130, 246, 0.2);">{{ tag }}</span>
                          </div>
                          <p style="margin: 0 0 8px 0; font-size: 12px; color: var(--text-secondary); line-height: 1.4;">{{ k.snippet }}</p>
                        </div>
                        <div style="display: flex; justify-content: space-between; align-items: center; border-top: 1px solid rgba(255,255,255,0.05); padding-top: 8px; margin-top: auto;">
                          <span style="font-size: 10px; color: var(--text-muted); font-family: monospace; max-width: 70%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{{ k.path }}</span>
                          <span class="audit-badge pass" style="font-size: 10px; padding: 2px 6px;">⏳ {{ k.readTime }}m</span>
                        </div>
                      </div>
                   </div>
                </div>
                <div v-if="knowledgeGroups.length === 0" style="text-align: center; color: var(--text-muted);">No se encontraron documentos.</div>
             </div>

             <div v-show="knowledgeViewMode === 'graph'" class="mt-4 graph-panel fade-in" :class="{ 'fullscreen-mode': isFullScreen }">
                 <div class="graph-toolbar">
                   <button class="tool-btn" @click="zoomIn">🔍+</button>
                   <button class="tool-btn" @click="zoomOut">🔍-</button>
                   <button class="tool-btn" @click="resetZoom">🎯</button>
                   <button class="tool-btn" @click="toggleFullScreen">⛶</button>
                 </div>
                 <div v-if="knowledgeGraphSvg" class="mermaid-viewport" @wheel="onGraphWheel" @mousedown="onGraphMouseDown" @mousemove="onGraphMouseMove" @mouseup="onGraphMouseUp" @mouseleave="onGraphMouseLeave">
                    <div class="mermaid-canvas" :style="{ transform: `translate(${graphPan.x}px, ${graphPan.y}px) scale(${graphZoom})` }" v-html="knowledgeGraphSvg"></div>
                 </div>
             </div>
          </div>
        </section>

        <!-- TOPOLOGY TAB -->
        <!-- PROJECT MAP TAB (Topology + Lifecycle) -->
        <section v-show="activeTab === 'map'" class="fade-in" role="region" aria-labelledby="map-title">
           
           <div class="panel glass-panel mb-section">
              <div class="panel-header" style="display: flex; justify-content: space-between; align-items: center; border-bottom: none; padding-bottom: 0; margin-bottom: 20px;">
                <div style="display: flex; align-items: center; gap: 16px;">
                  <h2 id="map-title" class="panel-title" style="margin: 0;">🗺️ Project Map (Architecture & DevOps)</h2>
                  <div v-if="state?.topology" class="audit-badge" :class="state.topology.global_score === 100 ? 'pass' : 'fail'" style="padding: 2px 8px; font-size: 11px;">
                    Score: {{ state.topology.global_score }}%
                  </div>
                </div>
                <div style="display: flex; gap: 16px; align-items: center;">
                    <div class="segmented-control" style="display: flex; background: rgba(0,0,0,0.3); border-radius: 8px; padding: 4px; border: 1px solid var(--border-color);">
                      <button @click="topologyViewMode = 'grid'" :class="{ 'btn-active': topologyViewMode === 'grid' }" style="background: transparent; border: none; color: white; padding: 6px 12px; border-radius: 6px; cursor: pointer; font-size: 13px; transition: all 0.2s;">Grid (Dominios)</button>
                      <button @click="topologyViewMode = 'graph'" :class="{ 'btn-active': topologyViewMode === 'graph' }" style="background: transparent; border: none; color: white; padding: 6px 12px; border-radius: 6px; cursor: pointer; font-size: 13px; transition: all 0.2s;">Graph (Relaciones)</button>
                    </div>
                    <input v-show="topologyViewMode === 'grid'" type="text" v-model="topologySearchQuery" placeholder="🔍 Buscar módulos..." style="padding: 8px 16px; border-radius: 8px; border: 1px solid var(--border-color); background: rgba(0,0,0,0.3); color: white; width: 250px; font-size: 13px;">
                </div>
              </div>

              <div class="topology-container mt-4" v-if="state?.topology?.application">
                <div v-show="topologyViewMode === 'grid'" class="knowledge-grid" style="display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px;">
                  <div v-for="n in flattenedTopology" :key="n.id || n.name" tabindex="0" class="card glass-panel clickable-card" role="button" @click="openDetail(n, 'Topology')" @keydown.enter="openDetail(n, 'Topology')" :style="{ padding: '16px', transition: 'all 0.2s', border: n.certified ? '1px solid rgba(16, 185, 129, 0.2)' : '1px solid rgba(239, 68, 68, 0.2)', cursor: 'pointer', borderRadius: '8px', display: 'flex', flexDirection: 'column', justifyContent: 'space-between', background: n.certified ? 'rgba(16, 185, 129, 0.05)' : 'rgba(239, 68, 68, 0.05)' }">
                    <div>
                      <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                        <h4 style="margin: 0 0 8px 0; font-size: 15px; color: var(--text-primary);">{{ n.name }}</h4>
                        <span class="icon">{{ n.certified ? '✅' : '❌' }}</span>
                      </div>
                      <div style="display: flex; gap: 6px; margin-bottom: 12px; flex-wrap: wrap;">
                        <span style="font-size: 10px; padding: 2px 8px; background: rgba(255, 255, 255, 0.1); border-radius: 12px; text-transform: uppercase;">{{ n.type }}</span>
                        <span v-for="cert in (n.missing_certs || [])" :key="cert" style="font-size: 10px; padding: 2px 8px; background: rgba(239, 68, 68, 0.1); color: #ef4444; border-radius: 12px; border: 1px solid rgba(239, 68, 68, 0.2);">Falta: {{ cert }}</span>
                      </div>
                    </div>
                    <div style="display: flex; justify-content: space-between; align-items: center; border-top: 1px solid rgba(255,255,255,0.05); padding-top: 10px; margin-top: auto;">
                        <span style="font-size: 11px; color: var(--text-muted); font-family: monospace;">{{ n.path || 'Root' }}</span>
                      </div>
                    </div>
                  </div>
                
                <div v-show="topologyViewMode === 'graph'" class="panel glass-panel fade-in graph-panel" :class="{ 'fullscreen-mode': isFullScreen }">
                   <div class="graph-toolbar">
                     <button class="tool-btn" @click="zoomIn" title="Acercar">🔍+</button>
                     <button class="tool-btn" @click="zoomOut" title="Alejar">🔍-</button>
                     <button class="tool-btn" @click="resetZoom" title="Centrar">🎯</button>
                     <button class="tool-btn" @click="toggleFullScreen" title="Pantalla Completa">⛶</button>
                   </div>
                   
                   <div v-if="topologyGraphSvg" class="mermaid-viewport" 
                        @wheel="onGraphWheel"
                        @mousedown="onGraphMouseDown"
                        @mousemove="onGraphMouseMove"
                        @mouseup="onGraphMouseUp"
                        @mouseleave="onGraphMouseLeave">
                      <div class="mermaid-canvas" :style="{ transform: `translate(${graphPan.x}px, ${graphPan.y}px) scale(${graphZoom})` }" v-html="topologyGraphSvg"></div>
                   </div>
                   <div v-if="!topologyGraphSvg" class="empty-state">Renderizando Topology Graph...</div>
                </div>
              </div>
              <div v-if="!state?.topology" class="empty-state">
                No topology map found. Run <code>qdd map</code> in the terminal to generate it.
              </div>
           </div>

           <!-- LIFECYCLE FLOW -->
           <div class="panel glass-panel">
               <h3 class="panel-title" style="margin-bottom: 24px; font-size: 18px; text-align: center;">Agile Continuous Lifecycle</h3>
               <div style="display: flex; align-items: center; justify-content: center; gap: 16px; flex-wrap: wrap; padding: 20px;">
                   <!-- Setup -->
                   <div class="corp-step clickable-card" @click="openDetail({title: 'Discovery', content: 'Iniciando el proyecto y escaneo de contexto. Comando: /qdd init'}, 'Lifecycle')" style="text-align: center; width: 120px; cursor: pointer; padding: 10px; border-radius: 8px;">
                      <div style="width: 50px; height: 50px; border-radius: 50%; background: rgba(96, 165, 250, 0.1); border: 2px solid #3b82f6; display: flex; align-items: center; justify-content: center; margin: 0 auto 12px; font-size: 20px;">1</div>
                      <div style="font-weight: bold; font-size: 14px;">Discovery</div>
                      <code style="font-size: 11px; color: var(--text-muted);">/qdd init</code>
                   </div>
                   <div style="color: var(--text-muted);">→</div>
                   <!-- Learn -->
                   <div class="corp-step clickable-card" @click="openDetail({title: 'Intelligence', content: 'Aprendizaje profundo del repositorio y sincronización del motor cognitivo. Comando: /qdd learn'}, 'Lifecycle')" style="text-align: center; width: 120px; cursor: pointer; padding: 10px; border-radius: 8px;">
                      <div style="width: 50px; height: 50px; border-radius: 50%; background: rgba(167, 139, 250, 0.1); border: 2px solid #a78bfa; display: flex; align-items: center; justify-content: center; margin: 0 auto 12px; font-size: 20px;">2</div>
                      <div style="font-weight: bold; font-size: 14px;">Intelligence</div>
                      <code style="font-size: 11px; color: var(--text-muted);">/qdd learn</code>
                   </div>
                   <div style="color: var(--text-muted);">→</div>
                   <!-- Circular Loop -->
                   <div style="background: rgba(0,0,0,0.2); border: 1px solid rgba(255,255,255,0.1); padding: 20px; border-radius: 16px; display: flex; align-items: center; gap: 16px;">
                      <div class="corp-step clickable-card" @click="openDetail({title: 'Audit', content: 'Auditoría continua de calidad y validación de reglas. Comando: /qdd validate'}, 'Lifecycle')" style="text-align: center; width: 120px; cursor: pointer; padding: 10px; border-radius: 8px;">
                          <div style="width: 50px; height: 50px; border-radius: 50%; background: rgba(16, 185, 129, 0.1); border: 2px solid #10b981; display: flex; align-items: center; justify-content: center; margin: 0 auto 12px; font-size: 20px;">3</div>
                          <div style="font-weight: bold; font-size: 14px;">Audit</div>
                          <code style="font-size: 11px; color: var(--text-muted);">/qdd validate</code>
                      </div>
                      <div style="color: var(--text-muted);">↻</div>
                      <div class="corp-step clickable-card" @click="openDetail({title: 'Sprint', content: 'Generación de sprints e iteraciones incrementales automatizadas. Comando: /qdd sprint'}, 'Lifecycle')" style="text-align: center; width: 120px; cursor: pointer; padding: 10px; border-radius: 8px;">
                          <div style="width: 50px; height: 50px; border-radius: 50%; background: rgba(245, 158, 11, 0.1); border: 2px solid #f59e0b; display: flex; align-items: center; justify-content: center; margin: 0 auto 12px; font-size: 20px;">4</div>
                          <div style="font-weight: bold; font-size: 14px;">Sprint</div>
                          <code style="font-size: 11px; color: var(--text-muted);">/qdd sprint</code>
                      </div>
                   </div>
                   <div style="color: var(--text-muted);">→</div>
                   <!-- Release -->
                   <div class="corp-step clickable-card" @click="openDetail({title: 'Release', content: 'Despliegue a producción y certificación de calidad. Comando: /qdd release'}, 'Lifecycle')" style="text-align: center; width: 120px; cursor: pointer; padding: 10px; border-radius: 8px;">
                      <div style="width: 50px; height: 50px; border-radius: 50%; background: rgba(236, 72, 153, 0.1); border: 2px solid #ec4899; display: flex; align-items: center; justify-content: center; margin: 0 auto 12px; font-size: 20px;">5</div>
                      <div style="font-weight: bold; font-size: 14px;">Release</div>
                      <code style="font-size: 11px; color: var(--text-muted);">/qdd release</code>
                   </div>
               </div>
           </div>
        </section>

        <!-- QUALITY GATES TAB (Sprints + Certifications + Findings) -->
        <section v-show="activeTab === 'quality'" class="fade-in" role="region" aria-labelledby="quality-title">
           <div class="panel glass-panel mb-section">
              <div class="panel-header" style="display: flex; justify-content: space-between; align-items: center;">
                 <h2 id="quality-title" class="panel-title" style="margin:0;">🛡️ Quality Gates & Pipeline</h2>
                 <div class="audit-badge" :class="certStats.fail === 0 && openFindingsCount === 0 ? 'pass' : 'fail'">
                    {{ certStats.fail === 0 && openFindingsCount === 0 ? 'HEALTHY' : 'NEEDS ATTENTION' }}
                 </div>
              </div>
              
              <div class="grid-layout cols-3 mt-4">
                 <!-- Sprints Column -->
                 <div class="card glass-panel" style="padding: 16px; border: 1px solid rgba(255,255,255,0.1); background: rgba(0,0,0,0.2);">
                    <h3 style="font-size: 16px; margin: 0 0 16px 0; border-bottom: 1px solid rgba(255,255,255,0.1); padding-bottom: 8px;">🏃 Sprints</h3>
                    <div style="font-size: 32px; font-weight: bold; margin-bottom: 16px;">{{ state.sprints?.length || 0 }} <span style="font-size: 14px; font-weight: normal; color: var(--text-muted);">Activos</span></div>
                    <button class="btn btn-outline" style="width: 100%;" @click="openDetail({title: 'Sprints Activos', content: '### Lista de Sprints\n' + (state.sprints || []).map(s => '- **' + s.id + '**: ' + s.status).join('\n')}, 'Quality')">Ver Sprints</button>
                 </div>
                 
                 <!-- Certifications Column -->
                 <div class="card glass-panel" style="padding: 16px; border: 1px solid rgba(255,255,255,0.1); background: rgba(0,0,0,0.2);">
                    <h3 style="font-size: 16px; margin: 0 0 16px 0; border-bottom: 1px solid rgba(255,255,255,0.1); padding-bottom: 8px;">📑 Certifications</h3>
                    <div style="display: flex; justify-content: space-between; margin-bottom: 16px;">
                       <div>
                         <div style="font-size: 24px; font-weight: bold; color: var(--success);">{{ certStats.pass }}</div>
                         <div style="font-size: 12px; color: var(--text-muted);">Aprobadas</div>
                       </div>
                       <div style="text-align: right;">
                         <div style="font-size: 24px; font-weight: bold; color: var(--danger);">{{ certStats.fail }}</div>
                         <div style="font-size: 12px; color: var(--text-muted);">Pendientes</div>
                       </div>
                    </div>
                    <div class="cert-list" style="max-height: 200px; overflow-y: auto; display: flex; flex-direction: column; gap: 8px;">
                       <div v-for="cert in filteredCerts" :key="cert.id" class="card glass-panel clickable-card" @click="openDetail({...cert, title: cert.id, type: 'Certification'}, 'Certification')" style="padding: 8px 12px; border-radius: 6px; cursor: pointer; display: flex; justify-content: space-between; align-items: center; border: 1px solid rgba(255,255,255,0.05); background: rgba(0,0,0,0.1);">
                          <div style="font-size: 13px; font-weight: 500;">{{ cert.id }}</div>
                          <div class="status-pill" :class="cert.status === 'PASS' ? 'resolved' : 'open'" style="font-size: 10px; padding: 2px 6px;">{{ cert.status }}</div>
                       </div>
                       <div v-if="filteredCerts.length === 0" style="color: var(--text-muted); font-size: 13px; text-align: center;">No hay certificaciones configuradas</div>
                    </div>
                 </div>
                 
                 <!-- Findings Column -->
                 <div class="card glass-panel" style="padding: 16px; border: 1px solid rgba(255,255,255,0.1); background: rgba(0,0,0,0.2);">
                    <h3 style="font-size: 16px; margin: 0 0 16px 0; border-bottom: 1px solid rgba(255,255,255,0.1); padding-bottom: 8px;">🐞 Deuda Técnica</h3>
                    <div style="font-size: 32px; font-weight: bold; color: var(--accent-color); margin-bottom: 16px;">{{ openFindingsCount }} <span style="font-size: 14px; font-weight: normal; color: var(--text-muted);">Abiertos</span></div>
                    <button class="btn btn-outline" style="width: 100%;" @click="openDetail({title: 'Radar de Bugs', content: '### Hallazgos Abiertos\n' + Object.values(findingsByPillar).flatMap(g => g.items.filter(i=>i.status.toUpperCase()!=='RESOLVED')).map(f => '- **' + f.id + '**: ' + f.status).join('\n')}, 'Quality')">Analizar Bugs</button>
                 </div>
              </div>
           </div>
        </section>

        <!-- POLICIES TAB -->
        <section v-show="activeTab === 'policies'" class="fade-in" role="region" aria-labelledby="policies-title">
            <div class="panel glass-panel">
              <div class="panel-header" style="margin-bottom: 20px;">
                <h2 id="policies-title" class="panel-title">Control de Políticas QDD</h2>
                <p style="color: var(--text-secondary); font-size: 13px;">Activa o desactiva módulos de certificación en tiempo real. Los cambios se guardan en <code>.qdd/policies.yaml</code>.</p>
              </div>

              <div class="policies-grid" style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px;">
                
                <div class="policy-card glass-panel" style="padding: 16px; border: 1px solid var(--border-color); display: flex; align-items: center; justify-content: space-between;">
                  <div>
                    <h3 style="margin: 0; font-size: 15px; color: var(--text-primary);">OWASP Security</h3>
                    <p style="margin: 4px 0 0; font-size: 12px; color: var(--text-secondary);">Análisis de vulnerabilidades (Inyecciones, Secrets, CORS).</p>
                  </div>
                  <label class="switch">
                    <input type="checkbox" :checked="state?.policies?.owasp" @change="togglePolicy('owasp')">
                    <span class="slider round"></span>
                  </label>
                </div>
                
                <div class="policy-card glass-panel" style="padding: 16px; border: 1px solid var(--border-color); display: flex; align-items: center; justify-content: space-between;">
                  <div>
                    <h3 style="margin: 0; font-size: 15px; color: var(--text-primary);">Clean Code</h3>
                    <p style="margin: 4px 0 0; font-size: 12px; color: var(--text-secondary);">Reglas generales de legibilidad y arquitectura limpia.</p>
                  </div>
                  <label class="switch">
                    <input type="checkbox" :checked="state?.policies?.clean_code" @change="togglePolicy('clean_code')">
                    <span class="slider round"></span>
                  </label>
                </div>

                <div class="policy-card glass-panel" style="padding: 16px; border: 1px solid var(--border-color); display: flex; align-items: center; justify-content: space-between; background: rgba(59, 130, 246, 0.05);">
                  <div>
                    <h3 style="margin: 0; font-size: 15px; color: var(--accent-color);">Regla: Zero-Else</h3>
                    <p style="margin: 4px 0 0; font-size: 12px; color: var(--text-secondary);">Obliga el uso de Early Returns (Guards). Si la apagas, se ignoran las fallas estructurales de este tipo.</p>
                  </div>
                  <label class="switch">
                    <input type="checkbox" :checked="state?.policies?.zero_else" @change="togglePolicy('zero_else')">
                    <span class="slider round"></span>
                  </label>
                </div>

                <div class="policy-card glass-panel" style="padding: 16px; border: 1px solid var(--border-color); display: flex; align-items: center; justify-content: space-between;">
                  <div>
                    <h3 style="margin: 0; font-size: 15px; color: var(--text-primary);">Traceability (ADRs)</h3>
                    <p style="margin: 4px 0 0; font-size: 12px; color: var(--text-secondary);">Requiere documentación estricta para cambios arquitectónicos.</p>
                  </div>
                  <label class="switch">
                    <input type="checkbox" :checked="state?.policies?.traceability" @change="togglePolicy('traceability')">
                    <span class="slider round"></span>
                  </label>
                </div>

                <div class="policy-card glass-panel" style="padding: 16px; border: 1px solid var(--border-color); display: flex; align-items: center; justify-content: space-between;">
                  <div>
                    <h3 style="margin: 0; font-size: 15px; color: var(--text-primary);">Beyond Limits (DoD/NASA)</h3>
                    <p style="margin: 4px 0 0; font-size: 12px; color: var(--text-secondary);">Reglas extremas (Sin asignación dinámica de memoria, etc).</p>
                  </div>
                  <label class="switch">
                    <input type="checkbox" :checked="state?.policies?.beyond_limits" @change="togglePolicy('beyond_limits')">
                    <span class="slider round"></span>
                  </label>
                </div>

                <div class="policy-card glass-panel" style="padding: 16px; border: 1px solid var(--border-color); display: flex; align-items: center; justify-content: space-between; background: rgba(236, 72, 153, 0.05);">
                  <div>
                    <h3 style="margin: 0; font-size: 15px; color: #ec4899;">Allow Execution (Mutation Mode)</h3>
                    <p style="margin: 4px 0 0; font-size: 12px; color: var(--text-secondary);">Si se apaga, el Agente y CLI operan 100% en Modo Auditoría. Ningún comando mutará código.</p>
                  </div>
                  <label class="switch">
                    <input type="checkbox" :checked="state?.policies?.allow_execution" @change="togglePolicy('allow_execution')">
                    <span class="slider round"></span>
                  </label>
                </div>

              </div>
              
              <div v-if="state?.policies && !state.policies.owasp" class="audit-badge fail" style="margin-top: 20px; display: inline-block;">
                ⚠️ OWASP Scanner Desactivado. El código no está siendo validado por brechas de seguridad.
              </div>
            </div>
        </section>

      </div>
      
      <!-- Slide Over Panel -->
      <div v-if="activeDetail" class="slide-over-backdrop" @click="closeDetail"></div>
      <aside class="slide-over" :class="{ 'is-open': activeDetail }" aria-label="Detail Panel">
        <div class="slide-over-header">
          <div class="slide-title-group">
            <span class="slide-type">{{ activeDetail?.type }}</span>
            <h2>{{ activeDetail?.id }}</h2>
          </div>
          <button class="close-btn" aria-label="Cerrar detalles" @click="closeDetail">✕</button>
        </div>
        <div class="slide-over-content">
          <div class="detail-block">
            <h4>Status</h4>
            <span class="status-pill" :class="activeDetail?.status === 'PASS' || activeDetail?.status === 'RESOLVED' || activeDetail?.status === 'certified' ? 'resolved' : (activeDetail?.status === 'IN-PROGRESS' ? 'in-progress' : 'open')">{{ activeDetail?.status }}</span>
          </div>

          <template v-if="activeDetail?.raw">
            <div class="detail-block" v-for="(val, key) in activeDetail.raw" :key="key" v-show="key !== 'id' && key !== 'status'">
              <h4 class="raw-key">{{ key.charAt(0).toUpperCase() + key.slice(1).replace(/_/g, ' ') }}</h4>
              
              <div v-if="Array.isArray(val)" class="raw-array">
                <ul>
                  <li v-for="(item, idx) in val" :key="idx">{{ item }}</li>
                </ul>
              </div>
              
              <div v-if="!Array.isArray(val) && typeof val === 'object' && val !== null" class="raw-object">
                <div v-for="(v, k) in val" :key="k" class="raw-object-item">
                  <span class="raw-sub-key">{{ k }}:</span> <span class="raw-sub-val">{{ v }}</span>
                </div>
              </div>
              
              <p v-if="!Array.isArray(val) && (typeof val !== 'object' || val === null)" class="detail-text">{{ val }}</p>
            </div>
          </template>

          <template v-if="activeDetail?.type === 'Knowledge'">
            <div class="knowledge-content">
              <pre class="detail-text" style="white-space: pre-wrap; word-break: break-word;">{{ activeDetail.content }}</pre>
            </div>
          </template>

          <template v-if="activeDetail?.type === 'Certification' && activeDetail.history">
            <h3 style="font-size: 14px; margin: 24px 0 12px; color: var(--text-primary); border-bottom: 1px solid var(--border-color); padding-bottom: 8px;">Run History (Timeline)</h3>
            <div class="prefect-board" style="display: flex; flex-direction: column; gap: 8px;">
               <div v-for="(run, idx) in activeDetail.history" :key="idx" class="run-row glass-panel" style="display: flex; align-items: center; justify-content: space-between; padding: 12px; border-radius: 6px; border-left: 4px solid;" :style="{ borderLeftColor: run.status === 'PASS' ? 'var(--success)' : 'var(--danger)', background: 'rgba(0,0,0,0.2)' }">
                  <div style="display: flex; flex-direction: column;">
                     <span style="font-family: monospace; font-size: 12px; color: var(--text-primary);">{{ run.run_id }}</span>
                     <span style="font-size: 11px; color: var(--text-muted);">{{ new Date(run.timestamp).toLocaleString() }}</span>
                  </div>
                  <div style="display: flex; align-items: center; gap: 12px;">
                     <span style="font-size: 12px; font-family: monospace; color: var(--text-muted);">{{ run.duration }}</span>
                     <span class="status-pill" :class="run.status === 'PASS' ? 'resolved' : 'open'" style="font-size: 10px; padding: 2px 6px;">{{ run.status }}</span>
                  </div>
               </div>
            </div>
          </template>
        </div>
      </aside>
    </main>
  </div>
</template>

<style>
/* Premium SaaS Theme - Linear/Vercel inspired */
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');

.btn {
  background: var(--bg-surface, rgba(255,255,255,0.05));
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 6px 14px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-family: inherit;
}
.btn:hover {
  background: rgba(255,255,255,0.1);
  border-color: rgba(255,255,255,0.2);
}
.btn-outline {
  background: transparent;
  border: 1px solid var(--border-color);
}
.tool-btn {
  background: rgba(0,0,0,0.4);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
  backdrop-filter: blur(8px);
}
.tool-btn:hover {
  background: rgba(255,255,255,0.1);
  border-color: rgba(255,255,255,0.3);
}

:root {
  --bg-dark: #09090b;
  --bg-panel: #18181b;
  --bg-panel-hover: #27272a;
  --border-color: rgba(255,255,255,0.08);
  --text-primary: #fafafa;
  --text-secondary: #a1a1aa;
  --text-muted: #71717a;
  
  --accent-color: #3b82f6;
  --accent-glow: rgba(59, 130, 246, 0.15);
  
  --success: #10b981;
  --success-bg: rgba(16, 185, 129, 0.1);
  --warning: #f59e0b;
  --danger: #ef4444;
  --danger-bg: rgba(239, 68, 68, 0.1);
  
  --font-sans: 'Inter', -apple-system, sans-serif;
}

/* SWITCH COMPONENT */
.switch {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
}
.switch input { 
  opacity: 0;
  width: 0;
  height: 0;
}
.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--bg-dark);
  border: 1px solid var(--text-muted);
  -webkit-transition: .2s;
  transition: .2s;
}
.slider:before {
  position: absolute;
  content: "";
  height: 16px;
  width: 16px;
  left: 3px;
  bottom: 3px;
  background-color: var(--text-muted);
  -webkit-transition: .2s;
  transition: .2s;
}
input:checked + .slider {
  background-color: var(--success);
  border-color: var(--success);
}
input:checked + .slider:before {
  background-color: white;
  -webkit-transform: translateX(20px);
  -ms-transform: translateX(20px);
  transform: translateX(20px);
}
.slider.round {
  border-radius: 24px;
}
.slider.round:before {
  border-radius: 50%;
}

/* MARKDOWN STYLES */
.markdown-body h1, .markdown-body h2, .markdown-body h3 {
  color: var(--text-primary);
  margin-top: 24px;
  margin-bottom: 12px;
}
.markdown-body h1 { font-size: 24px; border-bottom: 1px solid var(--border-color); padding-bottom: 8px; }
.markdown-body h2 { font-size: 20px; }
.markdown-body h3 { font-size: 16px; }
.markdown-body p { margin-bottom: 16px; }
.markdown-body code {
  background: rgba(0,0,0,0.3);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 13px;
  color: var(--accent-color);
}
.markdown-body pre {
  background: #0d1117;
  padding: 16px;
  border-radius: 8px;
  overflow-x: auto;
  border: 1px solid var(--border-color);
}
.markdown-body pre code {
  background: transparent;
  padding: 0;
  color: #e5e7eb;
}
.markdown-body blockquote {
  border-left: 4px solid var(--accent-color);
  margin: 0 0 16px 0;
  padding-left: 16px;
  color: var(--text-muted);
}
.markdown-body ul, .markdown-body ol {
  padding-left: 24px;
  margin-bottom: 16px;
}

.segmented-control .btn-active {
  background: rgba(59, 130, 246, 0.2) !important;
  color: #60a5fa !important;
}

body {
  margin: 0;
  padding: 0;
  background-color: var(--bg-dark);
  color: var(--text-primary);
  font-family: var(--font-sans);
  -webkit-font-smoothing: antialiased;
  letter-spacing: -0.01em;
}

*:focus-visible {
  outline: 2px solid var(--accent-color);
  outline-offset: 2px;
  border-radius: 4px;
}

/* DASHBOARD 2.0 STYLES */
.bg-orb {
  position: fixed;
  border-radius: 50%;
  filter: blur(80px);
  z-index: 0;
  opacity: 0.6;
  animation: float 10s infinite ease-in-out alternate;
}
.orb-1 {
  width: 500px;
  height: 500px;
  background: radial-gradient(circle, rgba(59, 130, 246, 0.4) 0%, rgba(59, 130, 246, 0) 70%);
  top: -150px;
  left: 10%;
}
.orb-2 {
  width: 450px;
  height: 450px;
  background: radial-gradient(circle, rgba(16, 185, 129, 0.3) 0%, rgba(16, 185, 129, 0) 70%);
  bottom: -100px;
  right: 10%;
  animation-delay: -5s;
}
.orb-3 {
  position: fixed;
  border-radius: 50%;
  filter: blur(100px);
  z-index: 0;
  opacity: 0.5;
  width: 600px;
  height: 600px;
  background: radial-gradient(circle, rgba(139, 92, 246, 0.25) 0%, rgba(139, 92, 246, 0) 70%);
  top: 30%;
  left: 40%;
  animation: float 15s infinite ease-in-out alternate-reverse;
}
@keyframes float {
  0% { transform: translate(0, 0) scale(1); }
  100% { transform: translate(30px, 50px) scale(1.1); }
}

.saas-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  position: relative;
  z-index: 1;
}

.sidebar {
  width: 260px;
  background-color: var(--bg-dark);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  z-index: 10; 
  background: rgba(24, 24, 27, 0.7); 
  backdrop-filter: blur(16px); 
  -webkit-backdrop-filter: blur(16px);
}

.brand {
  padding: 24px 20px;
  display: flex;
  align-items: center;
  gap: 12px;
  border-bottom: 1px solid var(--border-color);
}
.brand-logo {
  width: 24px;
  height: 24px;
  object-fit: contain;
}

.brand h1 {
  font-size: 15px;
  font-weight: 600;
  margin: 0;
  color: var(--text-primary);
  white-space: nowrap;
}

.version-tag {
  font-size: 10px;
  background: rgba(255,255,255,0.1);
  padding: 2px 6px;
  border-radius: 4px;
  color: var(--text-secondary);
  font-weight: 500;
}

.nav-section {
  padding: 20px 12px;
  flex: 1;
}

.nav-label {
  font-size: 11px;
  text-transform: uppercase;
  color: var(--text-muted);
  font-weight: 600;
  margin-bottom: 12px;
  padding-left: 8px;
  letter-spacing: 0.05em;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 13px;
  font-weight: 500;
  border-radius: 6px;
  transition: all 0.2s ease;
  margin-bottom: 2px;
}

.nav-item:hover {
  background-color: var(--bg-panel-hover);
  color: var(--text-primary);
}

.nav-item.active {
  background-color: var(--accent-glow);
  color: var(--accent-color);
}

.nav-item .icon {
  font-size: 14px;
  opacity: 0.8;
}

.sync-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(16, 185, 129, 0.2);
  border-top-color: var(--success);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.sidebar-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-secondary);
}

.pulse-dot {
  width: 8px;
  height: 8px;
  background-color: var(--success);
  border-radius: 50%;
  box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.4);
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.4); }
  70% { box-shadow: 0 0 0 6px rgba(16, 185, 129, 0); }
  100% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }
}

.pulse-dot.disconnected {
  background-color: var(--danger);
  box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.4);
  animation: pulse-danger 2s infinite;
}

@keyframes pulse-danger {
  0% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.4); }
  70% { box-shadow: 0 0 0 6px rgba(239, 68, 68, 0); }
  100% { box-shadow: 0 0 0 0 rgba(239, 68, 68, 0); }
}

.loading-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  font-size: 14px;
  color: var(--text-secondary);
}

.content-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}

.top-nav {
  height: 60px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32px;
}

.glassmorphism {
  background: rgba(24, 24, 27, 0.4) !important;
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(255,255,255,0.05) !important;
}

.breadcrumbs {
  font-size: 13px;
  color: var(--text-secondary);
}

.breadcrumbs .current {
  color: var(--text-primary);
  font-weight: 500;
}

.separator {
  margin: 0 8px;
  opacity: 0.5;
}

.omnibar {
  display: flex;
  align-items: center;
  background: rgba(0,0,0,0.3);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 20px;
  padding: 4px 12px;
  width: 300px;
  transition: all 0.3s;
}
.omnibar:focus-within {
  border-color: var(--accent-color);
  box-shadow: 0 0 0 2px var(--accent-glow);
  width: 400px;
}
.omni-input {
  background: transparent;
  border: none;
  color: #fff;
  outline: none;
  width: 100%;
  font-family: inherit;
  font-size: 13px;
}
.omni-input::placeholder { color: #52525b; }
.omni-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  filter: grayscale(1);
  transition: filter 0.2s;
}
.omni-btn:hover { filter: grayscale(0); }

.search-input {
  background: rgba(0,0,0,0.2);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 14px;
  width: 250px;
  transition: border-color 0.2s;
}
.search-input:focus {
  outline: none;
  border-color: var(--accent-color);
}

.spinner {
  display: inline-block;
  animation: spin 1s linear infinite;
}
@keyframes spin { 100% { transform: rotate(360deg); } }

.header-actions {
  display: flex;
  align-items: center;
  gap: 16px;
}

.audit-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 100px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.audit-badge.pass {
  background: var(--success-bg);
  color: var(--success);
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.audit-badge.fail {
  background: var(--danger-bg);
  color: var(--danger);
  border: 1px solid rgba(239, 68, 68, 0.2);
}

.indicator-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: currentColor;
}

.page-content {
  flex: 1;
  overflow-y: auto;
  padding: 32px;
  padding-bottom: 100px; /* space for slide over shadow */
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
}

.grid-layout {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 24px;
  margin-bottom: 24px;
}

.grid-layout.cols-1 {
  grid-template-columns: 1fr;
}

.grid-layout.cols-3 {
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
}

.mb-section {
  margin-bottom: 24px;
}

.panel {
  background-color: var(--bg-panel);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 24px;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.glass-panel {
  background: rgba(24, 24, 27, 0.6) !important;
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid rgba(255,255,255,0.08) !important;
  box-shadow: 0 12px 40px 0 rgba(0, 0, 0, 0.4), inset 0 1px 0 rgba(255,255,255,0.1) !important;
}
.glass-panel:hover {
  background: rgba(24, 24, 27, 0.8) !important;
  border: 1px solid rgba(255,255,255,0.15) !important;
  transform: translateY(-4px);
  box-shadow: 0 16px 50px 0 rgba(0, 0, 0, 0.5), inset 0 1px 0 rgba(255,255,255,0.15) !important;
}

.panel-header {
  margin-bottom: 16px;
}

.panel-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  margin-bottom: 16px;
}

.score-panel {
  display: flex;
  flex-direction: column;
}

.score-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.score-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.score-value {
  font-size: 56px;
  font-weight: 800;
  letter-spacing: -0.05em;
  line-height: 1;
  background: linear-gradient(135deg, #60a5fa 0%, #3b82f6 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  filter: drop-shadow(0 0 12px rgba(59,130,246,0.4));
}

.score-grade {
  font-size: 13px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 4px;
  text-transform: uppercase;
}

.grade-a, .grade-w { color: var(--success); background: var(--success-bg); }
.grade-b { color: var(--warning); background: rgba(245, 158, 11, 0.1); }
.grade-c, .grade-d, .grade-f { color: var(--danger); background: var(--danger-bg); }

/* Animated Ring */
.qdd-ring-container {
  width: 90px;
  height: 90px;
  flex-shrink: 0;
}
.qdd-ring {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}
.ring-bg {
  fill: none;
  stroke: rgba(255,255,255,0.05);
  stroke-width: 8;
}
.ring-progress {
  fill: none;
  stroke: var(--success);
  stroke-width: 8;
  stroke-dasharray: 283;
  stroke-dashoffset: 283;
  stroke-linecap: round;
  transition: stroke-dashoffset 1s cubic-bezier(0.16, 1, 0.3, 1);
}

.stack-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.stack-item {
  display: flex;
  align-items: center;
  gap: 8px;
  background-color: var(--bg-dark);
  border: 1px solid var(--border-color);
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 13px;
}

.clean-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.list-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid var(--border-color);
}

.list-row:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.clickable-row {
  cursor: pointer;
  padding: 12px;
  margin: 0 -12px;
  border-radius: 8px;
  border-bottom: none;
}
.clickable-row:not(:last-child) {
  margin-bottom: 4px;
}
.clickable-row:hover {
  background-color: var(--bg-panel-hover);
}

.row-main {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.telemetry-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
.telemetry-stat {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.t-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: 0.05em;
}
.t-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
}

@media (max-width: 768px) {
  .saas-layout {
    flex-direction: column;
  }
  .sidebar {
    width: 100%;
    height: auto;
    border-right: none;
    border-bottom: 1px solid var(--border-color);
  }
  .brand {
    display: none;
  }
  .nav-section {
    display: flex;
    overflow-x: auto;
    padding: 12px;
    gap: 8px;
    align-items: center;
  }
  .nav-label, .sidebar-footer {
    display: none;
  }
  .nav-item {
    white-space: nowrap;
    margin-bottom: 0;
    padding: 10px 16px;
  }
  .top-nav {
    flex-wrap: wrap;
    height: auto;
    padding: 16px;
    gap: 12px;
  }
  .breadcrumbs {
    width: 100%;
  }
  .omnibar {
    width: 100%;
    margin-left: 0;
  }
  .page-content {
    padding: 16px;
  }
  .grid-layout {
    grid-template-columns: 1fr;
  }
  .score-row {
    justify-content: center;
    gap: 32px;
  }
}

.item-id, .finding-id {
  font-size: 13px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  color: var(--text-primary);
  flex: 1;
}

.is-resolved .finding-id {
  color: var(--text-muted);
  text-decoration: line-through;
}

.icon-pass { color: var(--success); }
.icon-fail { color: var(--danger); }

.status-pill {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 100px;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.status-pill.resolved, .status-pill.pass {
  background: var(--success-bg);
  color: var(--success);
}

.status-pill.open, .status-pill.fail {
  background: var(--danger-bg);
  color: var(--danger);
}

.status-pill.in-progress {
  background: rgba(59, 130, 246, 0.1);
  color: var(--accent-color);
}

.sprint-list {
  display: flex;
  flex-direction: column;
}

.sprint-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.sprint-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.sprint-icon {
  font-size: 16px;
}

.sprint-id {
  font-weight: 500;
  font-size: 14px;
}

.empty-state {
  color: var(--text-muted);
  font-size: 13px;
  font-style: italic;
  padding: 12px 0;
}

/* Slide Over Panel */
.slide-over-backdrop {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0,0,0,0.4);
  backdrop-filter: blur(4px);
  z-index: 40;
}

.slide-over {
  position: absolute;
  top: 0;
  right: -400px;
  width: 400px;
  height: 100%;
  background-color: var(--bg-panel);
  border-left: 1px solid var(--border-color);
  box-shadow: -8px 0 32px rgba(0,0,0,0.5);
  transition: right 0.3s cubic-bezier(0.16, 1, 0.3, 1);
  z-index: 50;
  display: flex;
  flex-direction: column;
}

.slide-over.is-open {
  right: 0;
}

.slide-over-header {
  padding: 24px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.slide-title-group h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  font-family: ui-monospace, SFMono-Regular, monospace;
}

.slide-type {
  font-size: 11px;
  color: var(--accent-color);
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: 0.05em;
  margin-bottom: 4px;
  display: block;
}

.close-btn {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 16px;
  padding: 4px;
  transition: color 0.2s;
}

.close-btn:hover {
  color: var(--text-primary);
}

.slide-over-content {
  padding: 24px;
  flex: 1;
  overflow-y: auto;
}

.detail-block {
  margin-bottom: 24px;
}

.detail-block h4 {
  margin: 0 0 8px 0;
  font-size: 12px;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.raw-key {
  color: var(--text-secondary) !important;
}

.detail-text {
  font-size: 14px;
  color: var(--text-primary);
  line-height: 1.5;
  margin: 0;
}

.raw-object {
  background: var(--bg-dark);
  padding: 12px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-size: 12px;
}

.raw-object-item {
  margin-bottom: 4px;
}
.raw-object-item:last-child {
  margin-bottom: 0;
}

.raw-sub-key {
  color: var(--accent-color);
}
.raw-sub-val {
  color: var(--text-primary);
}

.raw-array ul {
  margin: 0;
  padding-left: 20px;
  color: var(--text-primary);
  font-size: 14px;
}

.raw-array li {
  margin-bottom: 4px;
}

.detail-footer-mock {
  margin-top: 40px;
  padding-top: 24px;
  border-top: 1px dashed var(--border-color);
  font-size: 12px;
}

.muted {
  color: var(--text-muted);
}

/* CERTIFICATIONS STATS & TABLE STYLES */
.stats-cards {
  display: flex;
  gap: 16px;
  margin-bottom: 24px;
}
.stat-card {
  flex: 1;
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}
.stat-card.pass {
  border-color: rgba(16, 185, 129, 0.3);
  background: rgba(16, 185, 129, 0.05);
}
.stat-card.fail {
  border-color: rgba(239, 68, 68, 0.3);
  background: rgba(239, 68, 68, 0.05);
}
.stat-card .stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
}
.stat-card.pass .stat-value {
  color: var(--success);
}
.stat-card.fail .stat-value {
  color: var(--danger);
}
.stat-card .stat-label {
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

.table-responsive {
  width: 100%;
  overflow-x: auto;
}
.qdd-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}
.qdd-table th {
  padding: 12px 16px;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border-color);
}
.qdd-table td {
  padding: 16px;
  border-bottom: 1px solid rgba(255,255,255,0.05);
  vertical-align: top;
}
.qdd-table tbody tr {
  transition: background 0.2s ease;
  cursor: pointer;
}
.qdd-table tbody tr:hover {
  background: rgba(255, 255, 255, 0.03);
}
.qdd-table .status-cell {
  text-align: center;
}
.qdd-table .cert-name-cell strong {
  display: block;
  font-size: 14px;
  color: var(--accent-color);
}
.qdd-table .cert-subtitle {
  display: block;
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 4px;
}
.type-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 100px;
  font-size: 11px;
  font-weight: 600;
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-primary);
}
.type-badge.core {
  background: rgba(59, 130, 246, 0.15);
  color: #60a5fa;
  border: 1px solid rgba(59, 130, 246, 0.3);
}
.type-badge.proyecto {
  background: rgba(168, 85, 247, 0.15);
  color: #c084fc;
  border: 1px solid rgba(168, 85, 247, 0.3);
}
.desc-cell .truncate-text {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.fade-in {
  animation: fadeIn 0.5s cubic-bezier(0.16, 1, 0.3, 1) both;
}
.stagger-1 { animation-delay: 0.1s; }
.stagger-2 { animation-delay: 0.2s; }
.stagger-3 { animation-delay: 0.3s; }
.stagger-4 { animation-delay: 0.4s; }

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}

/* Corporate Lifecycle Map Styles */
.corp-step {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  width: 140px;
  padding: 24px 16px;
  background: var(--bg-panel);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  position: relative;
  transition: all 0.3s ease;
  z-index: 2;
}

.corp-step:hover {
  transform: translateY(-4px);
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5);
  border-color: rgba(255, 255, 255, 0.15);
}

.corp-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.corp-title {
  font-weight: 600;
  font-size: 14px;
  color: var(--text-primary);
  margin-bottom: 8px;
  letter-spacing: -0.02em;
}

.corp-desc {
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.4;
  margin-bottom: 12px;
  flex-grow: 1;
}

.corp-cmd {
  font-size: 10px;
  background: rgba(0, 0, 0, 0.4);
  padding: 4px 8px;
  border-radius: 6px;
  color: var(--text-muted);
  border: 1px solid rgba(255, 255, 255, 0.05);
  font-family: monospace;
}

.corp-locked {
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.1em;
  color: var(--danger);
  background: rgba(239, 68, 68, 0.1);
  padding: 4px 8px;
  border-radius: 6px;
  margin-top: 4px;
}

.corp-step.is-disabled {
  background: rgba(255, 255, 255, 0.02);
  border: 1px dashed rgba(255, 255, 255, 0.1);
  opacity: 0.6;
}
.corp-step.is-disabled:hover {
  transform: none;
  box-shadow: none;
}

.lifecycle-radial {
  position: relative;
  width: 650px;
  height: 650px;
  margin: 0 auto;
  border-radius: 50%;
}
.lifecycle-radial-ring {
  position: absolute;
  top: 0; left: 0; right: 0; bottom: 0;
  border-radius: 50%;
  border: 4px dashed rgba(255,255,255,0.1);
  animation: spin 60s linear infinite;
  z-index: 1;
}
@keyframes spin { 100% { transform: rotate(360deg); } }

.lifecycle-hub {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 160px;
  height: 160px;
  border-radius: 50%;
  background: var(--bg-panel);
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  border: 2px solid var(--accent-color);
  box-shadow: 0 0 30px rgba(59, 130, 246, 0.2);
  z-index: 10;
}

.corp-step {
  position: absolute;
  top: 50%;
  left: 50%;
  margin-top: -80px;
  margin-left: -70px;
  background: var(--bg-panel);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px;
  width: 140px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  z-index: 2;
  transition: all 0.3s ease;
}
.corp-step:hover {
  transform: scale(1.05) !important;
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5);
  border-color: rgba(255, 255, 255, 0.15);
  z-index: 100;
}

.lifecycle-linear {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: 40px;
  margin-bottom: 0px;
  position: relative;
  z-index: 20;
}
.linear-connector {
  height: 50px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.3);
}
.linear-connector svg {
  width: 24px; height: 24px;
}
.corp-step.linear-step {
  position: relative;
  top: auto; left: auto;
  margin: 0;
  transform: none !important;
  z-index: 20;
}

/* 7 items loop (8 slots) layout - 45deg steps from -90deg (Top) */
/* Slot 0 (-90deg) is empty for the linear entry point */
.step-3 { transform: rotate(-45deg) translate(325px) rotate(45deg); }
.step-4 { transform: rotate(0deg) translate(325px) rotate(0deg); }
.step-5 { transform: rotate(45deg) translate(325px) rotate(-45deg); }
.step-6 { transform: rotate(90deg) translate(325px) rotate(-90deg); }
.step-7 { transform: rotate(135deg) translate(325px) rotate(-135deg); }
.step-8 { transform: rotate(180deg) translate(325px) rotate(-180deg); }
.step-9 { transform: rotate(225deg) translate(325px) rotate(-225deg); }

@media (max-width: 900px) {
  .lifecycle-radial {
    width: auto;
    height: auto;
    display: flex;
    flex-direction: column;
    gap: 16px;
    align-items: center;
  }
  .lifecycle-radial-ring {
    display: none;
  }
  .lifecycle-hub {
    position: relative;
    top: auto;
    left: auto;
    transform: none;
    margin-bottom: 24px;
  }
  .corp-step {
    position: relative;
    top: auto;
    left: auto;
    margin: 0;
    transform: none !important;
  }
}

.glow-icon {
  filter: drop-shadow(0 0 8px currentColor);
}

</style>

