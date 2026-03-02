package libvirt

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

// BackupRepository handles VM backup operations using tar
type BackupRepository struct {
	client    *libvirtx.Client
	backupDir string
}

// NewBackupRepository creates a new backup repository
func NewBackupRepository(client *libvirtx.Client, backupDir string) *BackupRepository {
	return &BackupRepository{
		client:    client,
		backupDir: backupDir,
	}
}

// Create creates a backup of a VM
func (r *BackupRepository) Create(backup *model.Backup, compress bool, encrypt bool, passphrase string) error {
	domainName := fmt.Sprintf("vm-%d", backup.VMID)

	// Check if VM exists
	domain, err := r.client.GetDomainByName(domainName)
	if err != nil {
		return fmt.Errorf("VM %d not found: %w", backup.VMID, err)
	}
	defer domain.Free()

	// Get VM state
	state, _, err := domain.GetState()
	if err != nil {
		return fmt.Errorf("failed to get VM state: %w", err)
	}

	// Note: For running VMs, we recommend stopping or using external snapshot tools
	// This simple implementation works best with stopped VMs
	_ = state // State check for future enhancement

	// Get VM XML configuration
	vmXML, err := domain.GetXMLDesc(0)
	if err != nil {
		return fmt.Errorf("failed to get VM XML: %w", err)
	}

	// Create backup directory
	backupSubdir := filepath.Join(r.backupDir, fmt.Sprintf("vm-%d", backup.VMID))
	if err := os.MkdirAll(backupSubdir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup filename
	backupFile := filepath.Join(backupSubdir, fmt.Sprintf("%s.tar", backup.ID))
	if compress {
		backupFile += ".gz"
	}
	if encrypt {
		backupFile += ".luks"
	}

	// Create backup archive
	if err := r.createBackupArchive(backupFile, vmXML, domainName, compress, encrypt, passphrase); err != nil {
		return fmt.Errorf("failed to create backup archive: %w", err)
	}

	// Get backup size
	info, err := os.Stat(backupFile)
	if err != nil {
		return fmt.Errorf("failed to get backup size: %w", err)
	}

	backup.SizeBytes = info.Size()
	backup.BackupPath = backupFile
	backup.Status = model.BackupStatusCompleted
	backup.CompletedAt = time.Now().Format(time.RFC3339)
	backup.Encrypted = encrypt
	backup.VerificationStatus = model.VerificationStatusNotRun

	// Calculate expiration
	if backup.RetentionDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, backup.RetentionDays)
		backup.ExpiresAt = expiresAt.Format(time.RFC3339)
	}

	return nil
}

// createBackupArchive creates a tar (optionally gzipped and encrypted) backup archive
func (r *BackupRepository) createBackupArchive(backupFile, vmXML, domainName string, compress bool, encrypt bool, passphrase string) error {
	// First, create the tar archive to a temporary file
	tempFile := backupFile + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return err
	}

	var writer io.Writer = file
	if compress {
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = gzWriter
	}

	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	// Add VM XML configuration
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "vm-config.xml",
		Mode: 0644,
		Size: int64(len(vmXML)),
	}); err != nil {
		os.Remove(tempFile)
		return err
	}

	if _, err := tarWriter.Write([]byte(vmXML)); err != nil {
		os.Remove(tempFile)
		return err
	}

	// Get disk information from XML and add disks to backup
	// For now, we'll use virsh to export disk data
	diskPaths, err := r.getDiskPaths(domainName)
	if err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to get disk paths: %w", err)
	}

	for _, diskPath := range diskPaths {
		if _, err := os.Stat(diskPath); os.IsNotExist(err) {
			continue // Skip non-existent disks
		}

		info, err := os.Stat(diskPath)
		if err != nil {
			continue
		}

		// Add disk to tar archive
		if err := tarWriter.WriteHeader(&tar.Header{
			Name: "disks/" + filepath.Base(diskPath),
			Mode: 0644,
			Size: info.Size(),
		}); err != nil {
			os.Remove(tempFile)
			return err
		}

		diskFile, err := os.Open(diskPath)
		if err != nil {
			os.Remove(tempFile)
			return err
		}

		if _, err := io.Copy(tarWriter, diskFile); err != nil {
			diskFile.Close()
			os.Remove(tempFile)
			return err
		}
		diskFile.Close()
	}

	// Close the writers to flush data
	if compress {
		if gz, ok := writer.(*gzip.Writer); ok {
			gz.Close()
		}
	}
	file.Close()

	// If encryption is requested, encrypt the archive
	if encrypt {
		if err := r.encryptFile(tempFile, backupFile, passphrase); err != nil {
			os.Remove(tempFile)
			return fmt.Errorf("failed to encrypt backup: %w", err)
		}
		os.Remove(tempFile)
	} else {
		// Just rename temp file to final file
		if err := os.Rename(tempFile, backupFile); err != nil {
			os.Remove(tempFile)
			return err
		}
	}

	return nil
}

