package validator

import "fmt"

// StorageValidator validates storage-related requests
type StorageValidator struct {
	MinDiskGB  float64
	MaxDiskGB  float64
	MaxPoolNameLength int
}

// DefaultStorageValidator returns a validator with sensible defaults
func DefaultStorageValidator() *StorageValidator {
	return &StorageValidator{
		MinDiskGB:       1,
		MaxDiskGB:       10240, // 10 TB
		MaxPoolNameLength: 64,
	}
}

// ValidateStoragePoolRequest validates a storage pool request
func (v *StorageValidator) ValidateStoragePoolRequest(name, poolType string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate name
	errs = append(errs, ValidateStoragePoolName(name)...)
	
	if len(name) > v.MaxPoolNameLength {
		errs = appendError(errs, "name", fmt.Sprintf("must be %d characters or less", v.MaxPoolNameLength))
	}
	
	// Validate type
	validTypes := map[string]bool{
		"dir":    true,
		"lvm":    true,
		"nfs":    true,
		"iscsi":  true,
		"zfs":    true,
		"glusterfs": true,
	}
	
	if poolType == "" {
		errs = appendError(errs, "type", "is required")
	} else if !validTypes[poolType] {
		errs = appendError(errs, "type", "must be one of: dir, lvm, nfs, iscsi, zfs, glusterfs")
	}
	
	return errs
}

// ValidateStorageDiskRequest validates a storage disk request
func (v *StorageValidator) ValidateStorageDiskRequest(name string, size float64, format string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate name
	if name == "" {
		errs = appendError(errs, "name", "is required")
	}
	
	if len(name) > 255 {
		errs = appendError(errs, "name", "must be 255 characters or less")
	}
	
	// Check for path traversal
	if pathTraversalPattern.MatchString(name) {
		errs = appendError(errs, "name", "cannot contain path traversal (..)")
	}
	
	// Validate size
	errs = append(errs, ValidateDiskSizeGB(size, v.MinDiskGB, v.MaxDiskGB)...)
	
	// Validate format
	if format != "" {
		validFormats := map[string]bool{
			"qcow2": true,
			"raw":   true,
			"vmdk":  true,
			"vdi":   true,
		}
		if !validFormats[format] {
			errs = appendError(errs, "format", "must be one of: qcow2, raw, vmdk, vdi")
		}
	}
	
	return errs
}

// ValidateDiskResizeRequest validates a disk resize request
func (v *StorageValidator) ValidateDiskResizeRequest(diskID string, newSize float64) ValidationErrors {
	var errs ValidationErrors
	
	// Validate disk ID
	if diskID == "" {
		errs = appendError(errs, "disk_id", "is required")
	}
	
	// Validate new size
	errs = append(errs, ValidateDiskSizeGB(newSize, v.MinDiskGB, v.MaxDiskGB)...)
	
	return errs
}

// ValidateDiskMoveRequest validates a disk move request
func (v *StorageValidator) ValidateDiskMoveRequest(diskID, targetPool string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate disk ID
	if diskID == "" {
		errs = appendError(errs, "disk_id", "is required")
	}
	
	// Validate target pool
	if targetPool == "" {
		errs = appendError(errs, "target_pool", "is required")
	}
	
	if len(targetPool) > 64 {
		errs = appendError(errs, "target_pool", "must be 64 characters or less")
	}
	
	return errs
}

// ValidateISOUploadRequest validates an ISO upload request
func (v *StorageValidator) ValidateISOUploadRequest(name string, size int64) ValidationErrors {
	var errs ValidationErrors
	
	// Validate name
	errs = append(errs, ValidateISOName(name)...)
	
	// Validate size (if provided)
	if size > 0 {
		maxSize := int64(50 * 1024 * 1024 * 1024) // 50 GB default max
		if size > maxSize {
			errs = appendError(errs, "size", "exceeds maximum allowed size (50 GB)")
		}
		
		if size < 1024*1024 { // 1 MB minimum
			errs = appendError(errs, "size", "must be at least 1 MB")
		}
	}
	
	return errs
}

