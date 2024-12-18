# Cadvisor Metrics with Google Cloud

The Prometheus Receiver can be used to send [Cadvisor](https://github.com/google/cadvisor) metrics to Google Cloud Monitoring.

## Prerequisites

**Google Cloud**

See the [prerequisites](../prerequisites.md) doc for Google Cloud prerequisites.

[Cadvisor](https://github.com/google/cadvisor) should be running as a container with the name `cadvisor`.

## Deployment

Run with Docker Compose. Docker Compose assumes that a `credentials.json`
service account key exists in this directory.

```bash
 docker-compose up -d
```

**_docker-compose.yml_**

```yaml
version: "3.9"
services:
  cadvisor:
    image: gcr.io/cadvisor/cadvisor:latest
    restart: always
    container_name: cadvisor
    hostname: cadvisor
    ports:
      - 8080:8080
    volumes:
      - /:/rootfs:ro
      - /var/run:/var/run:rw
      - /sys:/sys:ro
      - /var/lib/docker/:/var/lib/docker:ro

  agent:
    image: observiq/bindplane-otel-collector:1.30.0
    restart: always
    container_name: bindplane-otel-collector
    hostname: bindplane-otel-collector
    deploy:
      resources:
        limits:
          cpus: 0.50
          memory: 256M
    environment:
      - GOOGLE_APPLICATION_CREDENTIALS=/etc/otel/credentials.json
    volumes:
      - ${PWD}/config.yaml:/etc/otel/config.yaml
      - ${PWD}/credentials.json:/etc/otel/credentials.json
```
