export namespace main {
	
	export class AgentEnableHooksArgs {
	    agent: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentEnableHooksArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = source["agent"];
	    }
	}
	export class AgentHooksStatusArgs {
	    agent: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentHooksStatusArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = source["agent"];
	    }
	}
	export class GitCheckoutBranchArgs {
	    repoRoot: string;
	    branch: string;
	
	    static createFrom(source: any = {}) {
	        return new GitCheckoutBranchArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	        this.branch = source["branch"];
	    }
	}
	export class GitCommitArgs {
	    repoRoot: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new GitCommitArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	        this.message = source["message"];
	    }
	}
	export class GitCommitFileDiffArgs {
	    repoRoot: string;
	    sha: string;
	    path: string;
	    originalPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new GitCommitFileDiffArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	        this.sha = source["sha"];
	        this.path = source["path"];
	        this.originalPath = source["originalPath"];
	    }
	}
	export class GitDiffArgs {
	    repoRoot: string;
	    path: string;
	    staged: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GitDiffArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	        this.path = source["path"];
	        this.staged = source["staged"];
	    }
	}
	export class GitDiffContentArgs {
	    repoRoot: string;
	    path: string;
	    staged: boolean;
	    originalPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new GitDiffContentArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	        this.path = source["path"];
	        this.staged = source["staged"];
	        this.originalPath = source["originalPath"];
	    }
	}
	export class GitDiscardArgs {
	    repoRoot: string;
	    entries: types.GitDiscardEntry[];
	
	    static createFrom(source: any = {}) {
	        return new GitDiscardArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	        this.entries = this.convertValues(source["entries"], types.GitDiscardEntry);
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
	export class GitFetchArgs {
	    repoRoot: string;
	
	    static createFrom(source: any = {}) {
	        return new GitFetchArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	    }
	}
	export class GitLogArgs {
	    repoRoot: string;
	    limit?: number;
	    beforeSha?: string;
	
	    static createFrom(source: any = {}) {
	        return new GitLogArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	        this.limit = source["limit"];
	        this.beforeSha = source["beforeSha"];
	    }
	}
	export class GitPanelSnapshotArgs {
	    cwd: string;
	
	    static createFrom(source: any = {}) {
	        return new GitPanelSnapshotArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cwd = source["cwd"];
	    }
	}
	export class GitRemoteURLArgs {
	    repoRoot: string;
	    name?: string;
	
	    static createFrom(source: any = {}) {
	        return new GitRemoteURLArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	        this.name = source["name"];
	    }
	}
	export class GitResolveRepoArgs {
	    cwd: string;
	
	    static createFrom(source: any = {}) {
	        return new GitResolveRepoArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cwd = source["cwd"];
	    }
	}
	export class GitShowCommitArgs {
	    repoRoot: string;
	    sha: string;
	
	    static createFrom(source: any = {}) {
	        return new GitShowCommitArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	        this.sha = source["sha"];
	    }
	}
	export class GitStageArgs {
	    repoRoot: string;
	    paths: string[];
	
	    static createFrom(source: any = {}) {
	        return new GitStageArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	        this.paths = source["paths"];
	    }
	}
	export class GitStatusArgs {
	    repoRoot: string;
	
	    static createFrom(source: any = {}) {
	        return new GitStatusArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	    }
	}
	export class HistoryCommandsArgs {
	    prefix: string;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new HistoryCommandsArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prefix = source["prefix"];
	        this.limit = source["limit"];
	    }
	}
	export class ListSubdirsArgs {
	    path: string;
	    showHidden: boolean;
	    workspace: types.WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new ListSubdirsArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.showHidden = source["showHidden"];
	        this.workspace = this.convertValues(source["workspace"], types.WorkspaceEnv);
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
	export class LspResolveRootArgs {
	    path: string;
	    markers: string[];
	
	    static createFrom(source: any = {}) {
	        return new LspResolveRootArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.markers = source["markers"];
	    }
	}
	export class LspSendArgs {
	    id: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LspSendArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.message = source["message"];
	    }
	}
	export class OpenSettingsWindowArgs {
	    tab?: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenSettingsWindowArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tab = source["tab"];
	    }
	}
	export class OpenerOpenPathArgs {
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenerOpenPathArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	    }
	}
	export class OpenerOpenURLArgs {
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenerOpenURLArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	    }
	}
	export class ProcessExitArgs {
	    code: number;
	
	    static createFrom(source: any = {}) {
	        return new ProcessExitArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	    }
	}
	export class PtyResizeArgs {
	    id: number;
	    cols: number;
	    rows: number;
	
	    static createFrom(source: any = {}) {
	        return new PtyResizeArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.cols = source["cols"];
	        this.rows = source["rows"];
	    }
	}
	export class PtyWriteArgs {
	    id: number;
	    data: number[];
	
	    static createFrom(source: any = {}) {
	        return new PtyWriteArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.data = source["data"];
	    }
	}
	export class SecretsDeleteArgs {
	    service: string;
	    account: string;
	
	    static createFrom(source: any = {}) {
	        return new SecretsDeleteArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.service = source["service"];
	        this.account = source["account"];
	    }
	}
	export class SecretsGetAllArgs {
	    service: string;
	    accounts: string[];
	
	    static createFrom(source: any = {}) {
	        return new SecretsGetAllArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.service = source["service"];
	        this.accounts = source["accounts"];
	    }
	}
	export class SecretsGetArgs {
	    service: string;
	    account: string;
	
	    static createFrom(source: any = {}) {
	        return new SecretsGetArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.service = source["service"];
	        this.account = source["account"];
	    }
	}
	export class SecretsSetArgs {
	    service: string;
	    account: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new SecretsSetArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.service = source["service"];
	        this.account = source["account"];
	        this.password = source["password"];
	    }
	}
	export class ShellSessionCloseArgs {
	    id: number;
	
	    static createFrom(source: any = {}) {
	        return new ShellSessionCloseArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	    }
	}
	export class ShellSessionRunArgs {
	    id: number;
	    command: string;
	    cwd: string;
	
	    static createFrom(source: any = {}) {
	        return new ShellSessionRunArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.command = source["command"];
	        this.cwd = source["cwd"];
	    }
	}
	export class WorkspaceAuthorizeArgs {
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceAuthorizeArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	    }
	}
	export class WslHomeArgs {
	    distro: string;
	
	    static createFrom(source: any = {}) {
	        return new WslHomeArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.distro = source["distro"];
	    }
	}

}

