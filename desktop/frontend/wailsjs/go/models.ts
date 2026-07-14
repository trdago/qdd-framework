export namespace audit {
	
	export class QDDPolicies {
	    owasp: boolean;
	    clean_code: boolean;
	    zero_else: boolean;
	    beyond_limits: boolean;
	    traceability: boolean;
	    enterprise: boolean;
	    allow_execution: boolean;
	
	    static createFrom(source: any = {}) {
	        return new QDDPolicies(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.owasp = source["owasp"];
	        this.clean_code = source["clean_code"];
	        this.zero_else = source["zero_else"];
	        this.beyond_limits = source["beyond_limits"];
	        this.traceability = source["traceability"];
	        this.enterprise = source["enterprise"];
	        this.allow_execution = source["allow_execution"];
	    }
	}

}

export namespace dashboard {
	
	export class CertificationRun {
	    run_id: string;
	    timestamp: string;
	    status: string;
	    duration: string;
	
	    static createFrom(source: any = {}) {
	        return new CertificationRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run_id = source["run_id"];
	        this.timestamp = source["timestamp"];
	        this.status = source["status"];
	        this.duration = source["duration"];
	    }
	}
	export class DashboardCertification {
	    id: string;
	    name: string;
	    version: string;
	    status: string;
	    type: string;
	    raw?: Record<string, any>;
	    history?: CertificationRun[];
	
	    static createFrom(source: any = {}) {
	        return new DashboardCertification(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.status = source["status"];
	        this.type = source["type"];
	        this.raw = source["raw"];
	        this.history = this.convertValues(source["history"], CertificationRun);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DashboardFinding {
	    id: string;
	    status: string;
	    pillar: string;
	    desc: string;
	    raw: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new DashboardFinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.pillar = source["pillar"];
	        this.desc = source["desc"];
	        this.raw = source["raw"];
	    }
	}
	export class GraphEdge {
	    source: string;
	    target: string;
	    relation: string;
	
	    static createFrom(source: any = {}) {
	        return new GraphEdge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.target = source["target"];
	        this.relation = source["relation"];
	    }
	}
	export class GraphNode {
	    id: string;
	    type: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new GraphNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.name = source["name"];
	    }
	}
	export class DashboardGraphData {
	    nodes: GraphNode[];
	    edges: GraphEdge[];
	
	    static createFrom(source: any = {}) {
	        return new DashboardGraphData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodes = this.convertValues(source["nodes"], GraphNode);
	        this.edges = this.convertValues(source["edges"], GraphEdge);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DashboardKnowledgeDoc {
	    id: string;
	    path: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new DashboardKnowledgeDoc(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.content = source["content"];
	    }
	}
	export class DashboardSprint {
	    id: string;
	    status: string;
	    raw: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new DashboardSprint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.raw = source["raw"];
	    }
	}
	export class DashboardTelemetry {
	    uptime: string;
	    memory_alloc: string;
	    memory_sys: string;
	    goroutines: number;
	
	    static createFrom(source: any = {}) {
	        return new DashboardTelemetry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uptime = source["uptime"];
	        this.memory_alloc = source["memory_alloc"];
	        this.memory_sys = source["memory_sys"];
	        this.goroutines = source["goroutines"];
	    }
	}
	export class DashboardUnderstanding {
	    summary: string;
	    components: string[];
	    objectives: string[];
	    guidelines: string[];
	    next_steps: string;
	
	    static createFrom(source: any = {}) {
	        return new DashboardUnderstanding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = source["summary"];
	        this.components = source["components"];
	        this.objectives = source["objectives"];
	        this.guidelines = source["guidelines"];
	        this.next_steps = source["next_steps"];
	    }
	}
	
	
	export class HistoricalTrendPoint {
	    date: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new HistoricalTrendPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.score = source["score"];
	    }
	}
	export class ValueMetrics {
	    hours_saved: number;
	    debt_reduced: number;
	
	    static createFrom(source: any = {}) {
	        return new ValueMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hours_saved = source["hours_saved"];
	        this.debt_reduced = source["debt_reduced"];
	    }
	}
	export class QDDState {
	    score: number;
	    grade: string;
	    version: string;
	    audit_status: string;
	    findings: DashboardFinding[];
	    certifications: DashboardCertification[];
	    sprints: DashboardSprint[];
	    knowledge: DashboardKnowledgeDoc[];
	    understanding?: DashboardUnderstanding;
	    topology?: topology.ProjectTopology;
	    config: Record<string, any>;
	    telemetry: DashboardTelemetry;
	    working_on: string;
	    project_name: string;
	    value_metrics: ValueMetrics;
	    historical_trends: HistoricalTrendPoint[];
	    mcp_logs: string[];
	    usage_time: string;
	    policies: audit.QDDPolicies;
	    graph_data: DashboardGraphData;
	    auto_ui_certification: boolean;
	
	    static createFrom(source: any = {}) {
	        return new QDDState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.score = source["score"];
	        this.grade = source["grade"];
	        this.version = source["version"];
	        this.audit_status = source["audit_status"];
	        this.findings = this.convertValues(source["findings"], DashboardFinding);
	        this.certifications = this.convertValues(source["certifications"], DashboardCertification);
	        this.sprints = this.convertValues(source["sprints"], DashboardSprint);
	        this.knowledge = this.convertValues(source["knowledge"], DashboardKnowledgeDoc);
	        this.understanding = this.convertValues(source["understanding"], DashboardUnderstanding);
	        this.topology = this.convertValues(source["topology"], topology.ProjectTopology);
	        this.config = source["config"];
	        this.telemetry = this.convertValues(source["telemetry"], DashboardTelemetry);
	        this.working_on = source["working_on"];
	        this.project_name = source["project_name"];
	        this.value_metrics = this.convertValues(source["value_metrics"], ValueMetrics);
	        this.historical_trends = this.convertValues(source["historical_trends"], HistoricalTrendPoint);
	        this.mcp_logs = source["mcp_logs"];
	        this.usage_time = source["usage_time"];
	        this.policies = this.convertValues(source["policies"], audit.QDDPolicies);
	        this.graph_data = this.convertValues(source["graph_data"], DashboardGraphData);
	        this.auto_ui_certification = source["auto_ui_certification"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace topology {
	
	export class TopologyNode {
	    id: string;
	    name: string;
	    type: string;
	    path: string;
	    certified: boolean;
	    required_certs: string[];
	    missing_certs: string[];
	    tags: string[];
	    children: TopologyNode[];
	
	    static createFrom(source: any = {}) {
	        return new TopologyNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.path = source["path"];
	        this.certified = source["certified"];
	        this.required_certs = source["required_certs"];
	        this.missing_certs = source["missing_certs"];
	        this.tags = source["tags"];
	        this.children = this.convertValues(source["children"], TopologyNode);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProjectTopology {
	    application?: TopologyNode;
	    global_score: number;
	
	    static createFrom(source: any = {}) {
	        return new ProjectTopology(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.application = this.convertValues(source["application"], TopologyNode);
	        this.global_score = source["global_score"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

