export namespace config {
	
	export class UIConfig {
	    theme: string;
	    language: string;
	    startMinimized: boolean;
	    showTrayIcon: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UIConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.language = source["language"];
	        this.startMinimized = source["startMinimized"];
	        this.showTrayIcon = source["showTrayIcon"];
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
	    relay: RelayConfig;
	    log: LogConfig;
	    ui: UIConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = this.convertValues(source["server"], ServerConfig);
	        this.relay = this.convertValues(source["relay"], RelayConfig);
	        this.log = this.convertValues(source["log"], LogConfig);
	        this.ui = this.convertValues(source["ui"], UIConfig);
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

export namespace proxy {
	
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
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activeConns = source["activeConns"];
	        this.totalConns = source["totalConns"];
	        this.uploadBytes = source["uploadBytes"];
	        this.downloadBytes = source["downloadBytes"];
	    }
	}

}

