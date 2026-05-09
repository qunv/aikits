export namespace kg {
	
	export class GraphEdge {
	    id: number;
	    kind: string;
	    src: number;
	    dst: number;
	    confidence: number;
	    provenance: string;
	
	    static createFrom(source: any = {}) {
	        return new GraphEdge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.src = source["src"];
	        this.dst = source["dst"];
	        this.confidence = source["confidence"];
	        this.provenance = source["provenance"];
	    }
	}
	export class GraphNode {
	    id: number;
	    lang: string;
	    kind: string;
	    name: string;
	    fqn: string;
	    signature: string;
	    visibility: string;
	    startLine: number;
	
	    static createFrom(source: any = {}) {
	        return new GraphNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.lang = source["lang"];
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.fqn = source["fqn"];
	        this.signature = source["signature"];
	        this.visibility = source["visibility"];
	        this.startLine = source["startLine"];
	    }
	}
	export class GraphData {
	    nodes: GraphNode[];
	    edges: GraphEdge[];
	
	    static createFrom(source: any = {}) {
	        return new GraphData(source);
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
	
	
	export class IndexOptions {
	    Full: boolean;
	    Lang: string[];
	    Jobs: number;
	
	    static createFrom(source: any = {}) {
	        return new IndexOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Full = source["Full"];
	        this.Lang = source["Lang"];
	        this.Jobs = source["Jobs"];
	    }
	}
	export class IndexResult {
	    Indexed: number;
	    Unchanged: number;
	    Symbols: number;
	    Callsites: number;
	    Errors: number;
	
	    static createFrom(source: any = {}) {
	        return new IndexResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Indexed = source["Indexed"];
	        this.Unchanged = source["Unchanged"];
	        this.Symbols = source["Symbols"];
	        this.Callsites = source["Callsites"];
	        this.Errors = source["Errors"];
	    }
	}
	export class StatusResult {
	    RepoName: string;
	    RootPath: string;
	    Files: number;
	    Symbols: number;
	    Callsites: number;
	    Resolved: number;
	    // Go type: time
	    LastIndexed?: any;
	
	    static createFrom(source: any = {}) {
	        return new StatusResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RepoName = source["RepoName"];
	        this.RootPath = source["RootPath"];
	        this.Files = source["Files"];
	        this.Symbols = source["Symbols"];
	        this.Callsites = source["Callsites"];
	        this.Resolved = source["Resolved"];
	        this.LastIndexed = this.convertValues(source["LastIndexed"], null);
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
	export class Symbol {
	    ID: number;
	    Lang: string;
	    Kind: string;
	    Name: string;
	    FQN: string;
	    Signature: string;
	    Visibility: string;
	    StartLine: number;
	
	    static createFrom(source: any = {}) {
	        return new Symbol(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Lang = source["Lang"];
	        this.Kind = source["Kind"];
	        this.Name = source["Name"];
	        this.FQN = source["FQN"];
	        this.Signature = source["Signature"];
	        this.Visibility = source["Visibility"];
	        this.StartLine = source["StartLine"];
	    }
	}

}

export namespace scaffold {
	
	export class AgentInfo {
	    Name: string;
	    DestDir: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.DestDir = source["DestDir"];
	    }
	}
	export class InitOptions {
	    Target: string;
	    Agents: string[];
	
	    static createFrom(source: any = {}) {
	        return new InitOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Target = source["Target"];
	        this.Agents = source["Agents"];
	    }
	}
	export class InitResult {
	    TargetDir: string;
	    Created: string[];
	    Skipped: string[];
	
	    static createFrom(source: any = {}) {
	        return new InitResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.TargetDir = source["TargetDir"];
	        this.Created = source["Created"];
	        this.Skipped = source["Skipped"];
	    }
	}
	export class SkillAddOptions {
	    SkillName: string;
	    Agents: string[];
	    GitRoot: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillAddOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SkillName = source["SkillName"];
	        this.Agents = source["Agents"];
	        this.GitRoot = source["GitRoot"];
	    }
	}
	export class SkillAddResult {
	    SkillName: string;
	    Agent: string;
	    DestDir: string;
	    Created: string[];
	    Skipped: string[];
	    Err: any;
	
	    static createFrom(source: any = {}) {
	        return new SkillAddResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SkillName = source["SkillName"];
	        this.Agent = source["Agent"];
	        this.DestDir = source["DestDir"];
	        this.Created = source["Created"];
	        this.Skipped = source["Skipped"];
	        this.Err = source["Err"];
	    }
	}

}

export namespace types {
	
	export class SearchInput {
	    query: string;
	    contextTags?: string[];
	    scope?: string;
	    limit?: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.contextTags = source["contextTags"];
	        this.scope = source["scope"];
	        this.limit = source["limit"];
	    }
	}
	export class SearchResultItem {
	    id: string;
	    title: string;
	    content: string;
	    tags: string[];
	    scope: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResultItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.tags = source["tags"];
	        this.scope = source["scope"];
	        this.score = source["score"];
	    }
	}
	export class SearchResult {
	    results: SearchResultItem[];
	    totalMatches: number;
	    query: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], SearchResultItem);
	        this.totalMatches = source["totalMatches"];
	        this.query = source["query"];
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
	
	export class StoreInput {
	    title: string;
	    content: string;
	    tags?: string[];
	    scope?: string;
	
	    static createFrom(source: any = {}) {
	        return new StoreInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.content = source["content"];
	        this.tags = source["tags"];
	        this.scope = source["scope"];
	    }
	}
	export class StoreResult {
	    success: boolean;
	    id?: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new StoreResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.id = source["id"];
	        this.message = source["message"];
	    }
	}
	export class UpdateInput {
	    ID: string;
	    Title?: string;
	    Content?: string;
	    Tags: string[];
	    TagsProvided: boolean;
	    Scope?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Title = source["Title"];
	        this.Content = source["Content"];
	        this.Tags = source["Tags"];
	        this.TagsProvided = source["TagsProvided"];
	        this.Scope = source["Scope"];
	    }
	}
	export class UpdateResult {
	    success: boolean;
	    id: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.id = source["id"];
	        this.message = source["message"];
	    }
	}

}