// ValidateISODownloadRequest validates an ISO download request
func (v *StorageValidator) ValidateISODownloadRequest(url, name string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate URL
	if url == "" {
		errs = appendError(errs, "url", "is required")
	}
	
	if len(url) > 2048 {
		errs = appendError(errs, "url", "must be 2048 characters or less")
	}
	
	// Must be HTTP or HTTPS
	if len(url) < 7 || (url[:7] != "http://" && url[:8] != "https://") {
		errs = appendError(errs, "url", "must start with http:// or https://")
	}
	
	// Validate name if provided
	if name != "" {
		errs = append(errs, ValidateISOName(name)...)
	}
	
	return errs
}

// ValidateBackupRequest validates a backup request
func (v *StorageValidator) ValidateBackupRequest(vmID int, name string, storagePool string, retentionDays int) ValidationErrors {
	var errs ValidationErrors
	
	// Validate VM ID
	if vmID <= 0 {
		errs = appendError(errs, "vmid", "must be a positive integer")
	}
	
	// Validate name (optional)
	if name != "" {
		errs = append(errs, ValidateBackupName(name)...)
	}
	
	// Validate storage pool
	if storagePool == "" {
		errs = appendError(errs, "storage_pool", "is required")
	}
	
	if len(storagePool) > 64 {
		errs = appendError(errs, "storage_pool", "must be 64 characters or less")
	}
	
	// Validate retention days (optional)
	if retentionDays != 0 {
		errs = append(errs, ValidateRetentionDays(retentionDays, 1, 3650)...) // 1 day to 10 years
	}
	
	return errs
}

// ValidateBackupRestoreRequest validates a backup restore request
func (v *StorageValidator) ValidateBackupRestoreRequest(backupID string, targetVMID int, startAfter bool) ValidationErrors {
	var errs ValidationErrors
	
	// Validate backup ID
	if backupID == "" {
		errs = appendError(errs, "backup_id", "is required")
	}
	
	// Validate target VM ID
	if targetVMID <= 0 {
		errs = appendError(errs, "target_vmid", "must be a positive integer")
	}
	
	// startAfter is a boolean, no validation needed
	
	return errs
}

// ValidateSnapshotRequest validates a snapshot request
func (v *StorageValidator) ValidateSnapshotRequest(vmID int, name, description string, includeMemory bool) ValidationErrors {
	var errs ValidationErrors
	
	// Validate VM ID
	if vmID <= 0 {
		errs = appendError(errs, "vmid", "must be a positive integer")
	}
	
	// Validate name
	errs = append(errs, ValidateSnapshotName(name)...)
	
	// Validate description (optional)
	if description != "" && len(description) > 1024 {
		errs = appendError(errs, "description", "must be 1024 characters or less")
	}
	
	// includeMemory is a boolean, no validation needed
	
	return errs
}

// ValidateSnapshotRestoreRequest validates a snapshot restore request
func (v *StorageValidator) ValidateSnapshotRestoreRequest(vmID int, snapshotID string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate VM ID
	if vmID <= 0 {
		errs = appendError(errs, "vmid", "must be a positive integer")
	}
	
	// Validate snapshot ID
	if snapshotID == "" {
		errs = appendError(errs, "snapshot_id", "is required")
	}
	
	return errs
}

// ValidateNetworkRequest validates a network request
func (v *StorageValidator) ValidateNetworkRequest(name, networkType string, cidr string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate name
	if name == "" {
		errs = appendError(errs, "name", "is required")
	}
	
	if len(name) > 64 {
		errs = appendError(errs, "name", "must be 64 characters or less")
	}
	
	if !alphanumericPattern.MatchString(name) {
		errs = appendError(errs, "name", "contains invalid characters")
	}
	
	// Validate type
	validTypes := map[string]bool{
		"nat":     true,
		"bridge":  true,
		"isolated": true,
	}
	
	if networkType == "" {
		errs = appendError(errs, "type", "is required")
	} else if !validTypes[networkType] {
		errs = appendError(errs, "type", "must be one of: nat, bridge, isolated")
	}
	
	// Validate CIDR if provided
	if cidr != "" {
		errs = append(errs, validateCIDR(cidr)...)
	}
	
	return errs
}

