# Homepage Dashboard Integration

Complete guide for integrating Lab with [Homepage](https://gethomepage.dev/) dashboard using the Custom API widget.

## Prerequisites

1. **Lab API accessible** - Ensure your Lab instance is running and accessible
2. **API Key** - Create an API key in Lab Settings → API Keys
3. **Homepage installed** - Follow [Homepage installation guide](https://gethomepage.dev/getting-started/installation/)

---

## Quick Start

### Step 1: Create Lab API Key

1. Log into Lab web interface
2. Navigate to **Settings** → **API Keys**
3. Click **Create API Key**
4. Name it "Homepage Dashboard"
5. Copy the generated key (starts with `labkey_`)

### Step 2: Add to Homepage Configuration

Add the following to your Homepage `services.yaml`:

```yaml
- Lab:
    - Lab Dashboard:
        href: http://lab.local:8080
        description: Virtualization management dashboard
        icon: server.svg
        widget:
          type: customapi
          url: http://lab.local:8080/api/status/system
          refreshInterval: 10000
          mappings:
            - field: cpu_usage
              label: CPU
              format: percent
            - field: memory_usage
              label: Memory
              format: percent
            - field: disk_usage
              label: Disk
              format: percent
```

---

## Widget Configurations

### System Metrics Widget

Displays real-time CPU, memory, and disk usage.

```yaml
- Lab System:
    href: http://lab.local:8080
    widget:
      type: customapi
      url: http://lab.local:8080/api/status/system
      refreshInterval: 10000
      mappings:
        - field: cpu_usage
          label: CPU
          format: percent
        - field: memory_usage
          label: Memory
          format: percent
        - field: disk_usage
          label: Disk
          format: percent
        - field: uptime_seconds
          label: Uptime
          format: duration
```

**Preview:**
```
┌─────────────────────────┐
│ Lab System              │
├─────────────────────────┤
│ CPU     23.5%          │
│ Memory  67.2%          │
│ Disk    45.0%          │
│ Uptime  1d 0h 0m       │
└─────────────────────────┘
```

---

### VM Status List Widget

Shows all VMs with their current state (requires API key).

```yaml
- Lab VMs:
    href: http://lab.local:8080
    widget:
      type: customapi
      url: http://lab.local:8080/api/status/vms
      headers:
        Authorization: labkey_YOUR_API_KEY_HERE
      refreshInterval: 15000
      display: dynamic-list
      mappings:
        items: items
        name: name
        label: state
        limit: 10
```

**Preview:**
```
┌─────────────────────────┐
│ Lab VMs                 │
├─────────────────────────┤
│ plex-server    running │
│ home-assistant stopped │
│ proxmox-backup running │
└─────────────────────────┘
```

---

### Service Health Widget

Monitors core Lab services (libvirt, API).

```yaml
- Lab Services:
    href: http://lab.local:8080
    widget:
      type: customapi
      url: http://lab.local:8080/api/status/services
      refreshInterval: 30000
      mappings:
        - field: ok
          label: Services OK
          format: text
        - field: total
          label: Total
          format: text
```

**Advanced - Individual Service Status:**

```yaml
- Lab Services:
    href: http://lab.local:8080
    widget:
      type: customapi
      url: http://lab.local:8080/api/status/services
      refreshInterval: 30000
      display: dynamic-list
      mappings:
        items: items
        name: name
        label: status
        additionalField:
          field: message
          color: theme
```

**Preview:**
```
┌─────────────────────────┐
│ Lab Services            │
├─────────────────────────┤
│ libvirt        ok       │
│ api            ok       │
└─────────────────────────┘
```

---

### Storage Status Widget

Displays storage pool usage.

```yaml
- Lab Storage:
    href: http://lab.local:8080
    widget:
      type: customapi
      url: http://lab.local:8080/api/status/storage
      refreshInterval: 60000
      display: dynamic-list
      mappings:
        items: items
        name: name
        label: usage
        format: percent
        additionalField:
          field: state
          color: theme
```

**Preview:**
```
┌─────────────────────────┐
│ Lab Storage             │
├─────────────────────────┤
│ default       45.0%  ok │
│ backup-pool   12.3%  ok │
└─────────────────────────┘
```

---

### Network Status Widget

Shows virtual network status.

```yaml
- Lab Networks:
    href: http://lab.local:8080
    widget:
      type: customapi
      url: http://lab.local:8080/api/status/networks
      refreshInterval: 30000
      display: dynamic-list
      mappings:
        items: items
        name: name
        label: status
        additionalField:
          field: status
          color: theme  # green for active, red for inactive
```

**Preview:**
```
┌─────────────────────────┐
│ Lab Networks            │
├─────────────────────────┤
│ default       active    │
│ isolated      inactive  │
└─────────────────────────┘
```

---

### Recent Alerts Widget

Displays recent alerts and events (requires API key).

```yaml
- Lab Alerts:
    href: http://lab.local:8080
    widget:
      type: customapi
      url: http://lab.local:8080/api/status/alerts?limit=5
      headers:
        Authorization: labkey_YOUR_API_KEY_HERE
      refreshInterval: 30000
      display: dynamic-list
      mappings:
        items: items
        name: message
        label: level
        additionalField:
          field: timestamp
          format: relativeDate
```

**Preview:**
```
┌─────────────────────────────────┐
│ Lab Alerts                      │
├─────────────────────────────────┤
│ High CPU usage    warning  5m   │
│ Backup completed  info    1h    │
│ VM stopped        info    2h    │
└─────────────────────────────────┘
```

---

## Complete Services.yaml Example

Here's a complete example with all Lab widgets:

```yaml
- Lab:
    - System Metrics:
        href: http://lab.local:8080
        icon: server.svg
        widget:
          type: customapi
          url: http://lab.local:8080/api/status/system
          refreshInterval: 10000
          mappings:
            - field: cpu_usage
              label: CPU
              format: percent
            - field: memory_usage
              label: Memory
              format: percent
            - field: disk_usage
              label: Disk
              format: percent

    - VM Status:
        href: http://lab.local:8080
        icon: virtual-machine.svg
        widget:
          type: customapi
          url: http://lab.local:8080/api/status/vms
          headers:
            Authorization: labkey_YOUR_API_KEY_HERE
          refreshInterval: 15000
          display: dynamic-list
          mappings:
            items: items
            name: name
            label: state
            limit: 5

    - Storage:
        href: http://lab.local:8080
        icon: database.svg
        widget:
          type: customapi
          url: http://lab.local:8080/api/status/storage
          refreshInterval: 60000
          display: dynamic-list
          mappings:
            items: items
            name: name
            label: usage
            format: percent

    - Services:
        href: http://lab.local:8080
        icon: heart-pulse.svg
        widget:
          type: customapi
          url: http://lab.local:8080/api/status/services
          refreshInterval: 30000
          display: dynamic-list
          mappings:
            items: items
            name: name
            label: status
```

---

## Advanced Configurations

### Using Environment Variables for API Keys

Store your API key in an environment variable and reference it:

```yaml
# docker-compose.yml or .env file
LAB_API_KEY: labkey_YOUR_API_KEY_HERE

# services.yaml (using env var substitution)
- Lab VMs:
    widget:
      type: customapi
      url: http://lab.local:8080/api/status/vms
      headers:
        Authorization: ${LAB_API_KEY}
```

### Custom Refresh Intervals

Adjust refresh intervals based on data criticality:

```yaml
# System metrics - frequent updates (10s)
refreshInterval: 10000

# VM status - moderate updates (15-30s)
refreshInterval: 15000

# Storage - less frequent (1-5m)
refreshInterval: 60000

# Alerts - moderate updates (30s)
refreshInterval: 30000
```

### Conditional Formatting

Use Homepage's color options for visual status indicators:

```yaml
- Lab Services:
    widget:
      type: customapi
      url: http://lab.local:8080/api/status/services
      display: dynamic-list
      mappings:
        items: items
        name: name
        label: status
        additionalField:
          field: status
          color: adaptive  # green for ok, red for error
```

---

## Troubleshooting

### Widget Shows "Failed to Fetch"

1. **Check network connectivity** - Ensure Homepage can reach Lab API
2. **Verify URL** - Confirm the API URL is correct
3. **Check CORS** - Lab API should allow requests from Homepage origin

### Widget Shows "401 Unauthorized"

1. **API key missing** - Add `Authorization` header with your API key
2. **Invalid API key** - Verify the key is correct and not expired
3. **Key format** - Use `labkey <key>` or `Bearer <key>` format

### Widget Shows Empty Data

1. **No data available** - Create VMs, storage pools, etc.
2. **Metrics not collected** - Ensure metrics collector is running
3. **libvirt not connected** - Check libvirt service status

### Widget Not Updating

1. **Check refreshInterval** - Ensure it's set (default 10s)
2. **Browser cache** - Hard refresh the Homepage dashboard
3. **Homepage logs** - Check for errors in Homepage logs

---

## Security Best Practices

1. **Use HTTPS** - Always use HTTPS in production:
   ```yaml
   url: https://lab.example.com/api/status/system
   ```

2. **Dedicated API Keys** - Create separate API keys for each dashboard

3. **Limited Permissions** - API keys for dashboards should only have read permissions

4. **Set Expiration** - Consider setting expiration dates on API keys

5. **Network Isolation** - Keep dashboard traffic on internal network

---

## Supported Homepage Features

| Feature | Support | Notes |
|---------|---------|-------|
| Basic mappings | ✅ | All field types supported |
| Dynamic lists | ✅ | Perfect for VM/container lists |
| Custom headers | ✅ | For API key authentication |
| Refresh intervals | ✅ | Configurable per widget |
| Format types | ✅ | percent, number, duration, date, etc. |
| Conditional colors | ✅ | adaptive, theme, black, white |

---

## API Reference

For complete API documentation, see [STATUS_API.md](STATUS_API.md).

### Quick Reference

| Endpoint | Auth | Description |
|----------|------|-------------|
| `/api/status/system` | ❌ | CPU, memory, disk, uptime |
| `/api/status/vms` | ✅ | VM list and status |
| `/api/status/containers` | ✅ | Container list and status |
| `/api/status/storage` | ❌ | Storage pools |
| `/api/status/networks` | ❌ | Virtual networks |
| `/api/status/services` | ❌ | Service health |
| `/api/status/alerts` | ✅ | Recent alerts |

✅ = Requires API key  
❌ = Public (no auth required)

---

## Examples Gallery

### Minimal Setup

Just system metrics (no API key needed):

```yaml
- Lab:
    - Status:
        href: http://lab.local:8080
        widget:
          type: customapi
          url: http://lab.local:8080/api/status/system
          mappings:
            - field: cpu_usage
              format: percent
            - field: memory_usage
              format: percent
```

### Full Monitoring Dashboard

All widgets with API key authentication for complete visibility.

See "Complete Services.yaml Example" above.

---

**Last updated:** March 9, 2026
