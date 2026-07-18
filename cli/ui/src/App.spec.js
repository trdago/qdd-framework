import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import App from './App.vue'

vi.mock('vue-chartjs', () => {
  return {
    Line: {
      template: '<div>Mocked Chart</div>',
      props: ['data', 'options']
    }
  }
})

// Mock de ResizeObserver para Chart.js en jsdom
class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
global.ResizeObserver = ResizeObserver;

// Mock de Canvas para Chart.js en jsdom
HTMLCanvasElement.prototype.getContext = () => {
  return {
    fillRect: () => {},
    clearRect: () => {},
    getImageData: (x, y, w, h) => ({ data: new Array(w * h * 4) }),
    putImageData: () => {},
    createImageData: () => ([]),
    setTransform: () => {},
    drawImage: () => {},
    save: () => {},
    fillText: () => {},
    restore: () => {},
    beginPath: () => {},
    moveTo: () => {},
    lineTo: () => {},
    closePath: () => {},
    stroke: () => {},
    translate: () => {},
    scale: () => {},
    rotate: () => {},
    arc: () => {},
    fill: () => {},
    measureText: () => ({ width: 0 }),
    transform: () => {},
    rect: () => {},
    clip: () => {}
  }
};

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
      quality_status: 'PASS',
      findings: [],
      certifications: [],
      sprints: [],
      knowledge: [],
      config: {},
      telemetry: { uptime: '24h', memory_sys: '100 MB', goroutines: 42 },
      topology: { application: { name: 'QDD', children: [] } }
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

  it('garantiza que el Lifecycle Map (Ciclo de Mejora Continua) exista y renderice los pasos', async () => {
    await loadData(wrapper);
    
    // 1. Simular click en la pestaña Project Map
    const navItems = wrapper.findAll('.nav-item')
    const mapTab = navItems.find(w => w.text().includes('Project Map'))
    await mapTab.trigger('click')

    // 2. Buscar y hacer click en el botón de vista Lifecycle Map (ya no existe, se muestra debajo por defecto)
    
    // 3. Verificar que los 11 pasos del ciclo corporativo (radial + central) se rendericen
    const steps = wrapper.findAll('.corp-step')
    expect(steps.length).toBe(11)

    // 4. Verificar contenido clave de los pasos
    expect(steps[0].text()).toContain('QDD Framework') // Center node
    expect(steps[1].text()).toContain('Init')
    expect(steps[2].text()).toContain('Learn')
    expect(steps[3].text()).toContain('Map')
    expect(steps[4].text()).toContain('Validate')
    expect(steps[5].text()).toContain('Certify')
    expect(steps[6].text()).toContain('UI')
    expect(steps[7].text()).toContain('API')
    expect(steps[8].text()).toContain('DB')
    expect(steps[9].text()).toContain('Sprint')
    expect(steps[10].text()).toContain('Release')
  })

  it('alterna el estado de colapso de la barra lateral al hacer clic en el menú hamburguesa', async () => {
    const instance = wrapper.vm
    
    const sidebar = wrapper.find('nav.sidebar')
    const hamburgerBtn = wrapper.find('.hamburger-btn')

    // Por defecto inicia colapsado (según requerimiento)
    expect(sidebar.classes()).toContain('collapsed')

    // Click para expandir
    await hamburgerBtn.trigger('click')
    expect(sidebar.classes()).not.toContain('collapsed')

    // Click para colapsar nuevamente
    await hamburgerBtn.trigger('click')
    expect(sidebar.classes()).toContain('collapsed')
  })

  it('procesa datos de la historia y actualiza chartData dinámicamente sin hardcodear (Bug Regression Test)', async () => {
    await loadData(wrapper)
    
    const instance = MockEventSource.instances[0]
    const testDataWithHistory = {
      score: 95,
      grade: 'World-Class',
      version: 'v0.1.1',
      quality_status: 'PASS',
      findings: [],
      certifications: [],
      sprints: [],
      knowledge: [],
      config: {},
      telemetry: { uptime: '24h', memory_sys: '100 MB', goroutines: 42 },
      topology: { application: { name: 'QDD', children: [] } },
      historical_trends: [
        { date: 'Día 10', score: 10 },
        { date: 'Día 11', score: 99 }
      ]
    };
    instance.onmessage({ data: JSON.stringify(testDataWithHistory) })
    await wrapper.vm.$nextTick()

    // El chartData ahora debería mapear 'Día 10' y 'Día 11' en lugar de los mocks
    const currentChartData = wrapper.vm.chartData
    expect(currentChartData.labels).toEqual(['Día 10', 'Día 11'])
    expect(currentChartData.datasets[0].data).toEqual([10, 99])
  })
})
