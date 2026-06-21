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
	
	export class AuditEntry {
	    Seq: number;
	    // Go type: time
	    At: any;
	    Action: string;
	    HostRef: string;
	    Subject: string;
	    Detail: Record<string, any>;
	    PrevMAC: string;
	    MAC: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Seq = source["Seq"];
	        this.At = this.convertValues(source["At"], null);
	        this.Action = source["Action"];
	        this.HostRef = source["HostRef"];
	        this.Subject = source["Subject"];
	        this.Detail = source["Detail"];
	        this.PrevMAC = source["PrevMAC"];
	        this.MAC = source["MAC"];
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
	export class ResourceChange {
	    Name: string;
	    Kind: string;
	    Action: string;
	
	    static createFrom(source: any = {}) {
	        return new ResourceChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Kind = source["Kind"];
	        this.Action = source["Action"];
	    }
	}
	export class ServiceChange {
	    Service: string;
	    Action: string;
	    Reasons: string[];
	    DropsAnonymousVolumes: boolean;
	    InterruptsRunning: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServiceChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Service = source["Service"];
	        this.Action = source["Action"];
	        this.Reasons = source["Reasons"];
	        this.DropsAnonymousVolumes = source["DropsAnonymousVolumes"];
	        this.InterruptsRunning = source["InterruptsRunning"];
	    }
	}
	export class ComposePlan {
	    Project: string;
	    HostRef: string;
	    Services: ServiceChange[];
	    Networks: ResourceChange[];
	    Volumes: ResourceChange[];
	    Degraded: boolean;
	    Destructive: boolean;
	    Notes: string[];
	
	    static createFrom(source: any = {}) {
	        return new ComposePlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Project = source["Project"];
	        this.HostRef = source["HostRef"];
	        this.Services = this.convertValues(source["Services"], ServiceChange);
	        this.Networks = this.convertValues(source["Networks"], ResourceChange);
	        this.Volumes = this.convertValues(source["Volumes"], ResourceChange);
	        this.Degraded = source["Degraded"];
	        this.Destructive = source["Destructive"];
	        this.Notes = source["Notes"];
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
	    NetworkMode: string;
	    ComposeProject: string;
	    ComposeService: string;
	    ComposeConfigHash: string;
	    ComposeConfigFiles: string;
	    ComposeWorkingDir: string;
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
	        this.NetworkMode = source["NetworkMode"];
	        this.ComposeProject = source["ComposeProject"];
	        this.ComposeService = source["ComposeService"];
	        this.ComposeConfigHash = source["ComposeConfigHash"];
	        this.ComposeConfigFiles = source["ComposeConfigFiles"];
	        this.ComposeWorkingDir = source["ComposeWorkingDir"];
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
	export class HostNetworkRef {
	    ContainerID: string;
	    ContainerName: string;
	
	    static createFrom(source: any = {}) {
	        return new HostNetworkRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ContainerID = source["ContainerID"];
	        this.ContainerName = source["ContainerName"];
	    }
	}
	export class PortBinding {
	    HostRef: string;
	    ContainerID: string;
	    ContainerName: string;
	    HostIP: string;
	    HostPort: number;
	    ContainerPort: number;
	    Protocol: string;
	    Reach: string;
	    Flagged: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PortBinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.HostRef = source["HostRef"];
	        this.ContainerID = source["ContainerID"];
	        this.ContainerName = source["ContainerName"];
	        this.HostIP = source["HostIP"];
	        this.HostPort = source["HostPort"];
	        this.ContainerPort = source["ContainerPort"];
	        this.Protocol = source["Protocol"];
	        this.Reach = source["Reach"];
	        this.Flagged = source["Flagged"];
	    }
	}
	export class ExposureMap {
	    HostRef: string;
	    RemoteTransport: boolean;
	    Bindings: PortBinding[];
	    HostNetwork: HostNetworkRef[];
	
	    static createFrom(source: any = {}) {
	        return new ExposureMap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.HostRef = source["HostRef"];
	        this.RemoteTransport = source["RemoteTransport"];
	        this.Bindings = this.convertValues(source["Bindings"], PortBinding);
	        this.HostNetwork = this.convertValues(source["HostNetwork"], HostNetworkRef);
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
	export class Operation {
	    ID: string;
	    HostRef: string;
	    Kind: string;
	    Target: string;
	    OptionSet: Record<string, any>;
	    Result: string;
	    BytesReclaimed: number;
	    // Go type: time
	    StartedAt: any;
	    // Go type: time
	    EndedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Operation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.HostRef = source["HostRef"];
	        this.Kind = source["Kind"];
	        this.Target = source["Target"];
	        this.OptionSet = source["OptionSet"];
	        this.Result = source["Result"];
	        this.BytesReclaimed = source["BytesReclaimed"];
	        this.StartedAt = this.convertValues(source["StartedAt"], null);
	        this.EndedAt = this.convertValues(source["EndedAt"], null);
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

export namespace journal {
	
	export class AuditStatus {
	    Entries: domain.AuditEntry[];
	    State: string;
	    Verified: boolean;
	    VerifiedCount: number;
	    Error: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Entries = this.convertValues(source["Entries"], domain.AuditEntry);
	        this.State = source["State"];
	        this.Verified = source["Verified"];
	        this.VerifiedCount = source["VerifiedCount"];
	        this.Error = source["Error"];
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

