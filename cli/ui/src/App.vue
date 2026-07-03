<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const state = ref(null)
const loading = ref(true)
const activeDetail = ref(null)
const activeTab = ref('overview')
const omniInput = ref('')
let pollInterval = null

const executeOmni = () => {
    if (!omniInput.value) return;
    alert("Ejecutando QDD Intent: " + omniInput.value + "\n(Próximamente backend)");
    omniInput.value = '';
}

const openDetail = (item, type) => {
  activeDetail.value = { ...item, type }
}
const closeDetail = () => {
  activeDetail.value = null
}

const fetchState = async () => {
  try {
    const res = await fetch('/api/state')
    state.value = await res.json()
  } catch (e) {
    console.error("Error loading QDD state:", e)
    // Fallback mock data if API is not running
    state.value = {
      score: 100,
      grade: 'World-Class',
      version: 'v0.1.1',
      findings: [
        { id: 'FND-002', status: 'RESOLVED', desc: 'Uso ilegal de else detectado y corregido.' }
      ],
      certifications: [
        { id: 'CERT-005', status: 'PASS', name: 'Clean Code' }
      ],
      sprints: [
        { id: 'Sprint-2', status: 'IN-PROGRESS'}
      ],
      config: {
        architecture: ['Hexagonal Architecture'],
        languages: ['Go'],
        databases: ['PostgreSQL']
      },
      audit_status: 'PASS'
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchState()
  pollInterval = setInterval(fetchState, 2000) // Poll every 2 seconds
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<template>
  <div class="saas-layout" role="main">
    <div class="bg-orb orb-1"></div>
    <div class="bg-orb orb-2"></div>
    
    <nav class="sidebar" role="navigation" aria-label="Main Sidebar">
      <div class="brand">
        <img src="/logo.png" alt="QDD Logo" class="brand-logo" />
        <h1>QDD Framework</h1>
        <span class="version-tag">{{ state?.version }}</span>
      </div>
      <div class="nav-section">
        <div class="nav-label">Governance</div>
        <a href="#" class="nav-item" :class="{active: activeTab==='overview'}" @click.prevent="activeTab='overview'"><span class="icon">◱</span> Overview</a>
        <a href="#" class="nav-item" :class="{active: activeTab==='sprints'}" @click.prevent="activeTab='sprints'"><span class="icon">🏃</span> Sprints</a>
        <a href="#" class="nav-item" :class="{active: activeTab==='findings'}" @click.prevent="activeTab='findings'"><span class="icon">🐞</span> Findings</a>
        <a href="#" class="nav-item" :class="{active: activeTab==='certifications'}" @click.prevent="activeTab='certifications'"><span class="icon">🛡️</span> Certifications</a>
      </div>
      
      <div class="sidebar-footer">
        <div class="status-indicator" aria-live="polite">
          <div class="pulse-dot" aria-hidden="true"></div> Live Sync Active
        </div>
      </div>
    </nav>

    <main v-if="!loading && state" class="content-area" id="main-content" aria-live="polite">
      <header class="top-nav glassmorphism" role="banner" aria-label="App Header">
        <div class="breadcrumbs">
          <span>QDD</span> <span class="separator">/</span> <span class="current" style="text-transform: capitalize;">{{ activeTab }}</span>
        </div>
        
        <div class="omnibar">
            <input type="text" v-model="omniInput" @keyup.enter="executeOmni" placeholder="QDD Intent (e.g. qdd sprint 3)..." class="omni-input" />
            <button @click="executeOmni" class="omni-btn">✨</button>
        </div>
        <div class="header-actions">
          <div class="audit-badge" :class="state?.audit_status.startsWith('PASS') ? 'pass' : 'fail'">
            <span class="indicator-dot"></span> {{ state?.audit_status }}
          </div>
        </div>
      </header>

      <div class="page-content">
        <!-- OVERVIEW TAB -->
        <section v-show="activeTab === 'overview'" class="grid-layout glass-panel fade-in" aria-label="Key Metrics">
          <div class="panel score-panel" role="region" aria-labelledby="score-title" style="align-self: start;">
            <h3 id="score-title" class="panel-title">Quality Score</h3>
            <div class="score-container">
              <div class="score-value">{{ state.score }}</div>
              <div class="score-grade" :class="'grade-' + state.grade.charAt(0).toLowerCase()" style="white-space: nowrap;">Grade {{ state.grade }}</div>
            </div>
            <div class="qdd-ring-container">
              <svg class="qdd-ring" viewBox="0 0 100 100">
                <circle class="ring-bg" cx="50" cy="50" r="45"></circle>
                <circle class="ring-progress" cx="50" cy="50" r="45" :style="{'stroke-dashoffset': 283 - (283 * state.score) / 100}"></circle>
              </svg>
            </div>
          </div>
          
          <div class="panel" role="region" aria-labelledby="stack-title" style="align-self: start;">
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

        <!-- SPRINTS TAB -->
        <section v-show="activeTab === 'sprints'" class="panel glass-panel fade-in mb-section" role="region" aria-labelledby="sprints-title">
          <div class="panel-header">
            <h2 id="sprints-title" class="panel-title">Active Sprints</h2>
          </div>
          <div class="sprint-list" aria-live="polite">
            <div v-if="state.sprints.length === 0" class="empty-state">No active sprints found. Run `qdd sprint` to begin.</div>
            <div class="sprint-row clickable-row" v-for="sprint in state.sprints" :key="sprint.id" @click="openDetail(sprint, 'Sprint')">
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
            <div class="panel-header">
              <h2 id="cert-title" class="panel-title">Certifications</h2>
            </div>
            <ul class="clean-list" aria-label="Certifications List">
              <li v-for="cert in state.certifications" :key="cert.id" class="list-row clickable-row" role="listitem" @click="openDetail(cert, 'Certification')">
                <div class="row-main">
                  <svg v-if="cert.status === 'PASS' || cert.status.toLowerCase() === 'certified'" class="icon-pass" viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" stroke-width="2" fill="none"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>
                  <svg v-else class="icon-fail" viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" stroke-width="2" fill="none"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>
                  <span class="item-id">{{ cert.id }}</span>
                </div>
              </li>
            </ul>
        </section>
        
        <!-- FINDINGS TAB -->
        <section v-show="activeTab === 'findings'" class="panel glass-panel fade-in" role="region" aria-labelledby="findings-title">
            <div class="panel-header">
              <h2 id="findings-title" class="panel-title">Technical Debt</h2>
            </div>
            <ul class="clean-list" aria-label="Findings List">
              <li v-for="f in state.findings" :key="f.id" class="list-row clickable-row" :class="{ 'is-resolved': f.status.toUpperCase() === 'RESOLVED' }" role="listitem" @click="openDetail(f, 'Finding')">
                <div class="row-main">
                  <span class="finding-id">{{ f.id }}</span>
                  <span class="status-pill" :class="f.status.toUpperCase() === 'RESOLVED' ? 'resolved' : 'open'">{{ f.status }}</span>
                </div>
              </li>
            </ul>
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
          <button class="close-btn" @click="closeDetail">✕</button>
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
              
              <div v-else-if="typeof val === 'object' && val !== null" class="raw-object">
                <div v-for="(v, k) in val" :key="k" class="raw-object-item">
                  <span class="raw-sub-key">{{ k }}:</span> <span class="raw-sub-val">{{ v }}</span>
                </div>
              </div>
              
              <p v-else class="detail-text">{{ val }}</p>
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

/* DASHBOARD 2.0 STYLES */
.bg-orb {
  position: fixed;
  border-radius: 50%;
  filter: blur(80px);
  z-index: 0;
  opacity: 0.4;
  animation: float 10s infinite ease-in-out alternate;
}
.orb-1 {
  width: 400px;
  height: 400px;
  background: rgba(59, 130, 246, 0.2);
  top: -100px;
  left: -100px;
}
.orb-2 {
  width: 300px;
  height: 300px;
  background: rgba(16, 185, 129, 0.15);
  bottom: -50px;
  right: 10%;
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
}

.grid-layout {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1px));
  gap: 24px;
  margin-bottom: 24px;
}

.grid-layout.cols-2 {
  grid-template-columns: 1fr 1fr;
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
  position: relative;
}

.score-container {
  display: flex;
  align-items: baseline;
  gap: 12px;
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
  width: 120px;
  height: 120px;
  position: absolute;
  right: 24px;
  top: 50%;
  transform: translateY(-50%);
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

.fade-in {
  animation: fadeIn 0.4s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