// encryptFile encrypts a file using LUKS format via qemu-img
func (r *BackupRepository) encryptFile(srcFile, dstFile, passphrase string) error {
	// Use qemu-img to create an encrypted copy
	// qemu-img convert -O qcow2 --object secret,id=sec0,data=passphrase -o encrypt.format=luks,encrypt.key-secret=sec0 src dst
	// For simplicity, we use openssl for symmetric encryption
	cmd := exec.Command("openssl", "enc", "-aes-256-cbc", "-salt", "-pbkdf2", "-iter", "100000",
		"-in", srcFile,
		"-out", dstFile,
		"-pass", "pass:"+passphrase,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("openssl encryption failed: %w, output: %s", err, string(output))
	}
	return nil
}

// decryptFile decrypts a file encrypted with encryptFile
func (r *BackupRepository) decryptFile(srcFile, dstFile, passphrase string) error {
	cmd := exec.Command("openssl", "enc", "-aes-256-cbc", "-d", "-pbkdf2", "-iter", "100000",
		"-in", srcFile,
		"-out", dstFile,
		"-pass", "pass:"+passphrase,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("openssl decryption failed: %w, output: %s", err, string(output))
	}
	return nil
}

// getDiskPaths returns the paths to VM disk images
func (r *BackupRepository) getDiskPaths(domainName string) ([]string, error) {
	cmd := exec.Command("virsh", "domblklist", domainName, "--details")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var paths []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[0] != "Target" {
			// Last field is the source path
			path := fields[len(fields)-1]
			if path != "-" && strings.HasPrefix(path, "/") {
				paths = append(paths, path)
			}
		}
	}

	return paths, nil
}

// Restore restores a VM from a backup
func (r *BackupRepository) Restore(backup *model.Backup, targetVMID int, startAfter bool, passphrase string) error {
	if backup.BackupPath == "" {
		return fmt.Errorf("backup has no associated file")
	}

	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "backup-restore-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Handle encrypted backups
	backupFile := backup.BackupPath
	if backup.Encrypted {
		if passphrase == "" {
			return fmt.Errorf("passphrase required for encrypted backup")
		}
		decryptedFile := filepath.Join(tempDir, "decrypted.tar")
		if strings.HasSuffix(backup.BackupPath, ".gz") {
			decryptedFile += ".gz"
		}
		if err := r.decryptFile(backup.BackupPath, decryptedFile, passphrase); err != nil {
			return fmt.Errorf("failed to decrypt backup: %w", err)
		}
		backupFile = decryptedFile
		defer os.Remove(decryptedFile)
	}

	// Extract backup
	if err := r.extractBackup(backupFile, tempDir); err != nil {
		return fmt.Errorf("failed to extract backup: %w", err)
	}

	// Read VM configuration
	vmConfigPath := filepath.Join(tempDir, "vm-config.xml")
	vmXML, err := os.ReadFile(vmConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read VM config: %w", err)
	}

	// Modify VM ID in XML if restoring to different VMID
	if targetVMID > 0 && targetVMID != backup.VMID {
		vmXML = []byte(strings.ReplaceAll(string(vmXML),
			fmt.Sprintf("vm-%d", backup.VMID),
			fmt.Sprintf("vm-%d", targetVMID)))
	}

	// Define the VM in libvirt
	cmd := exec.Command("virsh", "define", "-")
	cmd.Stdin = strings.NewReader(string(vmXML))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to define VM: %w, output: %s", err, string(output))
	}

	// Restore disk images
	diskDir := filepath.Join(tempDir, "disks")
	if entries, err := os.ReadDir(diskDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			sourcePath := filepath.Join(diskDir, entry.Name())
			// Get target path from original config or use default location
			targetPath := filepath.Join(r.backupDir, "disks", fmt.Sprintf("vm-%d", targetVMID), entry.Name())
			if targetVMID == 0 {
				targetPath = filepath.Join(r.backupDir, "disks", fmt.Sprintf("vm-%d", backup.VMID), entry.Name())
			}

			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create disk directory: %w", err)
			}

			if err := copyFile(sourcePath, targetPath); err != nil {
				return fmt.Errorf("failed to restore disk %s: %w", entry.Name(), err)
			}
		}
	}

	// Start VM if requested
	if startAfter {
		vmName := fmt.Sprintf("vm-%d", targetVMID)
		if targetVMID == 0 {
			vmName = fmt.Sprintf("vm-%d", backup.VMID)
		}

		cmd := exec.Command("virsh", "start", vmName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to start VM: %w, output: %s", err, string(output))
		}
	}

	return nil
}