// validateCIDR validates a CIDR notation
func validateCIDR(cidr string) ValidationErrors {
	var errs ValidationErrors
	
	if cidr == "" {
		return errs
	}
	
	// Basic format check: should contain /
	found := false
	for _, r := range cidr {
		if r == '/' {
			found = true
			break
		}
	}
	
	if !found {
		errs = appendError(errs, "cidr", "must be in CIDR notation (e.g., 192.168.1.0/24)")
		return errs
	}
	
	// Split IP and prefix
	ip, prefix, ok := splitCIDR(cidr)
	if !ok {
		errs = appendError(errs, "cidr", "invalid format")
		return errs
	}
	
	// Validate IP parts
	ipParts := splitIP(ip)
	if len(ipParts) != 4 {
		errs = appendError(errs, "cidr", "IP must have 4 octets")
		return errs
	}
	
	for i, part := range ipParts {
		if part == "" {
			errs = appendError(errs, "cidr", fmt.Sprintf("octet %d is empty", i+1))
			continue
		}
		
		val := 0
		for _, r := range part {
			if r < '0' || r > '9' {
				errs = appendError(errs, "cidr", fmt.Sprintf("octet %d contains non-digit characters", i+1))
				break
			}
			val = val*10 + int(r-'0')
		}
		
		if val > 255 {
			errs = appendError(errs, "cidr", fmt.Sprintf("octet %d must be 0-255", i+1))
		}
	}
	
	// Validate prefix
	if prefix < 0 || prefix > 32 {
		errs = appendError(errs, "cidr", "prefix must be 0-32")
	}
	
	return errs
}

// splitCIDR splits a CIDR string into IP and prefix
func splitCIDR(cidr string) (string, int, bool) {
	for i, r := range cidr {
		if r == '/' {
			ip := cidr[:i]
			prefixStr := cidr[i+1:]
			
			prefix := 0
			for _, r := range prefixStr {
				if r < '0' || r > '9' {
					return "", 0, false
				}
				prefix = prefix*10 + int(r-'0')
			}
			
			return ip, prefix, true
		}
	}
	
	return "", 0, false
}

// splitIP splits an IP address into octets
func splitIP(ip string) []string {
	var parts []string
	current := ""
	
	for _, r := range ip {
		if r == '.' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	
	if current != "" {
		parts = append(parts, current)
	}
	
	return parts
}

// ValidateFirewallRuleRequest validates a firewall rule request
func (v *StorageValidator) ValidateFirewallRuleRequest(direction, action, protocol string, port string, source string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate direction
	validDirections := map[string]bool{
		"inbound":  true,
		"outbound": true,
	}
	
	if direction == "" {
		errs = appendError(errs, "direction", "is required")
	} else if !validDirections[direction] {
		errs = appendError(errs, "direction", "must be one of: inbound, outbound")
	}
	
	// Validate action
	validActions := map[string]bool{
		"accept": true,
		"drop":   true,
		"reject": true,
	}
	
	if action == "" {
		errs = appendError(errs, "action", "is required")
	} else if !validActions[action] {
		errs = appendError(errs, "action", "must be one of: accept, drop, reject")
	}
	
	// Validate protocol
	validProtocols := map[string]bool{
		"tcp":  true,
		"udp":  true,
		"icmp": true,
		"all":  true,
	}
	
	if protocol == "" {
		errs = appendError(errs, "protocol", "is required")
	} else if !validProtocols[protocol] {
		errs = appendError(errs, "protocol", "must be one of: tcp, udp, icmp, all")
	}
	
	// Validate port if TCP/UDP
	if protocol == "tcp" || protocol == "udp" {
		if port != "" {
			if !isValidPortRange(port) {
				errs = appendError(errs, "port", "must be a valid port or port range (e.g., 80 or 8000-9000)")
			}
		}
	}
	
	// Validate source (optional)
	if source != "" {
		// Could be IP, CIDR, or firewall group ID
		if len(source) > 255 {
			errs = appendError(errs, "source", "must be 255 characters or less")
		}
	}
	
	return errs
}

// isValidPortRange validates a port or port range
func isValidPortRange(port string) bool {
	// Single port
	if isValidPort(port) {
		return true
	}
	
	// Port range
	for i, r := range port {
		if r == '-' {
			start := port[:i]
			end := port[i+1:]
			
			if !isValidPort(start) || !isValidPort(end) {
				return false
			}
			
			// Parse and compare
			startVal := 0
			for _, r := range start {
				startVal = startVal*10 + int(r-'0')
			}
			
			endVal := 0
			for _, r := range end {
				endVal = endVal*10 + int(r-'0')
			}
			
			return startVal <= endVal
		}
	}
	
	return false
}
