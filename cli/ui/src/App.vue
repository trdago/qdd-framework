<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import mermaid from 'mermaid'
import TopologyNode from './components/TopologyNode.vue'

mermaid.initialize({ startOnLoad: false, theme: 'dark' })

const state = ref(null)
const loading = ref(true)
const activeDetail = ref(null)
const activeTab = ref('overview')
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
  topology: false
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

const filteredKnowledge = computed(() => {
  if (!state.value?.knowledge) return []
  if (!searchQuery.value) return state.value.knowledge
  const q = searchQuery.value.toLowerCase()
  return state.value.knowledge.filter(k => 
    k.path.toLowerCase().includes(q) || 
    k.content.toLowerCase().includes(q)
  )
})

const renderLifecycle = async () => {
  if (!state.value?.knowledge) return
  const doc = state.value.knowledge.find(k => k.path === 'docs/command-reference.md')
  if (doc) {
    const match = doc.content.match(/```mermaid\n([\s\S]*?)```/)
    if (match && match[1]) {
      try {
        const { svg } = await mermaid.render('mermaid-chart', match[1])
        lifecycleSvg.value = svg
      } catch (err) {
        console.error("Mermaid error:", err)
      }
    }
  }
}

watch(activeTab, (newTab) => {
  if (newTab === 'lifecycle') {
    renderLifecycle()
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

const openDetail = (item, type) => {
  activeDetail.value = { ...item, type }
}
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
        <a href="#" class="nav-item" role="tab" :aria-selected="activeTab==='intelligence'" :class="{active: activeTab==='intelligence'}" @click.prevent="activeTab='intelligence'"><span class="icon">🧠</span> <span style="flex:1;">Intelligence</span><div v-if="syncingTabs['intelligence'] || state?.working_on === 'learn'" class="sync-spinner" aria-hidden="true"></div></a>
        <a href="#" class="nav-item" role="tab" :aria-selected="activeTab==='topology'" :class="{active: activeTab==='topology'}" @click.prevent="activeTab='topology'"><span class="icon">🕸️</span> <span style="flex:1;">Project Topology</span><div v-if="syncingTabs['topology'] || state?.working_on === 'map'" class="sync-spinner" aria-hidden="true"></div></a>
        <a href="#" class="nav-item" role="tab" :aria-selected="activeTab==='sprints'" :class="{active: activeTab==='sprints'}" @click.prevent="activeTab='sprints'"><span class="icon">🏃</span> <span style="flex:1;">Sprints</span><div v-if="syncingTabs['sprints'] || state?.working_on === 'sprint'" class="sync-spinner" aria-hidden="true"></div></a>
        <a href="#" class="nav-item" role="tab" :aria-selected="activeTab==='findings'" :class="{active: activeTab==='findings'}" @click.prevent="activeTab='findings'"><span class="icon">🐞</span> <span style="flex:1;">Findings</span><div v-if="syncingTabs['findings'] || state?.working_on === 'audit' || state?.working_on === 'review'" class="sync-spinner" aria-hidden="true"></div></a>
        <a href="#" class="nav-item" role="tab" :aria-selected="activeTab==='certifications'" :class="{active: activeTab==='certifications'}" @click.prevent="activeTab='certifications'"><span class="icon">🛡️</span> <span style="flex:1;">Certifications</span><div v-if="syncingTabs['certifications'] || state?.working_on === 'certify'" class="sync-spinner" aria-hidden="true"></div></a>
        <a href="#" class="nav-item" role="tab" :aria-selected="activeTab==='knowledge'" :class="{active: activeTab==='knowledge'}" @click.prevent="activeTab='knowledge'"><span class="icon">📚</span> <span style="flex:1;">Knowledge</span><div v-if="syncingTabs['knowledge'] || state?.working_on === 'docs'" class="sync-spinner" aria-hidden="true"></div></a>
        <a href="#" class="nav-item" role="tab" :aria-selected="activeTab==='lifecycle'" :class="{active: activeTab==='lifecycle'}" @click.prevent="activeTab='lifecycle'"><span class="icon">🗺️</span> <span style="flex:1;">Lifecycle Map</span><div v-if="syncingTabs['lifecycle'] || state?.working_on === 'release'" class="sync-spinner" aria-hidden="true"></div></a>
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
          <span>QDD</span> <span class="separator">/</span> <span class="current" style="text-transform: capitalize;">{{ activeTab }}</span>
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
        <section v-show="activeTab === 'overview'" class="grid-layout cols-3 fade-in" aria-label="Key Metrics">
          <div class="panel glass-panel score-panel" role="region" aria-labelledby="score-title">
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
          
          <div class="panel glass-panel" role="region" aria-labelledby="telemetry-title">
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

          <div class="panel glass-panel" role="region" aria-labelledby="stack-title">
            <h3 id="stack-title" class="panel-title">Infrastructure Stack</h3>
            <div class="stack-grid" aria-label="Technology Stack">
              <div class="stack-item" v-if="state.config?.architecture">
                <span class="stack-icon">🏗️</span>
                <span class="stack-name">{{ typeof state.config.architecture === 'string' ? state.config.architecture : state.config.architecture[0] }}</span>
              </div>
              <div class="stack-item" v-for="lang in state.config?.languages" :key="lang">
                <span class="stack-icon">⚡</span>
                <span class="stack-name">{{ lang }}</span>
              </div>
              <div class="stack-item" v-for="db in state.config?.databases" :key="db">
                <span class="stack-icon">🗄️</span>
                <span class="stack-name">{{ db }}</span>
              </div>
            </div>
          </div>
        </section>

        <!-- INTELLIGENCE TAB -->
        <section v-show="activeTab === 'intelligence'" class="fade-in" role="region" aria-labelledby="intel-title">
          <div v-if="state?.understanding" class="intel-layout">
            <div class="panel glass-panel intel-main" role="region">
              <h2 class="panel-title">Platform Understanding</h2>
              <div class="intel-summary">
                <p>{{ state.understanding.summary }}</p>
              </div>
              
              <div class="intel-grid">
                <div class="intel-box">
                  <h4><span class="icon">🧩</span> Components</h4>
                  <ul>
                    <li v-for="comp in state.understanding.components" :key="comp">{{ comp }}</li>
                  </ul>
                </div>
                
                <div class="intel-box">
                  <h4><span class="icon">🎯</span> Objectives</h4>
                  <ul>
                    <li v-for="obj in state.understanding.objectives" :key="obj">{{ obj }}</li>
                  </ul>
                </div>
                
                <div class="intel-box">
                  <h4><span class="icon">📐</span> Guidelines</h4>
                  <ul>
                    <li v-for="guide in state.understanding.guidelines" :key="guide">{{ guide }}</li>
                  </ul>
                </div>
              </div>
              
              <div class="intel-next-steps panel glass-panel mt-4" style="border-color: var(--accent-color);">
                <h4><span class="icon">🚀</span> Next Steps (Architect Recommendation)</h4>
                <p>{{ state.understanding.next_steps }}</p>
              </div>
            </div>
          </div>
          <div v-if="!state.understanding" class="empty-state">
            No intelligence report found. Run <code>qdd learn</code> to generate the platform understanding report.
          </div>
        </section>

        <!-- TOPOLOGY TAB -->
        <section v-show="activeTab === 'topology'" class="panel glass-panel fade-in" role="region" aria-labelledby="topology-title">
            <div class="panel-header" style="display: flex; justify-content: space-between; align-items: center;">
              <h2 id="topology-title" class="panel-title" style="margin: 0;">Certification Topology Map</h2>
              <div v-if="state?.topology" class="audit-badge" :class="state.topology.global_score === 100 ? 'pass' : 'fail'">
                Score: {{ state.topology.global_score }}%
              </div>
            </div>
            
            <div class="topology-container mt-4" v-if="state?.topology?.application">
              <div class="tree-root">
                <TopologyNode :node="state.topology.application" />
              </div>
            </div>
            <div v-if="!state?.topology" class="empty-state">
              No topology map found. Run <code>qdd map</code> in the terminal to generate it.
            </div>
        </section>

        <!-- SPRINTS TAB -->
        <section v-show="activeTab === 'sprints'" class="panel glass-panel fade-in mb-section" role="region" aria-labelledby="sprints-title">
          <div class="panel-header">
            <h2 id="sprints-title" class="panel-title">Active Sprints</h2>
          </div>
          <div class="sprint-list" aria-live="polite">
            <div v-if="state.sprints.length === 0" class="empty-state">No active sprints found. Run `qdd sprint` to begin.</div>
            <div class="sprint-row clickable-row" tabindex="0" v-for="sprint in state.sprints" :key="sprint.id" @click="openDetail(sprint, 'Sprint')" @keydown.enter="openDetail(sprint, 'Sprint')">
              <div class="sprint-info">
                <span class="sprint-icon">🏃</span>
                <span class="sprint-id">{{ sprint.id }}</span>
              </div>
              <span class="status-pill in-progress">{{ sprint.status || 'ACTIVE' }}</span>
            </div>
          </div>
        </section>

        <!-- CERTIFICATIONS TAB -->
        <section v-show="activeTab === 'certifications'" class="panel glass-panel fade-in" role="region" aria-labelledby="cert-title">
            <div class="panel-header" style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 16px;">
              <h2 id="cert-title" class="panel-title" style="margin: 0;">Certifications</h2>
              <input type="text" v-model="certSearchQuery" placeholder="Filtrar por conceptos o estándares..." class="search-input" />
            </div>
            
            <div class="stats-cards">
              <div class="stat-card">
                <span class="stat-value">{{ certStats.total }}</span>
                <span class="stat-label">Total Certs</span>
              </div>
              <div class="stat-card pass">
                <span class="stat-value">{{ certStats.pass }}</span>
                <span class="stat-label">Certified</span>
              </div>
              <div class="stat-card fail" v-if="certStats.fail > 0">
                <span class="stat-value">{{ certStats.fail }}</span>
                <span class="stat-label">Pending</span>
              </div>
            </div>

            <div class="table-responsive mt-4">
              <table class="qdd-table">
                <thead>
                  <tr>
                    <th width="50">Status</th>
                    <th width="30%">Certificación</th>
                    <th width="15%">Tipo</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-if="filteredCerts.length === 0">
                    <td colspan="4" class="empty-state">No se encontraron certificaciones.</td>
                  </tr>
                  <tr v-for="cert in filteredCerts" :key="cert.id" class="clickable-row" tabindex="0" @click="openDetail(cert, 'Certification')" @keydown.enter="openDetail(cert, 'Certification')">
                    <td class="status-cell">
                      <svg v-if="cert.status === 'PASS' || cert.status.toLowerCase() === 'certified'" class="icon-pass" viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="2" fill="none"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>
                      <svg v-if="cert.status !== 'PASS' && cert.status.toLowerCase() !== 'certified'" class="icon-fail" viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="2" fill="none"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>
                    </td>
                    <td class="cert-name-cell">
                      <strong>{{ cert.id }}</strong>
                      <div class="cert-subtitle" v-if="cert.raw?.title">{{ cert.raw.title }}</div>
                    </td>
                    <td>
                      <span class="type-badge" :class="cert.typeLabel.toLowerCase()">{{ cert.typeLabel }}</span>
                    </td>
                    <td class="desc-cell">
                      <span class="truncate-text">{{ cert.raw?.description || 'Sin descripción' }}</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
        </section>
        
        <!-- FINDINGS TAB -->
        <section v-show="activeTab === 'findings'" class="panel glass-panel fade-in" role="region" aria-labelledby="findings-title">
            <div class="panel-header">
              <h2 id="findings-title" class="panel-title">Technical Debt</h2>
            </div>
            <ul class="clean-list" aria-label="Findings List">
              <li v-for="f in state.findings" :key="f.id" tabindex="0" class="list-row clickable-row" :class="{ 'is-resolved': f.status.toUpperCase() === 'RESOLVED' }" role="listitem" @click="openDetail(f, 'Finding')" @keydown.enter="openDetail(f, 'Finding')">
                <div class="row-main">
                  <span class="finding-id">{{ f.id }}</span>
                  <span class="status-pill" :class="f.status.toUpperCase() === 'RESOLVED' ? 'resolved' : 'open'">{{ f.status }}</span>
                </div>
              </li>
            </ul>
        </section>

        <!-- KNOWLEDGE TAB -->
        <section v-show="activeTab === 'knowledge'" class="panel glass-panel fade-in" role="region" aria-labelledby="knowledge-title">
            <div class="panel-header" style="display: flex; justify-content: space-between; align-items: center;">
              <h2 id="knowledge-title" class="panel-title" style="margin: 0;">Knowledge Base</h2>
              <input type="text" v-model="searchQuery" placeholder="Search architecture, modules, etc..." class="search-input" />
            </div>
            <div class="sprint-list" aria-live="polite">
              <div v-if="filteredKnowledge.length === 0" class="empty-state">No knowledge documents found matching "{{ searchQuery }}".</div>
              <div class="sprint-row clickable-row" tabindex="0" v-for="doc in filteredKnowledge" :key="doc.id" @click="openDetail(doc, 'Knowledge')" @keydown.enter="openDetail(doc, 'Knowledge')">
                <div class="sprint-info">
                  <span class="sprint-icon">📄</span>
                  <span class="item-id">{{ doc.path }}</span>
                </div>
                <span class="status-pill in-progress">Markdown</span>
              </div>
            </div>
        </section>

        <!-- LIFECYCLE TAB -->
        <section v-show="activeTab === 'lifecycle'" class="panel glass-panel fade-in" role="region" aria-labelledby="lifecycle-title" style="display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 60vh;">
            <div class="panel-header" style="width: 100%; text-align: center; margin-bottom: 2rem;">
              <h2 id="lifecycle-title" class="panel-title" style="font-size: 24px;">Ciclo de Mejora Continua</h2>
              <p style="color: var(--text-secondary); font-size: 14px;">Generado dinámicamente desde <code>docs/command-reference.md</code></p>
            </div>
            <div v-if="lifecycleSvg" class="mermaid-container fade-in" v-html="lifecycleSvg" style="background: rgba(0,0,0,0.2); padding: 24px; border-radius: 12px; border: 1px solid var(--border-color); overflow-x: auto; width: 100%; display: flex; justify-content: center;"></div>
            <div v-if="!lifecycleSvg" class="empty-state">No se encontró diagrama Mermaid o el documento aún no está indexado. Ejecuta <code>qdd learn</code>.</div>
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
        </div>
      </aside>
    </main>
  </div>
</template>

<style>
/* Premium SaaS Theme - Linear/Vercel inspired */
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');

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
  width: 400px;
  height: 400px;
  background: rgba(59, 130, 246, 0.4);
  top: -100px;
  left: 200px;
}
.orb-2 {
  width: 300px;
  height: 300px;
  background: rgba(16, 185, 129, 0.35);
  bottom: -50px;
  right: 20%;
  animation-delay: -5s;
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
  grid-template-columns: repeat(auto-fit, minmax(400px, 1px));
  gap: 24px;
  margin-bottom: 24px;
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
  background: rgba(24, 24, 27, 0.5) !important;
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(255,255,255,0.05) !important;
  box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.3) !important;
}
.glass-panel:hover {
  background: rgba(24, 24, 27, 0.7) !important;
  border: 1px solid rgba(255,255,255,0.1) !important;
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
  font-size: 48px;
  font-weight: 700;
  letter-spacing: -0.04em;
  line-height: 1;
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
  animation: fadeIn 0.4s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