export namespace shell {
	
	export class BgProcessInfo {
	    handle: number;
	    command: string;
	    cwd: string;
	    started_at_ms: number;
	    exited: boolean;
	    exit_code?: number;
	
	    static createFrom(source: any = {}) {
	        return new BgProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.handle = source["handle"];
	        this.command = source["command"];
	        this.cwd = source["cwd"];
	        this.started_at_ms = source["started_at_ms"];
	        this.exited = source["exited"];
	        this.exit_code = source["exit_code"];
	    }
	}

}

export namespace types {
	
	export class AiHttpRequestArgs {
	    url: string;
	    method: string;
	    headers: Record<string, string>;
	    body: number[];
	    allowPrivateNetwork: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AiHttpRequestArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.method = source["method"];
	        this.headers = source["headers"];
	        this.body = source["body"];
	        this.allowPrivateNetwork = source["allowPrivateNetwork"];
	    }
	}
	export class AiHttpStreamArgs {
	    url: string;
	    method: string;
	    headers: Record<string, string>;
	    body: number[];
	    allowPrivateNetwork: boolean;
	    onEventEvent: string;
	
	    static createFrom(source: any = {}) {
	        return new AiHttpStreamArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.method = source["method"];
	        this.headers = source["headers"];
	        this.body = source["body"];
	        this.allowPrivateNetwork = source["allowPrivateNetwork"];
	        this.onEventEvent = source["onEventEvent"];
	    }
	}
	export class DirEntry {
	    name: string;
	    kind: string;
	    size: number;
	    mtime: number;
	    gitignored: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DirEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.size = source["size"];
	        this.mtime = source["mtime"];
	        this.gitignored = source["gitignored"];
	    }
	}
	export class WorkspaceEnv {
	    kind: string;
	    distro?: string;
	    cwd: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceEnv(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.distro = source["distro"];
	        this.cwd = source["cwd"];
	    }
	}
	export class FsCopyArgs {
	    sources: string[];
	    destDir: string;
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new FsCopyArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sources = source["sources"];
	        this.destDir = source["destDir"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class FsCreateArgs {
	    path: string;
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new FsCreateArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class FsDeleteArgs {
	    path: string;
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new FsDeleteArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class FsGlobArgs {
	    pattern: string;
	    root: string;
	    maxResults?: number;
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new FsGlobArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pattern = source["pattern"];
	        this.root = source["root"];
	        this.maxResults = source["maxResults"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class FsGlobHit {
	    path: string;
	    rel: string;
	
	    static createFrom(source: any = {}) {
	        return new FsGlobHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.rel = source["rel"];
	    }
	}
	export class FsGlobResponse {
	    hits: FsGlobHit[];
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FsGlobResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hits = this.convertValues(source["hits"], FsGlobHit);
	        this.truncated = source["truncated"];
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
	export class FsGrepArgs {
	    root: string;
	    pattern: string;
	    glob?: string[];
	    caseInsensitive: boolean;
	    maxResults: number;
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new FsGrepArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.pattern = source["pattern"];
	        this.glob = source["glob"];
	        this.caseInsensitive = source["caseInsensitive"];
	        this.maxResults = source["maxResults"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class FsGrepHit {
	    path: string;
	    rel: string;
	    line: number;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new FsGrepHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.rel = source["rel"];
	        this.line = source["line"];
	        this.text = source["text"];
	    }
	}
	export class FsGrepResponse {
	    hits: FsGrepHit[];
	    truncated: boolean;
	    files_scanned: number;
	
	    static createFrom(source: any = {}) {
	        return new FsGrepResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hits = this.convertValues(source["hits"], FsGrepHit);
	        this.truncated = source["truncated"];
	        this.files_scanned = source["files_scanned"];
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
	export class FsListFilesArgs {
	    pattern: string;
	    root: string;
	    limit?: number;
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new FsListFilesArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pattern = source["pattern"];
	        this.root = source["root"];
	        this.limit = source["limit"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class FsListFilesResult {
	    files: string[];
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FsListFilesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = source["files"];
	        this.truncated = source["truncated"];
	    }
	}
	export class FsReadDirArgs {
	    path: string;
	    showHidden: boolean;
	    gitDecorations: boolean;
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new FsReadDirArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.showHidden = source["showHidden"];
	        this.gitDecorations = source["gitDecorations"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class FsRenameArgs {
	    from: string;
	    to: string;
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new FsRenameArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class FsSearchArgs {
	    query: string;
	    root: string;
	    limit: number;
	    showHidden: boolean;
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new FsSearchArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.root = source["root"];
	        this.limit = source["limit"];
	        this.showHidden = source["showHidden"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class FsSearchHit {
	    path: string;
	    rel: string;
	    name: string;
	    is_dir: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FsSearchHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.rel = source["rel"];
	        this.name = source["name"];
	        this.is_dir = source["is_dir"];
	    }
	}
	export class FsSearchResponse {
	    hits: FsSearchHit[];
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FsSearchResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hits = this.convertValues(source["hits"], FsSearchHit);
	        this.truncated = source["truncated"];
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
	export class FsStat {
	    size: number;
	    mtime: number;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new FsStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.size = source["size"];
	        this.mtime = source["mtime"];
	        this.kind = source["kind"];
	    }
	}
	export class FsWatchArgs {
	    paths: string[];
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new FsWatchArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paths = source["paths"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class FsWriteArgs {
	    path: string;
	    content: string;
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new FsWriteArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.content = source["content"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class GitBranchEntry {
	    name: string;
	    kind: string;
	    worktreePath?: string;
	    isHead: boolean;
	    isDetached: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GitBranchEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.worktreePath = source["worktreePath"];
	        this.isHead = source["isHead"];
	        this.isDetached = source["isDetached"];
	    }
	}
	export class GitBranchListResult {
	    branches: GitBranchEntry[];
	
	    static createFrom(source: any = {}) {
	        return new GitBranchListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.branches = this.convertValues(source["branches"], GitBranchEntry);
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
	export class GitChangedFile {
	    path: string;
	    originalPath?: string;
	    indexStatus: string;
	    worktreeStatus: string;
	    staged: boolean;
	    unstaged: boolean;
	    untracked: boolean;
	    statusLabel: string;
	
	    static createFrom(source: any = {}) {
	        return new GitChangedFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.originalPath = source["originalPath"];
	        this.indexStatus = source["indexStatus"];
	        this.worktreeStatus = source["worktreeStatus"];
	        this.staged = source["staged"];
	        this.unstaged = source["unstaged"];
	        this.untracked = source["untracked"];
	        this.statusLabel = source["statusLabel"];
	    }
	}
	export class GitCommitFileChange {
	    path: string;
	    originalPath?: string;
	    status: string;
	    statusLabel: string;
	    added: number;
	    removed: number;
	    isBinary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GitCommitFileChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.originalPath = source["originalPath"];
	        this.status = source["status"];
	        this.statusLabel = source["statusLabel"];
	        this.added = source["added"];
	        this.removed = source["removed"];
	        this.isBinary = source["isBinary"];
	    }
	}
	export class GitCommitResult {
	    commitSha: string;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new GitCommitResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.commitSha = source["commitSha"];
	        this.summary = source["summary"];
	    }
	}
	export class GitDiffContentResult {
	    originalContent: string;
	    modifiedContent: string;
	    isBinary: boolean;
	    fallbackPatch: string;
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GitDiffContentResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalContent = source["originalContent"];
	        this.modifiedContent = source["modifiedContent"];
	        this.isBinary = source["isBinary"];
	        this.fallbackPatch = source["fallbackPatch"];
	        this.truncated = source["truncated"];
	    }
	}
	export class GitDiffResult {
	    diffText: string;
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GitDiffResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.diffText = source["diffText"];
	        this.truncated = source["truncated"];
	    }
	}
	export class GitDiscardEntry {
	    path: string;
	    untracked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GitDiscardEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.untracked = source["untracked"];
	    }
	}
	export class GitLogEntry {
	    sha: string;
	    shortSha: string;
	    author: string;
	    authorEmail: string;
	    timestampSecs: number;
	    parents: string[];
	    subject: string;
	    filesChanged: number;
	    insertions: number;
	    deletions: number;
	
	    static createFrom(source: any = {}) {
	        return new GitLogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sha = source["sha"];
	        this.shortSha = source["shortSha"];
	        this.author = source["author"];
	        this.authorEmail = source["authorEmail"];
	        this.timestampSecs = source["timestampSecs"];
	        this.parents = source["parents"];
	        this.subject = source["subject"];
	        this.filesChanged = source["filesChanged"];
	        this.insertions = source["insertions"];
	        this.deletions = source["deletions"];
	    }
	}
	export class GitStatusSnapshot {
	    repoRoot: string;
	    branch: string;
	    upstream?: string;
	    ahead: number;
	    behind: number;
	    isDetached: boolean;
	    truncated: boolean;
	    changedFiles: GitChangedFile[];
	
	    static createFrom(source: any = {}) {
	        return new GitStatusSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	        this.branch = source["branch"];
	        this.upstream = source["upstream"];
	        this.ahead = source["ahead"];
	        this.behind = source["behind"];
	        this.isDetached = source["isDetached"];
	        this.truncated = source["truncated"];
	        this.changedFiles = this.convertValues(source["changedFiles"], GitChangedFile);
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
	export class GitRepoInfo {
	    repoRoot: string;
	    branch: string;
	    upstream?: string;
	    isDetached: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GitRepoInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoRoot = source["repoRoot"];
	        this.branch = source["branch"];
	        this.upstream = source["upstream"];
	        this.isDetached = source["isDetached"];
	    }
	}
	export class GitPanelSnapshot {
	    repo?: GitRepoInfo;
	    status?: GitStatusSnapshot;
	
	    static createFrom(source: any = {}) {
	        return new GitPanelSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repo = this.convertValues(source["repo"], GitRepoInfo);
	        this.status = this.convertValues(source["status"], GitStatusSnapshot);
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
	export class GitPushResult {
	    remote?: string;
	    branch?: string;
	    pushed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GitPushResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.remote = source["remote"];
	        this.branch = source["branch"];
	        this.pushed = source["pushed"];
	    }
	}
	
	
	export class HistoryListArgs {
	    query?: string;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new HistoryListArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.limit = source["limit"];
	    }
	}
	export class HistoryRecordArgs {
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new HistoryRecordArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	    }
	}
	export class HistorySuggestArgs {
	    line: string;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new HistorySuggestArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line = source["line"];
	        this.limit = source["limit"];
	    }
	}
	export class LmPingArgs {
	    baseUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new LmPingArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.baseUrl = source["baseUrl"];
	    }
	}
	export class LspSpawnArgs {
	    command: string;
	    args: string[];
	    env: Record<string, string>;
	    root: string;
	    maxRssMb?: number;
	    workspace: WorkspaceEnv;
	    onMessageEvent: string;
	    onExitEvent: string;
	
	    static createFrom(source: any = {}) {
	        return new LspSpawnArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.args = source["args"];
	        this.env = source["env"];
	        this.root = source["root"];
	        this.maxRssMb = source["maxRssMb"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
	        this.onMessageEvent = source["onMessageEvent"];
	        this.onExitEvent = source["onExitEvent"];
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
	export class PtyOpenArgs {
	    cols: number;
	    rows: number;
	    cwd?: string;
	    workspace: WorkspaceEnv;
	    blocks: boolean;
	    shell?: string;
	    onDataEvent: string;
	    onExitEvent: string;
	
	    static createFrom(source: any = {}) {
	        return new PtyOpenArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cols = source["cols"];
	        this.rows = source["rows"];
	        this.cwd = source["cwd"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
	        this.blocks = source["blocks"];
	        this.shell = source["shell"];
	        this.onDataEvent = source["onDataEvent"];
	        this.onExitEvent = source["onExitEvent"];
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
	export class ReadResult {
	    kind: string;
	    content?: string;
	    size: number;
	    limit?: number;
	
	    static createFrom(source: any = {}) {
	        return new ReadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.content = source["content"];
	        this.size = source["size"];
	        this.limit = source["limit"];
	    }
	}
	export class ShellBgKillArgs {
	    handle: number;
	
	    static createFrom(source: any = {}) {
	        return new ShellBgKillArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.handle = source["handle"];
	    }
	}
	export class ShellBgLogsArgs {
	    handle: number;
	    sinceOffset: number;
	
	    static createFrom(source: any = {}) {
	        return new ShellBgLogsArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.handle = source["handle"];
	        this.sinceOffset = source["sinceOffset"];
	    }
	}
	export class ShellBgSpawnArgs {
	    command: string;
	    cwd: string;
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new ShellBgSpawnArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.cwd = source["cwd"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class ShellRunArgs {
	    command: string;
	    cwd: string;
	    timeoutSecs: number;
	    workspace: WorkspaceEnv;
	
	    static createFrom(source: any = {}) {
	        return new ShellRunArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.cwd = source["cwd"];
	        this.timeoutSecs = source["timeoutSecs"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
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
	export class ShellSessionOpenArgs {
	    cwd: string;
	    workspace: WorkspaceEnv;
	    shell?: string;
	
	    static createFrom(source: any = {}) {
	        return new ShellSessionOpenArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cwd = source["cwd"];
	        this.workspace = this.convertValues(source["workspace"], WorkspaceEnv);
	        this.shell = source["shell"];
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
	export class StoreLoadArgs {
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new StoreLoadArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	    }
	}
	export class StoreSaveArgs {
	    path: string;
	    data: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new StoreSaveArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.data = source["data"];
	    }
	}

}

export namespace workspace {
	
	export class WSLDistro {
	    name: string;
	    default: boolean;
	    running: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WSLDistro(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.default = source["default"];
	        this.running = source["running"];
	    }
	}

}

