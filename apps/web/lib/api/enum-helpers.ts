import { VmStatus } from "../gen/lab/v1/vm_pb";
import { NodeStatus } from "../gen/lab/v1/node_pb";
import { ContainerStatus } from "../gen/lab/v1/container_pb";
import { StackStatus } from "../gen/lab/v1/stack_pb";
import {
  BiosType,
  MachineType,
  NetworkModel,
  NetworkType,
  OsType,
} from "../gen/lab/v1/common_pb";
import { DiskBus, DiskFormat } from "../gen/lab/v1/storage_pb";
import { ProxySSLMode } from "../gen/lab/v1/proxy_pb";

export function vmStatusToString(status: VmStatus): string {
  switch (status) {
    case VmStatus.RUNNING:
      return "running";
    case VmStatus.STOPPED:
      return "stopped";
    case VmStatus.PAUSED:
      return "paused";
    case VmStatus.SUSPENDED:
      return "suspended";
    default:
      return "stopped";
  }
}

export function nodeStatusToString(status: NodeStatus): string {
  switch (status) {
    case NodeStatus.ONLINE:
      return "online";
    case NodeStatus.OFFLINE:
      return "offline";
    case NodeStatus.MAINTENANCE:
      return "maintenance";
    default:
      return "offline";
  }
}

export function containerStatusToString(status: ContainerStatus): string {
  switch (status) {
    case ContainerStatus.RUNNING:
      return "running";
    case ContainerStatus.STOPPED:
      return "stopped";
    case ContainerStatus.FROZEN:
      return "frozen";
    default:
      return "stopped";
  }
}

export function stackStatusToString(status: StackStatus): string {
  switch (status) {
    case StackStatus.RUNNING:
      return "running";
    case StackStatus.PARTIALLY_RUNNING:
      return "partially_running";
    case StackStatus.STOPPED:
      return "stopped";
    default:
      return "stopped";
  }
}

export function osTypeToString(type: OsType): string {
  switch (type) {
    case OsType.LINUX:
      return "linux";
    case OsType.WINDOWS:
      return "windows";
    case OsType.SOLARIS:
      return "solaris";
    default:
      return "other";
  }
}

export function machineTypeToString(type: MachineType): string {
  switch (type) {
    case MachineType.PC:
      return "pc";
    case MachineType.Q35:
      return "q35";
    case MachineType.VIRT:
      return "virt";
    default:
      return "pc";
  }
}

export function biosTypeToString(type: BiosType): string {
  switch (type) {
    case BiosType.SEABIOS:
      return "seabios";
    case BiosType.OVMF:
      return "ovmf";
    default:
      return "seabios";
  }
}

export function networkTypeToString(type: NetworkType): string {
  switch (type) {
    case NetworkType.BRIDGE:
      return "bridge";
    default:
      return "user";
  }
}

export function networkModelToString(model: NetworkModel): string {
  switch (model) {
    case NetworkModel.VIRTIO:
      return "virtio";
    case NetworkModel.E1000:
      return "e1000";
    case NetworkModel.RTL8139:
      return "rtl8139";
    default:
      return "virtio";
  }
}

export function osTypeFromString(s: string): OsType {
  switch (s) {
    case "linux":
      return OsType.LINUX;
    case "windows":
      return OsType.WINDOWS;
    case "solaris":
      return OsType.SOLARIS;
    default:
      return OsType.OTHER;
  }
}

export function networkTypeFromString(s: string): NetworkType {
  return s === "bridge" ? NetworkType.BRIDGE : NetworkType.USER;
}

export function networkModelFromString(s: string): NetworkModel {
  switch (s) {
    case "e1000":
      return NetworkModel.E1000;
    case "rtl8139":
      return NetworkModel.RTL8139;
    default:
      return NetworkModel.VIRTIO;
  }
}

export function diskBusToString(bus: DiskBus): string {
  switch (bus) {
    case DiskBus.VIRTIO:
      return "virtio";
    case DiskBus.SATA:
      return "sata";
    case DiskBus.SCSI:
      return "scsi";
    case DiskBus.IDE:
      return "ide";
    case DiskBus.USB:
      return "usb";
    case DiskBus.NVME:
      return "nvme";
    default:
      return "virtio";
  }
}

export function diskBusFromString(s: string): DiskBus {
  switch (s) {
    case "sata":
      return DiskBus.SATA;
    case "scsi":
      return DiskBus.SCSI;
    case "ide":
      return DiskBus.IDE;
    case "usb":
      return DiskBus.USB;
    case "nvme":
      return DiskBus.NVME;
    default:
      return DiskBus.VIRTIO;
  }
}

export function diskFormatToString(format: DiskFormat): string {
  switch (format) {
    case DiskFormat.QCOW2:
      return "qcow2";
    case DiskFormat.RAW:
      return "raw";
    case DiskFormat.VMDK:
      return "vmdk";
    case DiskFormat.VDI:
      return "vdi";
    case DiskFormat.VHDX:
      return "vhdx";
    default:
      return "qcow2";
  }
}

export function diskFormatFromString(s: string): DiskFormat {
  switch (s) {
    case "raw":
      return DiskFormat.RAW;
    case "vmdk":
      return DiskFormat.VMDK;
    case "vdi":
      return DiskFormat.VDI;
    case "vhdx":
      return DiskFormat.VHDX;
    default:
      return DiskFormat.QCOW2;
  }
}

export function proxySSLModeToString(mode: ProxySSLMode): string {
  switch (mode) {
    case ProxySSLMode.PROXY_SSL_MODE_NONE:
      return "None";
    case ProxySSLMode.PROXY_SSL_MODE_SELF_SIGNED:
      return "Self-Signed";
    case ProxySSLMode.PROXY_SSL_MODE_ACME:
      return "ACME (Let's Encrypt)";
    case ProxySSLMode.PROXY_SSL_MODE_CUSTOM:
      return "Custom";
    default:
      return "None";
  }
}
