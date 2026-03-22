# Status API Reference

Generic REST API endpoints for monitoring dashboards and automation tools.

## Overview

The Status API provides read-only access to system metrics, infrastructure status, and alerts. Designed to be consumed by monitoring dashboards like Homepage, Grafana, or custom tools.

## Authentication

### Public Endpoints

The following endpoints do **not** require authentication:

- `/api/status/system` - System metrics (CPU, memory, disk)
- `/api/status/storage` - Storage pool status
- `/api/status/networks` - Virtual network status
- `/api/status/services` - Core service health

### Protected Endpoints

The following endpoints require **API key authentication**:

- `/api/status/vms` - VM list and status
- `/api/status/containers` - Container list and status
- `/api/status/alerts` - Recent alerts/events

### API Key Authentication

Include the API key in the request header using one of these formats:

```bash
# Format 1: Authorization header with labkey prefix
Authorization: labkey your_api_key_here

# Format 2: Authorization header with Bearer prefix
Authorization: Bearer your_api_key_here

# Format 3: X-API-Key header
X-API-Key: your_api_key_here
```

### Creating an API Key

1. Log into the Lab web interface
2. Navigate to **Settings** → **API Keys**
3. Click **Create API Key**
4. Provide a name (e.g., "Homepage Dashboard")
5. Optionally set permissions and expiration
6. **Copy the generated key** - it won't be shown again

Example API key: `labkey_xK9mN2pL5qR8sT1uW4yZ7aB3cD6eF0gH`

---

## Endpoints

### GET /api/status/system

Returns basic system metrics.

**Authentication:** Not required (public)

**Response:**

