// package: teleport.terminal.v1
// file: v1/cluster.proto

import * as jspb from "google-protobuf";

export class Cluster extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  getConnected(): boolean;
  setConnected(value: boolean): void;

  hasAcl(): boolean;
  clearAcl(): void;
  getAcl(): ClusterACL | undefined;
  setAcl(value?: ClusterACL): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Cluster.AsObject;
  static toObject(includeInstance: boolean, msg: Cluster): Cluster.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Cluster, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Cluster;
  static deserializeBinaryFromReader(message: Cluster, reader: jspb.BinaryReader): Cluster;
}

export namespace Cluster {
  export type AsObject = {
    name: string,
    connected: boolean,
    acl?: ClusterACL.AsObject,
  }
}

export class ResourceAccess extends jspb.Message {
  getList(): boolean;
  setList(value: boolean): void;

  getRead(): boolean;
  setRead(value: boolean): void;

  getEdit(): boolean;
  setEdit(value: boolean): void;

  getCreate(): boolean;
  setCreate(value: boolean): void;

  getDelete(): boolean;
  setDelete(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ResourceAccess.AsObject;
  static toObject(includeInstance: boolean, msg: ResourceAccess): ResourceAccess.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ResourceAccess, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ResourceAccess;
  static deserializeBinaryFromReader(message: ResourceAccess, reader: jspb.BinaryReader): ResourceAccess;
}

export namespace ResourceAccess {
  export type AsObject = {
    list: boolean,
    read: boolean,
    edit: boolean,
    create: boolean,
    pb_delete: boolean,
  }
}

export class ClusterACL extends jspb.Message {
  hasSessions(): boolean;
  clearSessions(): void;
  getSessions(): ResourceAccess | undefined;
  setSessions(value?: ResourceAccess): void;

  hasAuthconnectors(): boolean;
  clearAuthconnectors(): void;
  getAuthconnectors(): ResourceAccess | undefined;
  setAuthconnectors(value?: ResourceAccess): void;

  hasRoles(): boolean;
  clearRoles(): void;
  getRoles(): ResourceAccess | undefined;
  setRoles(value?: ResourceAccess): void;

  hasUsers(): boolean;
  clearUsers(): void;
  getUsers(): ResourceAccess | undefined;
  setUsers(value?: ResourceAccess): void;

  hasTrustedclusters(): boolean;
  clearTrustedclusters(): void;
  getTrustedclusters(): ResourceAccess | undefined;
  setTrustedclusters(value?: ResourceAccess): void;

  hasEvents(): boolean;
  clearEvents(): void;
  getEvents(): ResourceAccess | undefined;
  setEvents(value?: ResourceAccess): void;

  hasTokens(): boolean;
  clearTokens(): void;
  getTokens(): ResourceAccess | undefined;
  setTokens(value?: ResourceAccess): void;

  hasServers(): boolean;
  clearServers(): void;
  getServers(): ResourceAccess | undefined;
  setServers(value?: ResourceAccess): void;

  hasAppservers(): boolean;
  clearAppservers(): void;
  getAppservers(): ResourceAccess | undefined;
  setAppservers(value?: ResourceAccess): void;

  hasDbservers(): boolean;
  clearDbservers(): void;
  getDbservers(): ResourceAccess | undefined;
  setDbservers(value?: ResourceAccess): void;

  hasKubeservers(): boolean;
  clearKubeservers(): void;
  getKubeservers(): ResourceAccess | undefined;
  setKubeservers(value?: ResourceAccess): void;

  clearSshloginsList(): void;
  getSshloginsList(): Array<string>;
  setSshloginsList(value: Array<string>): void;
  addSshlogins(value: string, index?: number): string;

  hasAccessrequests(): boolean;
  clearAccessrequests(): void;
  getAccessrequests(): ResourceAccess | undefined;
  setAccessrequests(value?: ResourceAccess): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ClusterACL.AsObject;
  static toObject(includeInstance: boolean, msg: ClusterACL): ClusterACL.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ClusterACL, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ClusterACL;
  static deserializeBinaryFromReader(message: ClusterACL, reader: jspb.BinaryReader): ClusterACL;
}

export namespace ClusterACL {
  export type AsObject = {
    sessions?: ResourceAccess.AsObject,
    authconnectors?: ResourceAccess.AsObject,
    roles?: ResourceAccess.AsObject,
    users?: ResourceAccess.AsObject,
    trustedclusters?: ResourceAccess.AsObject,
    events?: ResourceAccess.AsObject,
    tokens?: ResourceAccess.AsObject,
    servers?: ResourceAccess.AsObject,
    appservers?: ResourceAccess.AsObject,
    dbservers?: ResourceAccess.AsObject,
    kubeservers?: ResourceAccess.AsObject,
    sshloginsList: Array<string>,
    accessrequests?: ResourceAccess.AsObject,
  }
}

export class ClusterAuthSettings extends jspb.Message {
  getType(): string;
  setType(value: string): void;

  getSecondfactor(): string;
  setSecondfactor(value: string): void;

  hasU2fs(): boolean;
  clearU2fs(): void;
  getU2fs(): AuthSettingsU2F | undefined;
  setU2fs(value?: AuthSettingsU2F): void;

  clearAuthprovidersList(): void;
  getAuthprovidersList(): Array<AuthProvider>;
  setAuthprovidersList(value: Array<AuthProvider>): void;
  addAuthproviders(value?: AuthProvider, index?: number): AuthProvider;

  getHasmessageoftheday(): boolean;
  setHasmessageoftheday(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ClusterAuthSettings.AsObject;
  static toObject(includeInstance: boolean, msg: ClusterAuthSettings): ClusterAuthSettings.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ClusterAuthSettings, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ClusterAuthSettings;
  static deserializeBinaryFromReader(message: ClusterAuthSettings, reader: jspb.BinaryReader): ClusterAuthSettings;
}

export namespace ClusterAuthSettings {
  export type AsObject = {
    type: string,
    secondfactor: string,
    u2fs?: AuthSettingsU2F.AsObject,
    authprovidersList: Array<AuthProvider.AsObject>,
    hasmessageoftheday: boolean,
  }
}

export class AuthProvider extends jspb.Message {
  getType(): string;
  setType(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDisplay(): string;
  setDisplay(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AuthProvider.AsObject;
  static toObject(includeInstance: boolean, msg: AuthProvider): AuthProvider.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AuthProvider, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AuthProvider;
  static deserializeBinaryFromReader(message: AuthProvider, reader: jspb.BinaryReader): AuthProvider;
}

export namespace AuthProvider {
  export type AsObject = {
    type: string,
    name: string,
    display: string,
  }
}

export class AuthSettingsU2F extends jspb.Message {
  getAppid(): string;
  setAppid(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AuthSettingsU2F.AsObject;
  static toObject(includeInstance: boolean, msg: AuthSettingsU2F): AuthSettingsU2F.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AuthSettingsU2F, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AuthSettingsU2F;
  static deserializeBinaryFromReader(message: AuthSettingsU2F, reader: jspb.BinaryReader): AuthSettingsU2F;
}

export namespace AuthSettingsU2F {
  export type AsObject = {
    appid: string,
  }
}

