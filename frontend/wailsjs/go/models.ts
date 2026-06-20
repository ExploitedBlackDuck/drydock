export namespace app {
	
	export class AddHostInput {
	    name: string;
	    transport: string;
	    endpoint: string;
	    observeMode: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AddHostInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.transport = source["transport"];
	        this.endpoint = source["endpoint"];
	        this.observeMode = source["observeMode"];
	    }
	}
	export class HostDTO {
	    id: string;
	    name: string;
	    transport: string;
	    endpoint: string;
	    trust: string;
	    observeMode: boolean;
	    connected: boolean;
	    engineVersion: string;
	    apiVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new HostDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.transport = source["transport"];
	        this.endpoint = source["endpoint"];
	        this.trust = source["trust"];
	        this.observeMode = source["observeMode"];
	        this.connected = source["connected"];
	        this.engineVersion = source["engineVersion"];
	        this.apiVersion = source["apiVersion"];
	    }
	}

}

export namespace domain {
	
	export class Port {
	    IP: string;
	    PrivatePort: number;
	    PublicPort: number;
	    Protocol: string;
	
	    static createFrom(source: any = {}) {
	        return new Port(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.IP = source["IP"];
	        this.PrivatePort = source["PrivatePort"];
	        this.PublicPort = source["PublicPort"];
	        this.Protocol = source["Protocol"];
	    }
	}
	export class Container {
	    ID: string;
	    HostRef: string;
	    Name: string;
	    Image: string;
	    State: string;
	    Status: string;
	    Ports: Port[];
	    ComposeProject: string;
	    ComposeService: string;
	    // Go type: time
	    Created: any;
	
	    static createFrom(source: any = {}) {
	        return new Container(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.HostRef = source["HostRef"];
	        this.Name = source["Name"];
	        this.Image = source["Image"];
	        this.State = source["State"];
	        this.Status = source["Status"];
	        this.Ports = this.convertValues(source["Ports"], Port);
	        this.ComposeProject = source["ComposeProject"];
	        this.ComposeService = source["ComposeService"];
	        this.Created = this.convertValues(source["Created"], null);
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
	export class Image {
	    ID: string;
	    HostRef: string;
	    Repo: string;
	    Tag: string;
	    Size: number;
	    Dangling: boolean;
	    InUse: boolean;
	    // Go type: time
	    Created: any;
	
	    static createFrom(source: any = {}) {
	        return new Image(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.HostRef = source["HostRef"];
	        this.Repo = source["Repo"];
	        this.Tag = source["Tag"];
	        this.Size = source["Size"];
	        this.Dangling = source["Dangling"];
	        this.InUse = source["InUse"];
	        this.Created = this.convertValues(source["Created"], null);
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
	export class Network {
	    ID: string;
	    HostRef: string;
	    Name: string;
	    Driver: string;
	    InUse: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Network(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.HostRef = source["HostRef"];
	        this.Name = source["Name"];
	        this.Driver = source["Driver"];
	        this.InUse = source["InUse"];
	    }
	}
	
	export class PruneCategory {
	    Kind: string;
	    Label: string;
	    ObjectCount: number;
	    ReclaimableBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new PruneCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Kind = source["Kind"];
	        this.Label = source["Label"];
	        this.ObjectCount = source["ObjectCount"];
	        this.ReclaimableBytes = source["ReclaimableBytes"];
	    }
	}
	export class VolumeRef {
	    Name: string;
	    Size: number;
	    InUse: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VolumeRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Size = source["Size"];
	        this.InUse = source["InUse"];
	    }
	}
	export class PruneImpact {
	    Categories: PruneCategory[];
	    Volumes: VolumeRef[];
	    TotalReclaimable: number;
	
	    static createFrom(source: any = {}) {
	        return new PruneImpact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Categories = this.convertValues(source["Categories"], PruneCategory);
	        this.Volumes = this.convertValues(source["Volumes"], VolumeRef);
	        this.TotalReclaimable = source["TotalReclaimable"];
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
	export class ResourceSample {
	    HostRef: string;
	    ContainerID: string;
	    // Go type: time
	    At: any;
	    CPUPct: number;
	    MemBytes: number;
	    NetRx: number;
	    NetTx: number;
	    BlkRead: number;
	    BlkWrite: number;
	
	    static createFrom(source: any = {}) {
	        return new ResourceSample(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.HostRef = source["HostRef"];
	        this.ContainerID = source["ContainerID"];
	        this.At = this.convertValues(source["At"], null);
	        this.CPUPct = source["CPUPct"];
	        this.MemBytes = source["MemBytes"];
	        this.NetRx = source["NetRx"];
	        this.NetTx = source["NetTx"];
	        this.BlkRead = source["BlkRead"];
	        this.BlkWrite = source["BlkWrite"];
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
	export class StackService {
	    Name: string;
	    Containers: Container[];
	    Running: number;
	    Total: number;
	
	    static createFrom(source: any = {}) {
	        return new StackService(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Containers = this.convertValues(source["Containers"], Container);
	        this.Running = source["Running"];
	        this.Total = source["Total"];
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
	export class Stack {
	    Project: string;
	    HostRef: string;
	    Services: StackService[];
	    Running: number;
	    Total: number;
	    State: string;
	
	    static createFrom(source: any = {}) {
	        return new Stack(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Project = source["Project"];
	        this.HostRef = source["HostRef"];
	        this.Services = this.convertValues(source["Services"], StackService);
	        this.Running = source["Running"];
	        this.Total = source["Total"];
	        this.State = source["State"];
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
	
	export class Volume {
	    Name: string;
	    HostRef: string;
	    Driver: string;
	    Mountpoint: string;
	    Size: number;
	    InUse: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Volume(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.HostRef = source["HostRef"];
	        this.Driver = source["Driver"];
	        this.Mountpoint = source["Mountpoint"];
	        this.Size = source["Size"];
	        this.InUse = source["InUse"];
	    }
	}

}

