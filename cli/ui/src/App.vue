<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const state = ref(null)
const loading = ref(true)
let pollInterval = null

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
      config: {
        architecture: ['Hexagonal Architecture'],
        languages: ['Go'],
        databases: ['PostgreSQL']
      }
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
    <nav class="sidebar" role="navigation" aria-label="Main Sidebar">
      <div class="brand">
        <img src="/logo.png" alt="QDD Logo" class="brand-logo" />
        <h1>QDD Framework</h1>
        <span class="version-tag">{{ state?.version }}</span>
      </div>
      <div class="nav-section">
        <div class="nav-label">Governance</div>
        <a href="#" class="nav-item active"><span class="icon">◱</span> Dashboard</a>
        <a href="#" class="nav-item"><span class="icon">🏃</span> Sprints</a>
        <a href="#" class="nav-item"><span class="icon">🐞</span> Findings</a>
        <a href="#" class="nav-item"><span class="icon">🛡️</span> Certifications</a>
      </div>
      
      <div class="sidebar-footer">
        <div class="status-indicator" aria-live="polite">
          <div class="pulse-dot" aria-hidden="true"></div> Live Sync Active
        </div>
      </div>
    </nav>

    <main v-if="!loading && state" class="content-area" id="main-content" aria-live="polite">
      <header class="top-nav" role="banner" aria-label="App Header">
        <div class="breadcrumbs">
          <span>QDD</span> <span class="separator">/</span> <span class="current">Dashboard</span>
        </div>
        <div class="header-actions">
          <div class="audit-badge" :class="state?.audit_status.startsWith('PASS') ? 'pass' : 'fail'">
            <span class="indicator-dot"></span> {{ state?.audit_status }}
          </div>
        </div>
      </header>

      <div class="page-content">
        <section class="grid-layout" aria-label="Key Metrics">
          <div class="panel score-panel" role="region" aria-labelledby="score-title" style="align-self: start;">
            <h3 id="score-title" class="panel-title">Quality Score</h3>
            <div class="score-container">
              <div class="score-value">{{ state.score }}</div>
              <div class="score-grade" :class="'grade-' + state.grade.charAt(0).toLowerCase()" style="white-space: nowrap;">Grade {{ state.grade }}</div>
            </div>
            <div class="score-chart-mock"></div>
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

        <section class="panel mb-section" role="region" aria-labelledby="sprints-title">
          <div class="panel-header">
            <h2 id="sprints-title" class="panel-title">Active Sprints</h2>
          </div>
          <div class="sprint-list" aria-live="polite">
            <div v-if="state.sprints.length === 0" class="empty-state">No active sprints found. Run `qdd sprint` to begin.</div>
            <div class="sprint-row" v-for="sprint in state.sprints" :key="sprint.id">
              <div class="sprint-info">
                <span class="sprint-icon">🏃</span>
                <span class="sprint-id">{{ sprint.id }}</span>
              </div>
              <span class="status-pill in-progress">{{ sprint.status }}</span>
            </div>
          </div>
        </section>

        <div class="grid-layout cols-2">
          <section class="panel" role="region" aria-labelledby="cert-title">
            <div class="panel-header">
              <h2 id="cert-title" class="panel-title">Certifications</h2>
            </div>
            <ul class="clean-list" aria-label="Certifications List">
              <li v-for="cert in state.certifications" :key="cert.id" class="list-row" role="listitem">
                <div class="row-main">
                  <svg v-if="cert.status === 'PASS'" class="icon-pass" viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" stroke-width="2" fill="none"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>
                  <svg v-else class="icon-fail" viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" stroke-width="2" fill="none"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>
                  <span class="item-id">{{ cert.id }}</span>
                </div>
                <span class="item-desc">{{ cert.name }}</span>
              </li>
            </ul>
          </section>

          <section class="panel" role="region" aria-labelledby="findings-title">
            <div class="panel-header">
              <h2 id="findings-title" class="panel-title">Technical Debt</h2>
            </div>
            <ul class="clean-list" aria-label="Findings List">
              <li v-for="f in state.findings" :key="f.id" class="list-row" :class="{ 'is-resolved': f.status === 'RESOLVED' }" role="listitem">
                <div class="row-main">
                  <span class="finding-id">{{ f.id }}</span>
                  <span class="status-pill" :class="f.status === 'RESOLVED' ? 'resolved' : 'open'">{{ f.status }}</span>
                </div>
                <p class="finding-desc">{{ f.desc }}</p>
              </li>
            </ul>
          </section>
        </div>
      </div>
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

/* Layout */
.saas-layout {
  display: flex;
  min-height: 100vh;
}

/* Sidebar */
.sidebar {
  width: 260px;
  background-color: var(--bg-panel);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  padding: 1.5rem 1rem;
  position: sticky;
  top: 0;
  height: 100vh;
  box-sizing: border-box;
}

.brand {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0 0.5rem 2rem;
}

.brand-logo {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  object-fit: contain;
}

.brand h1 {
  font-size: 1.1rem;
  font-weight: 600;
  margin: 0;
  letter-spacing: -0.02em;
}