```json
{
  "cpu_usage": 23.5,
  "memory_usage": 67.2,
  "disk_usage": 45.0,
  "uptime_seconds": 86400,
  "updated_at": "2026-03-09T12:00:00Z"
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `cpu_usage` | float | CPU usage percentage (0-100) |
| `memory_usage` | float | Memory usage percentage (0-100) |
| `disk_usage` | float | Disk usage percentage (0-100) |
| `uptime_seconds` | int | System uptime in seconds |
| `updated_at` | string | ISO 8601 timestamp |

---

### GET /api/status/vms

Returns list of virtual machines with their current state.

**Authentication:** Required (API key)

**Response:**

```json
{
  "items": [
    {
      "id": "vm-1",
      "name": "plex-server",
      "state": "running"
    },
    {
      "id": "vm-2",
      "name": "home-assistant",
      "state": "stopped"
    }
  ],
  "total": 2,
  "running": 1,
  "stopped": 1
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `items` | array | List of VMs |
| `items[].id` | string | VM ID |
| `items[].name` | string | VM name |
| `items[].state` | string | VM state (`running`, `stopped`, `paused`, etc.) |
| `total` | int | Total VM count |
| `running` | int | Running VM count |
| `stopped` | int | Stopped VM count |

---

### GET /api/status/containers

Returns list of containers with their current state.

**Authentication:** Required (API key)

**Response:**

```json
{
  "items": [
    {
      "id": "container-1",
      "name": "nginx-proxy",
      "state": "running"
    }
  ],
  "total": 1,
  "running": 1,
  "stopped": 0
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `items` | array | List of containers |
| `items[].id` | string | Container ID |
| `items[].name` | string | Container name |
| `items[].state` | string | Container state |
| `total` | int | Total container count |
| `running` | int | Running container count |
| `stopped` | int | Stopped container count |

---

### GET /api/status/storage

Returns storage pool information.

**Authentication:** Not required (public)

**Response:**

```json
{
  "items": [
    {
      "name": "default",
      "state": "active",
      "capacity": 1000000000000,
      "allocated": 450000000000,
      "available": 550000000000,
      "usage": 45.0
    }
  ],
  "total": 1
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `items` | array | List of storage pools |
| `items[].name` | string | Pool name |
| `items[].state` | string | Pool state (`active`, `inactive`, etc.) |
| `items[].capacity` | int | Total capacity in bytes |
| `items[].allocated` | int | Allocated space in bytes |
| `items[].available` | int | Available space in bytes |
| `items[].usage` | float | Usage percentage (0-100) |
| `total` | int | Total storage pool count |

---

### GET /api/status/networks

Returns virtual network information.

**Authentication:** Not required (public)

**Response:**

```json
{
  "items": [
    {
      "name": "default",
      "status": "active"
    },
    {
      "name": "isolated",
      "status": "inactive"
    }
  ],
  "total": 2
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `items` | array | List of networks |
| `items[].name` | string | Network name |
| `items[].status` | string | Network status (`active`, `inactive`) |
| `total` | int | Total network count |

---

### GET /api/status/services

Returns core service health status.

**Authentication:** Not required (public)

**Response:**

```json
{
  "items": [
    {
      "name": "libvirt",
      "status": "ok"
    },
    {
      "name": "api",
      "status": "ok"
    }
  ],
  "total": 2,
  "ok": 2
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `items` | array | List of services |
| `items[].name` | string | Service name |
| `items[].status` | string | Service status (`ok`, `error`, `disabled`) |
| `items[].message` | string | Optional status message |
| `total` | int | Total service count |
| `ok` | int | Healthy service count |

---

### GET /api/status/alerts

Returns recent alerts/events.

**Authentication:** Required (API key)

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 10 | Maximum number of alerts to return |

**Response:**

```json
{
  "items": [
    {
      "id": "alert-1",
      "level": "warning",
      "message": "High CPU usage detected",
      "timestamp": "2026-03-09T11:30:00Z",
      "source": "monitor"
    }
  ],
  "total": 1
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `items` | array | List of alerts |
| `items[].id` | string | Alert ID |
| `items[].level` | string | Alert severity (`info`, `warning`, `error`, `critical`) |
| `items[].message` | string | Alert message |
| `items[].timestamp` | string | ISO 8601 timestamp |
| `items[].source` | string | Alert source |
| `total` | int | Total alert count |

---

## Error Responses

### 401 Unauthorized

Returned when API key is missing or invalid for protected endpoints.

```json
{
  "error": "missing API key"
}
```

or

```json
{
  "error": "invalid API key"
}
```

### 500 Internal Server Error

Returned when an internal error occurs.

```json
{
  "error": "error message details"
}
```

---

## Example Requests

### cURL Examples

```bash
# Public endpoint (no auth)
curl http://localhost:8080/api/status/system

# Protected endpoint (with API key)
curl -H "Authorization: labkey_xK9mN2pL5qR8sT1uW4yZ7aB3cD6eF0gH" \
  http://localhost:8080/api/status/vms

# With X-API-Key header
curl -H "X-API-Key: xK9mN2pL5qR8sT1uW4yZ7aB3cD6eF0gH" \
  http://localhost:8080/api/status/alerts?limit=5
```

### JavaScript/TypeScript Example

```typescript
async function getVMStatus(apiKey: string) {
  const response = await fetch('http://localhost:8080/api/status/vms', {
    headers: {
      'Authorization': `labkey ${apiKey}`
    }
  });
  
  if (!response.ok) {
    throw new Error(`HTTP error: ${response.status}`);
  }
  
  return await response.json();
}
```

### Python Example

```python
import requests

def get_vm_status(api_key: str):
    headers = {
        'Authorization': f'labkey {api_key}'
    }
    response = requests.get(
        'http://localhost:8080/api/status/vms',
        headers=headers
    )
    response.raise_for_status()
    return response.json()
```

---

## Security Considerations

1. **Use HTTPS in production** - API keys should never be transmitted over unencrypted connections.

2. **Scope API keys appropriately** - Create dedicated API keys for each dashboard/tool.

3. **Set expiration dates** - API keys can have expiration dates for added security.

4. **Monitor API key usage** - All API key usage is logged in the audit log.

5. **Revoke compromised keys** - Immediately revoke any API key that may have been exposed.

---

## Integration Examples

- **[Homepage Dashboard](HOMEPAGE_EXAMPLES.md)** - Complete Homepage widget configurations
- Grafana (coming soon)
- Custom dashboards (use the examples above)

---

## Troubleshooting

### "missing API key" error

Ensure you're including the API key in the request header using one of the supported formats.

### "invalid API key" error

- Check that the API key is correct (no typos)
- Verify the API key hasn't expired
- Ensure the API key hasn't been revoked

### Empty data in responses

- Some endpoints return empty arrays if no data is available (e.g., no VMs created yet)
- Check that libvirt is running and accessible
- Verify the metrics collector is running for system metrics

---

**Last updated:** March 9, 2026
