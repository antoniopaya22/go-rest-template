# Kubernetes

The chart in `deployments/helm/go-rest-template` deploys the API as a `Deployment` with a
`Service`. It expects an existing PostgreSQL database; it does not install one.

## Install

```sh
helm upgrade --install api deployments/helm/go-rest-template \
  --namespace api --create-namespace \
  --set secrets.values.AUTH_SECRET="$(openssl rand -base64 48)" \
  --set secrets.values.DATABASE_URL='postgres://user:password@postgres:5432/app?sslmode=require'
```

The two secrets are required: in production mode the server refuses to start without them.

## What it creates

| Resource | When |
|---|---|
| `Deployment` | Always |
| `Service` with the ports `http` (80) and `admin` (9090) | Always |
| `ConfigMap` with the non-secret configuration | Always |
| `ServiceAccount` | `serviceAccount.create`, on by default |
| `Secret` | `secrets.create`, on by default, unless `secrets.existingSecret` is set |
| `Ingress` | `ingress.enabled` |
| `HorizontalPodAutoscaler` | `autoscaling.enabled` |
| `PodDisruptionBudget` | `podDisruptionBudget.enabled` |
| `NetworkPolicy` | `networkPolicy.enabled` |
| `ServiceMonitor` | `serviceMonitor.enabled`; needs the Prometheus Operator CRDs |

## Configuration

Non-secret variables go under `config` and reach the container through the `ConfigMap`. Any
variable of the [configuration reference](../reference/configuration.md) can be added:

```yaml
config:
  APP_ENV: production
  LOG_LEVEL: INFO
  HTTP_CORS_ALLOWED_ORIGINS: https://app.example.com
  HTTP_TRUSTED_PROXIES: "1"
  AUTH_REGISTRATION_ENABLED: "false"
```

The chart always sets `HTTP_PORT=8080` and `ADMIN_PORT=9090`, which the probes and the
`Service` rely on.

Secrets go under `secrets.values` and reach the container through a `Secret`:

```yaml
secrets:
  values:
    AUTH_SECRET: "..."
    DATABASE_URL: "postgres://..."
```

To use a `Secret` managed elsewhere, such as External Secrets or Sealed Secrets, name it
instead. Its keys must be environment variable names:

```yaml
secrets:
  create: false
  existingSecret: api-secrets
```

Pods are restarted when the `ConfigMap` or the chart-managed `Secret` changes, through
checksum annotations.

## Security defaults

The pod runs with a restricted security context:

```yaml
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 65532
  runAsGroup: 65532
  fsGroup: 65532
  seccompProfile:
    type: RuntimeDefault

securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop: [ALL]
```

The root filesystem is read-only; `/tmp` is an `emptyDir`. The service account token is not
mounted (`serviceAccount.automount: false`), since the application does not call the
Kubernetes API.

## Probes

| Probe | Path | Notes |
|---|---|---|
| Startup | `/health/live` | Every 2 seconds, up to 30 failures: a minute for migrations before liveness applies |
| Liveness | `/health/live` | Every 10 seconds |
| Readiness | `/health/ready` | Every 5 seconds; fails when the database does not answer or the pod is shutting down |

`terminationGracePeriodSeconds` is 30 and must stay longer than `APP_SHUTDOWN_TIMEOUT`.

## Migrations

`config.DATABASE_AUTO_MIGRATE` is `"true"`: each pod applies pending migrations when it
starts. Migrations take a PostgreSQL advisory lock, so the two default replicas starting
together do not conflict.

To migrate as a step of your pipeline instead, set it to `"false"` and run `/app/migrate
up` from the same image, for example as a `Job` you run before the upgrade. The chart does
not include such a job.

## Scaling and availability

```yaml
replicaCount: 2            # ignored when autoscaling is enabled

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 75

podDisruptionBudget:
  enabled: true
  minAvailable: 1
```

The default resources are a request of 100m CPU and 64Mi memory and a memory limit of
256Mi. Size them from your own measurements. Remember that rate limiting is per pod: the
effective limit grows with the number of replicas.

## Exposing the API

```yaml
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: api.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: api-example-com-tls
      hosts: [api.example.com]
```

The ingress routes to the `http` port only. Set `HTTP_TRUSTED_PROXIES` to the number of
proxies in front of the pods so that client addresses, and therefore rate limiting and
logs, are correct.

## Restricting traffic

```yaml
networkPolicy:
  enabled: true
  apiFrom:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: ingress-nginx
  adminFrom:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: monitoring
```

With the policy enabled, an empty `apiFrom` allows any source to reach the API port, and an
empty `adminFrom` allows **no one** to reach the admin port. The policy restricts ingress
only.

## Metrics

The admin port is exposed by the `Service` for in-cluster scraping. With the Prometheus
Operator:

```yaml
serviceMonitor:
  enabled: true
  interval: 30s
  labels:
    release: prometheus
```

## EdDSA keys and other files

Mount extra files with `extraVolumes` and `extraVolumeMounts`, and point the application at
them:

```yaml
config:
  AUTH_ALGORITHM: EdDSA
  AUTH_PRIVATE_KEY_FILE: /etc/auth/key.pem

extraVolumes:
  - name: auth-key
    secret:
      secretName: api-auth-key

extraVolumeMounts:
  - name: auth-key
    mountPath: /etc/auth
    readOnly: true
```

`extraEnv` adds environment variables with the full Kubernetes syntax, for example from a
field reference.

## Checking the chart

```sh
helm lint deployments/helm/go-rest-template
helm template api deployments/helm/go-rest-template --set ingress.enabled=true
```

CI runs both, with the optional resources enabled.
