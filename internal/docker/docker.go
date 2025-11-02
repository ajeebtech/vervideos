package docker

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	ContainerName = "vervids-storage"
	VolumeName    = "vervids-data"
	StoragePath   = "/storage/projects"
)

// IsDockerInstalled checks if Docker is available
func IsDockerInstalled() bool {
	cmd := exec.Command("docker", "--version")
	err := cmd.Run()
	return err == nil
}

// IsContainerRunning checks if the vervids storage container is running
func IsContainerRunning() bool {
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", ContainerName), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == ContainerName
}

// IsContainerExists checks if the container exists (running or stopped)
func IsContainerExists() bool {
	cmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", ContainerName), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == ContainerName
}

// CreateContainer creates and starts the storage container
func CreateContainer() error {
	// Check if container already exists
	if IsContainerExists() {
		// If exists but not running, start it
		if !IsContainerRunning() {
			return StartContainer()
		}
		return nil
	}

	// Create volume if it doesn't exist
	cmd := exec.Command("docker", "volume", "create", VolumeName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}

	// Run container
	cmd = exec.Command("docker", "run", "-d",
		"--name", ContainerName,
		"-v", fmt.Sprintf("%s:%s", VolumeName, StoragePath),
		"alpine:latest",
		"tail", "-f", "/dev/null")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	return nil
}

// StartContainer starts an existing container
func StartContainer() error {
	cmd := exec.Command("docker", "start", ContainerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// CopyToContainer copies a file from host to container
func CopyToContainer(srcPath, destPath string) error {
	containerPath := fmt.Sprintf("%s:%s", ContainerName, destPath)
	cmd := exec.Command("docker", "cp", srcPath, containerPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy to container: %w", err)
	}
	return nil
}

// CopyFromContainer copies a file from container to host
func CopyFromContainer(srcPath, destPath string) error {
	containerPath := fmt.Sprintf("%s:%s", ContainerName, srcPath)
	cmd := exec.Command("docker", "cp", containerPath, destPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy from container: %w", err)
	}
	return nil
}

// ExecInContainer executes a command inside the container
func ExecInContainer(command ...string) (string, error) {
	args := append([]string{"exec", ContainerName}, command...)
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute in container: %w", err)
	}
	return string(output), nil
}

// CreateDirectory creates a directory inside the container
func CreateDirectory(path string) error {
	_, err := ExecInContainer("mkdir", "-p", path)
	return err
}

// GetVolumeInfo returns information about the volume
func GetVolumeInfo() (map[string]string, error) {
	cmd := exec.Command("docker", "volume", "inspect", VolumeName, "--format", "{{.Mountpoint}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get volume info: %w", err)
	}

	info := map[string]string{
		"name":       VolumeName,
		"mountpoint": strings.TrimSpace(string(output)),
	}

	return info, nil
}

