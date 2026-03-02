package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestVMTemplatesURLsValid verifies that all ISO URLs in VM templates are accessible.
// This test makes real HTTP requests to validate the URLs haven't expired or changed.
func TestVMTemplatesURLsValid(t *testing.T) {
	templates := VMTemplates()
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	ctx := context.Background()
	
	for _, template := range templates {
		t.Run(template.Name, func(t *testing.T) {
			// Test x86_64 URL
			if template.ISOURLx86_64 != "" {
				t.Run("x86_64", func(t *testing.T) {
					url := template.GetISOURLForArch("x86_64")
					if url == "" {
						t.Skip("URL is empty for x86_64")
					}
					
					// Create HEAD request to check if URL exists (lightweight)
					req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
					if err != nil {
						t.Fatalf("Failed to create request: %v", err)
					}
					
					// Set User-Agent to avoid being blocked
					req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Lab VM Template Test)")
					
					resp, err := client.Do(req)
					if err != nil {
						t.Errorf("Failed to reach URL for %s (x86_64): %v\nURL: %s", 
							template.Name, err, url)
						return
					}
					defer resp.Body.Close()
					
					// Check if we got a successful response (2xx or 3xx)
					// Some servers return 302/301 redirects which is acceptable
					if resp.StatusCode >= 400 {
						t.Errorf("URL returned HTTP %d for %s (x86_64)\nURL: %s", 
							resp.StatusCode, template.Name, url)
					} else {
						fmt.Printf("✓ %s (x86_64): %s -> HTTP %d\n", 
							template.Name, url, resp.StatusCode)
					}
				})
			}
			
			// Test aarch64 URL
			if template.ISOURLaarch64 != "" {
				t.Run("aarch64", func(t *testing.T) {
					url := template.GetISOURLForArch("aarch64")
					if url == "" {
						t.Skip("URL is empty for aarch64")
					}
					
					req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
					if err != nil {
						t.Fatalf("Failed to create request: %v", err)
					}
					
					req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Lab VM Template Test)")
					
					resp, err := client.Do(req)
					if err != nil {
						t.Errorf("Failed to reach URL for %s (aarch64): %v\nURL: %s", 
							template.Name, err, url)
						return
					}
					defer resp.Body.Close()
					
					if resp.StatusCode >= 400 {
						t.Errorf("URL returned HTTP %d for %s (aarch64)\nURL: %s", 
							resp.StatusCode, template.Name, url)
					} else {
						fmt.Printf("✓ %s (aarch64): %s -> HTTP %d\n", 
							template.Name, url, resp.StatusCode)
					}
				})
			}
			
			// Skip if no URLs provided (user must provide ISO)
			if template.ISOURLx86_64 == "" && template.ISOURLaarch64 == "" {
				t.Skip("Template does not provide ISO URLs (user must provide)")
			}
		})
	}
}

// TestVMTemplatesHaveRequiredFields verifies all templates have required fields populated.
func TestVMTemplatesHaveRequiredFields(t *testing.T) {
	templates := VMTemplates()
	
	for _, template := range templates {
		t.Run(template.Name, func(t *testing.T) {
			if template.ID == "" {
				t.Error("Template ID is required")
			}
			if template.Name == "" {
				t.Error("Template name is required")
			}
			if template.Icon == "" {
				t.Error("Template icon is required")
			}
			if template.ISOName == "" {
				t.Error("Template ISO name is required")
			}
			// At least one architecture URL should be provided (or both empty for user-provided ISO)
			if template.ISOURLx86_64 == "" && template.ISOURLaarch64 == "" && template.ID != "windows-11" && template.ID != "windows-server-2022" {
				t.Error("Template should have at least one ISO URL (or be Windows)")
			}
			if template.CPUCores <= 0 {
				t.Error("Template CPU cores must be positive")
			}
			if template.MemoryGB <= 0 {
				t.Error("Template memory must be positive")
			}
			if template.DiskGB <= 0 {
				t.Error("Template disk size must be positive")
			}
		})
	}
}

// TestVMTemplatesArchSubstitution verifies architecture URL selection works correctly.
func TestVMTemplatesArchSubstitution(t *testing.T) {
	templates := VMTemplates()
	
	for _, template := range templates {
		if template.ISOURLx86_64 == "" && template.ISOURLaarch64 == "" {
			continue // Skip templates without URLs
		}
		
		t.Run(template.Name, func(t *testing.T) {
			// Test x86_64 URL
			x86URL := template.GetISOURLForArch("x86_64")
			if x86URL != "" {
				if x86URL != template.ISOURLx86_64 {
					t.Errorf("x86_64 URL mismatch: got %s, want %s", x86URL, template.ISOURLx86_64)
				}
			}
			
			// Test aarch64 URL
			armURL := template.GetISOURLForArch("aarch64")
			if armURL != "" {
				if armURL != template.ISOURLaarch64 {
					t.Errorf("aarch64 URL mismatch: got %s, want %s", armURL, template.ISOURLaarch64)
				}
			}
			
			// Test that URLs are different for different architectures (if both exist)
			if x86URL != "" && armURL != "" && x86URL == armURL {
				t.Error("x86_64 and aarch64 URLs should be different")
			}
			
			// Test SupportsArch method
			if x86URL != "" && !template.SupportsArch("x86_64") {
				t.Error("Template should support x86_64")
			}
			if armURL != "" && !template.SupportsArch("aarch64") {
				t.Error("Template should support aarch64")
			}
		})
	}
}
