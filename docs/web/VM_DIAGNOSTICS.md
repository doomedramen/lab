# VM Diagnostics Guide

This guide explains the VM diagnostics features and known limitations.

## Diagnostics Panel

The Diagnostics panel (available on each VM's detail page) provides comprehensive information about the VM's configuration and runtime state.

### Available Information

#### Domain Info

- **Domain ID**: Libvirt's internal domain ID
- **UUID**: Unique identifier for the VM
- **State**: Current runtime state (running, stopped, paused, etc.)
- **Memory**: Current and maximum memory allocation
- **CPUs**: Number of virtual CPUs
- **Persistent**: Whether the VM configuration persists across reboots
- **Autostart**: Whether the VM starts automatically

#### Network Interfaces

- Interface name (e.g., vnet0)
- MAC address
- IP address and protocol (IPv4/IPv6)
- Network prefix

**Note:** IP addresses are only visible when:

- The VM uses bridge networking (not user-mode/NAT)
- QEMU guest agent is installed and running in the guest OS

#### Storage

- Target device name (e.g., vda)
- Source file path
- Driver type (qcow2, raw)
- Bus type (virtio, ide, etc.)

#### Console Devices

- Serial console device path
- Console device path
- Guest agent channel status
- VNC server address and port

#### Host Information

- Hostname
- Architecture
- Libvirt URI and version

## Known Limitations

### 1. No IP Address Display

**Symptom:** The IP address field shows empty even when the VM is running.

**Cause:** The VM uses user-mode networking (NAT), which doesn't report guest IPs to the host.

**Solutions:**

1. **Use Bridge Networking** (recommended for production):

   ```xml
   <interface type='bridge'>
     <source bridge='br0'/>
     <model type='virtio'/>
   </interface>
   ```

2. **Install QEMU Guest Agent** in the guest OS:
   - Alpine: `apk add qemu-guest-agent && rc-update add qemu-guest-agent`
   - Ubuntu/Debian: `apt install qemu-guest-agent`
   - The guest agent reports IP addresses to the host

### 2. Shutdown Button Doesn't Work

**Symptom:** Clicking "Shutdown" has no effect, VM continues running.

**Cause:** The guest OS doesn't respond to ACPI shutdown signals.

**Solutions:**

1. **Use Force Stop** - The "Force Stop" button sends a hard power-off signal

2. **Configure ACPI in Guest OS**:
   - Ensure ACPI is enabled in the kernel
   - Install and configure acpid or systemd-logind
   - For systemd: `systemctl enable systemd-logind`

3. **Check VM Configuration** - Ensure ACPI is enabled:
   ```xml
   <features>
     <acpi/>
     <apic/>
   </features>
   ```

### 3. Serial Console Shows No Output

**Symptom:** Serial console connects but shows blank screen or no boot messages.

**Cause:** The guest OS bootloader and kernel are not configured for serial output.

**Solutions by OS:**

#### Alpine Linux

1. Edit `/etc/update-extlinux.conf`:
   ```
   default_kernel_opts="... console=ttyAMA0,115200"
   ```
2. Run `update-extlinux`
3. Reboot

#### Ubuntu/Debian

1. Edit `/etc/default/grub`:
   ```
   GRUB_CMDLINE_LINUX="console=ttyAMA0,115200 console=hvc0"
   ```
2. Update GRUB: `update-grub`
3. Reboot

#### Generic Linux

Add to kernel command line:

```
console=ttyAMA0,115200 console=hvc0
```

#### Windows

Windows doesn't support serial console by default. Use VNC instead.

### 4. VNC Console Connection Fails

**Symptom:** VNC console shows "Connection closed" error.

**Possible Causes:**

1. VM is not running
2. VNC is not enabled in VM configuration
3. API server can't reach VNC port

**Troubleshooting:**

1. Check VM state in Diagnostics panel
2. Verify VNC port is shown in Diagnostics > Console tab
3. Check if VNC server is listening: `virsh qemu-monitor-command vm-100 --hmp 'info vnc'`

## Best Practices

### For Production VMs

1. **Use Bridge Networking** - Provides proper network visibility and performance
2. **Install QEMU Guest Agent** - Enables IP reporting and graceful shutdown
3. **Configure Serial Console** - Essential for headless debugging
4. **Enable ACPI** - Allows proper shutdown handling

### For Development/Testing VMs

1. **User-mode Networking** is fine for isolated testing
2. **VNC Console** is usually sufficient for interaction
3. **Force Stop** is acceptable for non-production workloads

## Diagnostic Commands

For advanced troubleshooting, these CLI commands provide the same information as the Diagnostics panel:

```bash
# Basic VM info
virsh dominfo vm-100

# Full XML configuration
virsh dumpxml vm-100

# Network interfaces (requires guest agent)
virsh domifaddr vm-100

# VNC status
virsh qemu-monitor-command vm-100 --hmp 'info vnc'

# Serial console devices
virsh qemu-monitor-command vm-100 --hmp 'info chardev'

# Disk information
virsh domblklist vm-100
virsh domblkinfo vm-100 vda
```

## API Endpoints

The diagnostics data is available via the Connect RPC API:

```protobuf
rpc GetVMDiagnostics(GetVMDiagnosticsRequest) returns (GetVMDiagnosticsResponse);
```

Request:

```json
{ "vmid": 100 }
```

Response includes all diagnostic information shown in the panel.
