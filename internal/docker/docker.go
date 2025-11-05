package docker

import (
    "errors"
    "fmt"
    "os/exec"
    "regexp"
    "strconv"
    "strings"
    "time"
)

const (
    ContainerName  = "vervids-storage"
    VolumeName     = "vervids-data"
    StoragePath    = "/vervids"
    MinDockerSemver = "24.0.0"
)

// IsDockerInstalled checks if Docker is available
func IsDockerInstalled() bool {
	cmd := exec.Command("docker", "--version")
	err := cmd.Run()
	return err == nil
}

// IsDockerDaemonRunning checks if Docker daemon is accessible
func IsDockerDaemonRunning() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stderr = nil // Suppress stderr
	err := cmd.Run()
	return err == nil
}

// StartDockerDesktop starts Docker Desktop (macOS)
func StartDockerDesktop() error {
	// Try to start Docker Desktop on macOS
	cmd := exec.Command("open", "-a", "Docker")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start Docker Desktop: %w", err)
	}
	return nil
}

// WaitForDocker waits for Docker daemon to become available (with timeout)
func WaitForDocker(maxWaitSeconds int) error {
	for i := 0; i < maxWaitSeconds; i++ {
		if IsDockerDaemonRunning() {
			return nil
		}
		// Wait 1 second before checking again
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("Docker daemon did not become available within %d seconds", maxWaitSeconds)
}

func GetDockerVersion() (string, error) {
    out, err := exec.Command("docker", "--version").CombinedOutput()
    if err != nil {
        return "", err
    }
    // Example: Docker version 24.0.7, build ...
    re := regexp.MustCompile(`Docker version ([0-9]+)\.([0-9]+)\.([0-9]+)`)
    m := re.FindStringSubmatch(string(out))
    if len(m) != 4 {
        return "", errors.New("unable to parse docker version")
    }
    return fmt.Sprintf("%s.%s.%s", m[1], m[2], m[3]), nil
}

func versionGTE(a, b string) bool {
    as := strings.Split(a, ".")
    bs := strings.Split(b, ".")
    for i := 0; i < 3; i++ {
        ai, _ := strconv.Atoi(as[i])
        bi, _ := strconv.Atoi(bs[i])
        if ai > bi {
            return true
        }
        if ai < bi {
            return false
        }
    }
    return true
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

// IsVolumeExists checks if the Docker volume exists
func IsVolumeExists() bool {
	cmd := exec.Command("docker", "volume", "inspect", VolumeName)
	// Suppress stderr to avoid printing errors if volume doesn't exist
	cmd.Stderr = nil
	err := cmd.Run()
	return err == nil
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

	// Create volume (ignore error if it already exists)
	volumeCmd := exec.Command("docker", "volume", "create", VolumeName)
	output, err := volumeCmd.CombinedOutput()
	if err != nil {
		// Check if error is because volume already exists
		outputStr := strings.ToLower(string(output))
		if strings.Contains(outputStr, "already exists") {
			// Volume exists, that's fine - continue
		} else {
			return fmt.Errorf("failed to create volume: %w (output: %s)", err, string(output))
		}
	}

	// Run container
	cmd := exec.Command("docker", "run", "-d",
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

// PathExistsInContainer checks if a path exists inside the container
func PathExistsInContainer(path string) bool {
    _, err := ExecInContainer("sh", "-lc", fmt.Sprintf("[ -e %q ]", path))
    return err == nil
}

// DeleteDirectory deletes a directory and all its contents recursively inside the container
func DeleteDirectory(path string) error {
    _, err := ExecInContainer("rm", "-rf", path)
    return err
}

// EnsureDockerReady validates Docker installation, version and container state
// It will automatically start Docker Desktop if needed (macOS)
func EnsureDockerReady() error {
    if !IsDockerInstalled() {
        return fmt.Errorf("Docker is required. Please install Docker %s or newer.", MinDockerSemver)
    }
    
    // Check if Docker daemon is running
    if !IsDockerDaemonRunning() {
        // Try to start Docker Desktop automatically (macOS)
        fmt.Println("🐳 Docker is not running. Starting Docker Desktop...")
        if err := StartDockerDesktop(); err != nil {
            return fmt.Errorf("Docker is not running. Please start Docker Desktop manually: %w", err)
        }
        
        // Wait for Docker to become available (max 30 seconds)
        fmt.Println("⏳ Waiting for Docker to start...")
        if err := WaitForDocker(30); err != nil {
            return fmt.Errorf("Docker did not start in time. Please ensure Docker Desktop is running: %w", err)
        }
        fmt.Println("✓ Docker is ready")
    }
    
    v, err := GetDockerVersion()
    if err != nil {
        return fmt.Errorf("failed to read Docker version: %v", err)
    }
    if !versionGTE(v, MinDockerSemver) {
        return fmt.Errorf("Docker %s or newer is required (found %s). Please upgrade.", MinDockerSemver, v)
    }
    if !IsContainerRunning() {
        if IsContainerExists() {
            return StartContainer()
        }
        return CreateContainer()
    }
    return nil
}

