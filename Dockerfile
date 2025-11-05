# Dockerfile for vervids storage container
FROM alpine:latest

# Note: Python is no longer required - asset parsing is done natively in Go
# Keeping minimal alpine base for storage container

# Create storage directories
RUN mkdir -p /storage/projects

# Set working directory
WORKDIR /storage

# Add a simple health check
HEALTHCHECK --interval=30s --timeout=3s \
  CMD test -d /storage/projects || exit 1

# Keep container running
CMD ["tail", "-f", "/dev/null"]

