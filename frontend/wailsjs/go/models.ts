export namespace config {
	
	export class AuthUser {
	    username: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthUser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	    }
	}
	export class AuthConfig {
	    enabled: boolean;
	    users: AuthUser[];
	
	    static createFrom(source: any = {}) {
	        return new AuthConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.users = this.convertValues(source["users"], AuthUser);
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
	
	export class RouteConfig {
	    enabled: boolean;
	    activeFile: string;
	
	    static createFrom(source: any = {}) {
	        return new RouteConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.activeFile = source["activeFile"];
	    }
	}
	export class UIConfig {
	    theme: string;
	    language: string;
	    startMinimized: boolean;
	    autoStartProxy: boolean;
	    showTrayIcon: boolean;
	    closeToTray: boolean;
	    trayStatusAndIp: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UIConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.language = source["language"];
	        this.startMinimized = source["startMinimized"];
	        this.autoStartProxy = source["autoStartProxy"];
	        this.showTrayIcon = source["showTrayIcon"];
	        this.closeToTray = source["closeToTray"];
	        this.trayStatusAndIp = source["trayStatusAndIp"];
	    }
	}
	export class LogConfig {
	    level: string;
	    maxSizeMb: number;
	    maxBackups: number;
	    output: string;
	
	    static createFrom(source: any = {}) {
	        return new LogConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.maxSizeMb = source["maxSizeMb"];
	        this.maxBackups = source["maxBackups"];
	        this.output = source["output"];
	    }
	}
	export class RelayConfig {
	    dialTimeoutSec: number;
	    readTimeoutSec: number;
	    maxConnections: number;
	    keepaliveSec: number;
	
	    static createFrom(source: any = {}) {
	        return new RelayConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dialTimeoutSec = source["dialTimeoutSec"];
	        this.readTimeoutSec = source["readTimeoutSec"];
	        this.maxConnections = source["maxConnections"];
	        this.keepaliveSec = source["keepaliveSec"];
	    }
	}
	export class ProtocolConfig {
	    enabled: boolean;
	    host: string;
	    port: number;
	
	    static createFrom(source: any = {}) {
	        return new ProtocolConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.host = source["host"];
	        this.port = source["port"];
	    }
	}
	export class ServerConfig {
	    socks5: ProtocolConfig;
	    http: ProtocolConfig;
	
	    static createFrom(source: any = {}) {
	        return new ServerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.socks5 = this.convertValues(source["socks5"], ProtocolConfig);
	        this.http = this.convertValues(source["http"], ProtocolConfig);
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
	export class Config {
	    server: ServerConfig;
	    auth: AuthConfig;
	    relay: RelayConfig;
	    log: LogConfig;
	    ui: UIConfig;
	    route: RouteConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = this.convertValues(source["server"], ServerConfig);
	        this.auth = this.convertValues(source["auth"], AuthConfig);
	        this.relay = this.convertValues(source["relay"], RelayConfig);
	        this.log = this.convertValues(source["log"], LogConfig);
	        this.ui = this.convertValues(source["ui"], UIConfig);
	        this.route = this.convertValues(source["route"], RouteConfig);
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
	
	export class OutboundBinding {
	    mode: string;
	    localIp: string;
	    interface: string;
	
	    static createFrom(source: any = {}) {
	        return new OutboundBinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.localIp = source["localIp"];
	        this.interface = source["interface"];
	    }
	}
	
	
	
	export class RouteFileInfo {
	    name: string;
	    isActive: boolean;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new RouteFileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.isActive = source["isActive"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class RouteRule {
	    id: string;
	    name: string;
	    enabled: boolean;
	    priority: number;
	    protocols: string[];
	    matchType: string;
	    targets: string[];
	    outbound: OutboundBinding;
	    remark: string;
	
	    static createFrom(source: any = {}) {
	        return new RouteRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	        this.priority = source["priority"];
	        this.protocols = source["protocols"];
	        this.matchType = source["matchType"];
	        this.targets = source["targets"];
	        this.outbound = this.convertValues(source["outbound"], OutboundBinding);
	        this.remark = source["remark"];
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
	export class RouteRuleSet {
	    name: string;
	    version: number;
	    updatedAt: string;
	    description: string;
	    rules: RouteRule[];
	
	    static createFrom(source: any = {}) {
	        return new RouteRuleSet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.updatedAt = source["updatedAt"];
	        this.description = source["description"];
	        this.rules = this.convertValues(source["rules"], RouteRule);
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

export namespace logger {
	
	export class Entry {
	    time: string;
	    level: string;
	    message: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.level = source["level"];
	        this.message = source["message"];
	        this.source = source["source"];
	    }
	}

}

export namespace platform {
	
	export class NetworkInterface {
	    name: string;
	    displayName: string;
	    addresses: string[];
	    up: boolean;
	    loopback: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NetworkInterface(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.addresses = source["addresses"];
	        this.up = source["up"];
	        this.loopback = source["loopback"];
	    }
	}
	export class TrayState {
	    enabled: boolean;
	    visible: boolean;
	    platform: string;
	    supportsMenu: boolean;
	    nativeStarted: boolean;
	    hideDescription: string;
	
	    static createFrom(source: any = {}) {
	        return new TrayState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.visible = source["visible"];
	        this.platform = source["platform"];
	        this.supportsMenu = source["supportsMenu"];
	        this.nativeStarted = source["nativeStarted"];
	        this.hideDescription = source["hideDescription"];
	    }
	}

}

export namespace proxy {
	
	export class ConnectionSnapshot {
	    id: number;
	    protocol: string;
	    clientAddr: string;
	    targetAddr: string;
	    routeRuleName: string;
	    outboundIp: string;
	    outboundIface: string;
	    uploadBytes: number;
	    downloadBytes: number;
	    openedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.protocol = source["protocol"];
	        this.clientAddr = source["clientAddr"];
	        this.targetAddr = source["targetAddr"];
	        this.routeRuleName = source["routeRuleName"];
	        this.outboundIp = source["outboundIp"];
	        this.outboundIface = source["outboundIface"];
	        this.uploadBytes = source["uploadBytes"];
	        this.downloadBytes = source["downloadBytes"];
	        this.openedAt = source["openedAt"];
	    }
	}
	export class Status {
	    running: boolean;
	    startedAt: string;
	    socks5Addr: string;
	    httpAddr: string;
	    activeConns: number;
	    totalConns: number;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.startedAt = source["startedAt"];
	        this.socks5Addr = source["socks5Addr"];
	        this.httpAddr = source["httpAddr"];
	        this.activeConns = source["activeConns"];
	        this.totalConns = source["totalConns"];
	    }
	}

}

export namespace stats {
	
	export class Stats {
	    activeConns: number;
	    totalConns: number;
	    uploadBytes: number;
	    downloadBytes: number;
	    uploadRate: number;
	    downloadRate: number;
	    authFailures: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activeConns = source["activeConns"];
	        this.totalConns = source["totalConns"];
	        this.uploadBytes = source["uploadBytes"];
	        this.downloadBytes = source["downloadBytes"];
	        this.uploadRate = source["uploadRate"];
	        this.downloadRate = source["downloadRate"];
	        this.authFailures = source["authFailures"];
	    }
	}

}

