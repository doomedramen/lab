package validator

import (
	"fmt"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// VMCreateRequestValidator validates VM creation requests
type VMCreateRequestValidator struct {
	MinMemoryGB  float64
	MaxMemoryGB  float64
	MinCPUCores  int
	MaxCPUCores  int
	MinDiskGB    float64
	MaxDiskGB    float64
	MaxTags      int
	MaxNetworks  int
}

// DefaultVMCreateRequestValidator returns a validator with sensible defaults
func DefaultVMCreateRequestValidator() *VMCreateRequestValidator {
	return &VMCreateRequestValidator{
		MinMemoryGB:  0.5,
		MaxMemoryGB:  1024,
		MinCPUCores:  1,
		MaxCPUCores:  128,
		MinDiskGB:    1,
		MaxDiskGB:    10240, // 10 TB
		MaxTags:      20,
		MaxNetworks:  8,
	}
}

// Validate validates a VM create request
func (v *VMCreateRequestValidator) Validate(name string, memory float64, cpuCores int, diskSize float64, os model.OSConfig, network []model.NetworkConfig, tags []string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate name
	errs = append(errs, ValidateVMName(name)...)
	
	// Validate memory
	errs = append(errs, ValidateMemoryGB(memory, v.MinMemoryGB, v.MaxMemoryGB)...)
	
	// Validate CPU cores
	errs = append(errs, ValidateCPUCores(cpuCores, v.MinCPUCores, v.MaxCPUCores)...)
	
	// Validate disk size
	errs = append(errs, ValidateDiskSizeGB(diskSize, v.MinDiskGB, v.MaxDiskGB)...)
	
	// Validate OS
	errs = append(errs, validateOSConfig(os)...)
	
	// Validate network configuration
	errs = append(errs, v.validateNetworkConfig(network)...)
	
	// Validate tags
	errs = append(errs, ValidateTags(tags)...)
	
	return errs
}

// VMUpdateRequestValidator validates VM update requests
type VMUpdateRequestValidator struct {
	MinMemoryGB float64
	MaxMemoryGB float64
	MinCPUCores int
	MaxCPUCores int
}

// DefaultVMUpdateRequestValidator returns a validator with sensible defaults
func DefaultVMUpdateRequestValidator() *VMUpdateRequestValidator {
	return &VMUpdateRequestValidator{
		MinMemoryGB: 0.5,
		MaxMemoryGB: 1024,
		MinCPUCores: 1,
		MaxCPUCores: 128,
	}
}

// Validate validates a VM update request
func (v *VMUpdateRequestValidator) Validate(name *string, description *string, memory *float64, cpuCores *int, tags []string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate optional name
	if name != nil {
		errs = append(errs, ValidateVMName(*name)...)
	}
	
	// Validate optional description
	if description != nil {
		errs = append(errs, ValidateVMDescription(*description)...)
	}
	
	// Validate optional memory
	if memory != nil {
		errs = append(errs, ValidateMemoryGB(*memory, v.MinMemoryGB, v.MaxMemoryGB)...)
	}
	
	// Validate optional CPU cores
	if cpuCores != nil {
		errs = append(errs, ValidateCPUCores(*cpuCores, v.MinCPUCores, v.MaxCPUCores)...)
	}
	
	// Validate tags (if provided)
	if tags != nil {
		errs = append(errs, ValidateTags(tags)...)
	}
	
	return errs
}

// validateOSConfig validates OS configuration
func validateOSConfig(os model.OSConfig) ValidationErrors {
	var errs ValidationErrors
	
	// OS type is required
	if os.Type == "" {
		errs = appendError(errs, "os.type", "is required")
	}
	
	// Validate OS type value
	switch os.Type {
	case model.OSTypeLinux, model.OSTypeWindows, model.OSTypeSolaris, model.OSTypeOther:
		// Valid
	default:
		errs = appendError(errs, "os.type", "must be one of: linux, windows, solaris, other")
	}
	
	// OS version is optional but if provided should not be too long
	if os.Version != "" && len(os.Version) > 64 {
		errs = appendError(errs, "os.version", "must be 64 characters or less")
	}
	
	return errs
}

// validateNetworkConfig validates network configuration
func (v *VMCreateRequestValidator) validateNetworkConfig(network []model.NetworkConfig) ValidationErrors {
	var errs ValidationErrors
	
	if len(network) == 0 {
		// At least one network is required
		errs = appendError(errs, "network", "at least one network interface is required")
		return errs
	}
	
	if len(network) > v.MaxNetworks {
		errs = appendError(errs, "network", fmt.Sprintf("cannot have more than %d network interfaces", v.MaxNetworks))
	}
	
	for i, net := range network {
		fieldPrefix := fmt.Sprintf("network[%d]", i)
		
		// Validate network type
		switch net.Type {
		case model.NetworkTypeUser, model.NetworkTypeBridge:
			// Valid
		default:
			errs = append(errs, &ValidationError{
				Field:   fmt.Sprintf("%s.type", fieldPrefix),
				Message: "must be one of: user, bridge",
			})
		}
		
		// Bridge is required for bridge type
		if net.Type == model.NetworkTypeBridge && net.Bridge == "" {
			errs = append(errs, &ValidationError{
				Field:   fmt.Sprintf("%s.bridge", fieldPrefix),
				Message: "is required when type is bridge",
			})
		} else if net.Bridge != "" {
			errs = append(errs, ValidateBridgeName(net.Bridge)...)
		}
		
		// Validate network model
		switch net.Model {
		case model.NetworkModelVirtio, model.NetworkModelE1000, model.NetworkModelRTL8139:
			// Valid
		case "":
			// Default will be applied
		default:
			errs = append(errs, &ValidationError{
				Field:   fmt.Sprintf("%s.model", fieldPrefix),
				Message: "must be one of: virtio, e1000, rtl8139",
			})
		}
		
		// Validate VLAN
		if net.VLAN < 0 || net.VLAN > 4094 {
			errs = append(errs, &ValidationError{
				Field:   fmt.Sprintf("%s.vlan", fieldPrefix),
				Message: "must be between 0 and 4094",
			})
		}
		
		// Validate port forwards
		for j, pf := range net.PortForwards {
			if !validatePortForward(pf) {
				errs = append(errs, &ValidationError{
					Field:   fmt.Sprintf("%s.port_forwards[%d]", fieldPrefix, j),
					Message: "must be in format host_port:guest_port (e.g., 2222:22)",
				})
			}
		}
	}
	
	return errs
}

// validatePortForward validates a port forward string
func validatePortForward(pf string) bool {
	parts := splitPortForward(pf)
	if len(parts) != 2 {
		return false
	}
	
	hostPort := parts[0]
	guestPort := parts[1]
	
	return isValidPort(hostPort) && isValidPort(guestPort)
}

// splitPortForward splits a port forward string
func splitPortForward(pf string) []string {
	result := make([]string, 0, 2)
	current := ""
	
	for _, r := range pf {
		if r == ':' {
			result = append(result, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	
	if current != "" {
		result = append(result, current)
	}
	
	return result
}

// isValidPort validates a port number string
func isValidPort(port string) bool {
	if port == "" {
		return false
	}
	
	for _, r := range port {
		if r < '0' || r > '9' {
			return false
		}
	}
	
	// Parse and check range
	var p int
	for _, r := range port {
		p = p*10 + int(r-'0')
	}
	
	return p >= 1 && p <= 65535
}

// ValidateDiskConfig validates disk configuration
func ValidateDiskConfig(size float64, format, bus string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate size
	if size <= 0 {
		errs = appendError(errs, "size", "must be greater than 0")
	}
	
	if size > 10240 { // 10 TB
		errs = appendError(errs, "size", "must be at most 10240 GB (10 TB)")
	}
	
	// Validate format
	if format != "" {
		switch format {
		case "qcow2", "raw", "vmdk", "vdi":
			// Valid
		default:
			errs = appendError(errs, "format", "must be one of: qcow2, raw, vmdk, vdi")
		}
	}
	
	// Validate bus
	if bus != "" {
		switch bus {
		case "virtio", "sata", "scsi", "ide":
			// Valid
		default:
			errs = appendError(errs, "bus", "must be one of: virtio, sata, scsi, ide")
		}
	}
	
	return errs
}