.version-tag {
  font-size: 0.7rem;
  background: var(--bg-panel-hover);
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.nav-section {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  flex: 1;
}

.nav-label {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  font-weight: 600;
  margin: 1rem 0 0.5rem 0.5rem;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.75rem;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 0.9rem;
  font-weight: 500;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.nav-item:hover {
  background-color: var(--bg-panel-hover);
  color: var(--text-primary);
}

.nav-item.active {
  background-color: var(--bg-panel-hover);
  color: var(--text-primary);
  box-shadow: inset 2px 0 0 var(--text-primary);
}

.nav-item .icon {
  opacity: 0.7;
  font-size: 1rem;
}

.sidebar-footer {
  padding: 1rem 0.5rem 0;
  border-top: 1px solid var(--border-color);
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.pulse-dot {
  width: 8px;
  height: 8px;
  background-color: var(--success);
  border-radius: 50%;
  position: relative;
}

.pulse-dot::after {
  content: "";
  position: absolute;
  top: -2px; left: -2px; right: -2px; bottom: -2px;
  border-radius: 50%;
  border: 1px solid var(--success);
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

@keyframes pulse {
  0% { transform: scale(1); opacity: 1; }
  100% { transform: scale(2.5); opacity: 0; }
}

/* Main Content */
.content-area {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.top-nav {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 2rem;
  border-bottom: 1px solid var(--border-color);
  background-color: var(--bg-dark);
}

.breadcrumbs {
  font-size: 0.9rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.breadcrumbs .separator {
  margin: 0 0.5rem;
  color: var(--text-muted);
}

.breadcrumbs .current {
  color: var(--text-primary);
}

.audit-badge {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
  font-weight: 600;
  padding: 0.35rem 0.75rem;
  border-radius: 99px;
  border: 1px solid var(--border-color);
  background: var(--bg-panel);
}

.indicator-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.audit-badge.pass .indicator-dot { background-color: var(--success); }
.audit-badge.fail .indicator-dot { background-color: var(--danger); }
.audit-badge.pass { color: var(--text-secondary); }
.audit-badge.fail { color: var(--danger); border-color: var(--danger-bg); background: var(--danger-bg); }

/* Page Content */
.page-content {
  padding: 2rem;
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
}

.grid-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
  margin-bottom: 1.5rem;
}

.mb-section {
  margin-bottom: 1.5rem;
}

/* Panels */
.panel {
  background-color: var(--bg-panel);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 1.5rem;
  transition: transform 0.2s ease, border-color 0.2s ease;
  position: relative;
  overflow: hidden;
}

.panel:hover {
  border-color: rgba(255,255,255,0.15);
}

.panel-header {
  margin-bottom: 1rem;
}

.panel-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: -0.01em;
}

/* Score Panel Specifics */
.score-panel {
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.score-container {
  display: flex;
  align-items: baseline;
  gap: 1rem;
  margin-top: 1rem;
}

.score-value {
  font-size: 4.5rem;
  font-weight: 700;
  line-height: 1;
  letter-spacing: -0.04em;
  color: var(--text-primary);
}

.score-grade {
  font-size: 1.25rem;
  font-weight: 600;
  padding: 0.25rem 0.75rem;
  border-radius: 6px;
  background: var(--bg-panel-hover);
}

.grade-a, .grade-w { color: var(--success); }
.grade-b { color: var(--accent-color); }
.grade-c { color: var(--warning); }
.grade-d, .grade-f { color: var(--danger); }

/* Stack Grid */
.stack-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 0.75rem;
  margin-top: 1rem;
}

.stack-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background-color: var(--bg-dark);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-secondary);
  transition: all 0.2s ease;
}

.stack-item:hover {
  background-color: var(--bg-panel-hover);
  color: var(--text-primary);
}

/* Clean Lists */
.clean-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.list-row {
  display: flex;
  flex-direction: column;
  padding: 0.75rem 1rem;
  background-color: var(--bg-dark);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  gap: 0.5rem;
}

.row-main {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.item-id {
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--text-primary);
}

.item-desc {
  font-size: 0.85rem;
  color: var(--text-secondary);
  padding-left: 1.75rem;
}

.icon-pass { color: var(--success); }
.icon-fail { color: var(--danger); }

/* Findings */
.finding-id {
  font-weight: 600;
  font-size: 0.9rem;
  font-family: monospace;
}

.finding-desc {
  margin: 0;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.is-resolved {
  opacity: 0.5;
}
.is-resolved .finding-id {
  text-decoration: line-through;
}

/* Sprints */
.sprint-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.sprint-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  background-color: var(--bg-dark);
  border: 1px solid var(--border-color);
  border-radius: 8px;
}

.sprint-info {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.sprint-id {
  font-weight: 600;
  font-size: 0.95rem;
}

/* Pills & Badges */
.status-pill {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
}

.status-pill.resolved { background: var(--success-bg); color: var(--success); }
.status-pill.open { background: var(--danger-bg); color: var(--danger); }
.status-pill.in-progress { background: var(--accent-glow); color: var(--accent-color); border: 1px solid rgba(59, 130, 246, 0.2); }

.empty-state {
  color: var(--text-muted);
  font-size: 0.9rem;
  font-style: italic;
  padding: 1rem 0;
}
</style>
