# Docker Metrics with Google Cloud

The Docker Receiver can be used to send Docker metrics to Google Cloud Monitoring.

## Prerequisites

**Google Cloud**

See the [prerequisites](../prerequisites.md) doc for Google Cloud prerequisites.

**Docker Socket**

The provided configuration assumes the agent is running on the Docker system. By default, the `endpoint` collected from is `unix:///var/run/docker.sock`.

The user running the agent must have permission to read the [docker socket](https://docs.docker.com/engine/install/linux-postinstall/).

```bash
sudo usermod -aG docker bindplane-otel-collector
```

## Deployment

**On Host**

If the agent is running on the Docker host (not within a container), deployment is as simple as installing the agent and updating the configuration.

**Docker Container**

If running the agent as a container, you will need to mount the Docker socket with a read only volume mount. Additionally, configuration and credentials will need to be mounted.

Run as a container

```bash
docker run -d \
    --restart=always \
    --volume="/var/run/docker.sock:/var/run/docker.sock:ro"  \
    --volume="$(pwd)/config.yaml:/etc/otel/config.yaml" \
    --volume="$(pwd)/credentials.json:/etc/otel/credentials.json" \
    -e "GOOGLE_APPLICATION_CREDENTIALS=/etc/otel/credentials.json" \
    -e "DOCKER_AGENT_HOSTNAME=$(hostname)" \
    observiq/bindplane-otel-collector:v1.30.0
```

Run with Docker Compose

```yaml
version: "3.9"
services:
  agent:
    restart: always
    # Run as root if using a configuration that requires
    # root privileges.
    #user: root
    image: observiq/bindplane-otel-collector:v1.30.0
    restart: always
    deploy:
      resources:
        limits:
          cpus: 0.50
          memory: 256M
    environment:
      - GOOGLE_APPLICATION_CREDENTIALS=/etc/otel/credentials.json
      - DOCKER_AGENT_HOSTNAME=${HOSTNAME}
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ${PWD}/config.yaml:/etc/otel/config.yaml
      - ${PWD}/credentials.json:/etc/otel/credentials.json
```
