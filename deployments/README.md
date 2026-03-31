# Deployments

This directory holds runnable deployment manifests for the template.

## Local Docker Compose

Use [docker-compose.yml](docker-compose.yml) for a local PostgreSQL-backed environment:

```powershell
make compose-up
```

Stop and remove it with:

```powershell
make compose-down
```

The compose stack builds the API image from [build/package/Dockerfile](../build/package/Dockerfile) and wires the application to the PostgreSQL service using the same environment variables documented in [configs/env.example](../configs/env.example).