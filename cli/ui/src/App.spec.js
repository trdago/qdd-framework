import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import App from './App.vue'

// Mock de EventSource para aislar la prueba del backend real
class MockEventSource {
  constructor(url) {
    this.url = url;
    MockEventSource.instances.push(this);
  }
  close() {}
}
MockEventSource.instances = [];
global.EventSource = MockEventSource;

describe('App.vue - Enterprise Dashboard', () => {
  let wrapper;

  beforeEach(() => {
    MockEventSource.instances = [];
    wrapper = mount(App, {
      attachTo: document.body
    })
  })

  const loadData = async (wrapper) => {
    const instance = MockEventSource.instances[0]
    const testData = {
      score: 95,
      grade: 'World-Class',
      version: 'v0.1.1',
      audit_status: 'PASS',
      findings: [],
      certifications: [],
      sprints: [],
      knowledge: [],
      config: {},
      telemetry: { uptime: '24h', memory_sys: '100 MB', goroutines: 42 }
    };
    instance.onmessage({ data: JSON.stringify(testData) });
    await wrapper.vm.$nextTick();
  }

  it('renderiza correctamente el layout principal y el Score (Quality Score)', async () => {
    await loadData(wrapper);
    expect(wrapper.find('#score-title').exists()).toBe(true)
    expect(wrapper.find('.score-value').exists()).toBe(true)
    expect(wrapper.find('.qdd-ring').exists()).toBe(true)
  })

  it('procesa correctamente los eventos SSE e inyecta la Telemetría (MEM, UPTIME, GOROUTINES)', async () => {
    await loadData(wrapper);
    expect(wrapper.find('#telemetry-title').exists()).toBe(true)
    
    const htmlText = wrapper.html()
    expect(htmlText).toContain('UPTIME')
    expect(htmlText).toContain('24h')
    expect(htmlText).toContain('MEM (SYS)')
    expect(htmlText).toContain('100 MB')
    expect(htmlText).toContain('GOROUTINES')
    expect(htmlText).toContain('42')
  })

  it('permite navegación entre pestañas (Tabs) ocultando paneles no activos', async () => {
    await loadData(wrapper);
    
    // La pestaña por defecto es 'overview'
    const overviewSection = wrapper.find('section[aria-label="Overview Dashboard"]')
    expect(overviewSection.element.style.display).not.toBe('none')

    const qualitySection = wrapper.find('section[aria-labelledby="quality-title"]')
    expect(qualitySection.element.style.display).toBe('none')
    
    // Simulamos el click en el Tab de Quality Gates
    const navItems = wrapper.findAll('.nav-item')
    const qualityTab = navItems.find(w => w.text().includes('Quality Gates'))
    await qualityTab.trigger('click')

    // Ahora Quality debe ser visible y Overview oculto
    expect(overviewSection.element.style.display).toBe('none')
    expect(qualitySection.element.style.display).not.toBe('none')
  })
})