// extractBackup extracts a tar backup archive
func (r *BackupRepository) extractBackup(backupFile, destDir string) error {
	file, err := os.Open(backupFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var reader io.Reader = file
	if strings.HasSuffix(backupFile, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destDir, header.Name)

		// Create directories
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		// Create file
		outFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}

		if _, err := io.Copy(outFile, tarReader); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}

	return nil
}

// Delete deletes a backup file
func (r *BackupRepository) Delete(backup *model.Backup) error {
	if backup.BackupPath == "" {
		return nil // Nothing to delete
	}

	if err := os.Remove(backup.BackupPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete backup file: %w", err)
		}
	}

	// Try to delete parent directory if empty
	backupDir := filepath.Dir(backup.BackupPath)
	if entries, err := os.ReadDir(backupDir); err == nil && len(entries) == 0 {
		os.Remove(backupDir)
	}

	return nil
}

// Verify verifies a backup's integrity using qemu-img check
// Returns the raw output and any error encountered
func (r *BackupRepository) Verify(backup *model.Backup) (string, error) {
	if backup.BackupPath == "" {
		return "", fmt.Errorf("backup has no associated file")
	}

	// For encrypted backups, we can't directly verify - need to decrypt first
	// This is a simplified implementation that just checks file integrity
	if backup.Encrypted {
		// For encrypted backups, we verify by checking if the file can be read
		file, err := os.Open(backup.BackupPath)
		if err != nil {
			return "", fmt.Errorf("cannot open encrypted backup: %w", err)
		}
		defer file.Close()

		// Read first few bytes to verify file is readable
		buf := make([]byte, 16)
		if _, err := file.Read(buf); err != nil {
			return "", fmt.Errorf("encrypted backup file is corrupted: %w", err)
		}

		// Check for "Salted__" magic header (OpenSSL encrypted format)
		if string(buf[:8]) != "Salted__" {
			return "", fmt.Errorf("encrypted backup file has invalid format")
		}

		return "Encrypted backup file verified (format check only)", nil
	}

	// For tar archives, verify using tar -t
	if strings.HasSuffix(backup.BackupPath, ".tar") || strings.HasSuffix(backup.BackupPath, ".tar.gz") {
		cmd := exec.Command("tar", "-tzf", backup.BackupPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return string(output), fmt.Errorf("backup archive is corrupted: %w", err)
		}
		return "Backup archive verified successfully\n" + string(output), nil
	}

	// For qcow2/raw disk images, use qemu-img check
	cmd := exec.Command("qemu-img", "check", backup.BackupPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("backup verification failed: %w", err)
	}

	return string(output), nil
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}

// GetBackupSize returns the size of a backup file
func (r *BackupRepository) GetBackupSize(backupPath string) (int64, error) {
	info, err := os.Stat(backupPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// BackupExists checks if a backup file exists
func (r *BackupRepository) BackupExists(backupPath string) bool {
	_, err := os.Stat(backupPath)
	return err == nil
}
