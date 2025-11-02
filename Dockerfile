# Dockerfile for vervids storage container
FROM alpine:latest

# Install Python for asset parsing
RUN apk add --no-cache python3 py3-pip

# Create storage directories
RUN mkdir -p /storage/projects

# Set working directory
WORKDIR /storage

# Add a simple health check
HEALTHCHECK --interval=30s --timeout=3s \
  CMD test -d /storage/projects || exit 1

# Keep container running
CMD ["tail", "-f", "/dev/null"]

