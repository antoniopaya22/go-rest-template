# Security Policy

## Supported versions

| Version | Supported |
|---|---|
| `3.x` (the `main` branch) | Yes |
| `2.x` | No |
| `1.0.0` | No, kept for reference only |

Only the latest release receives security fixes.

## Reporting a vulnerability

Please **do not open a public issue**. Report it privately through GitHub's
[private vulnerability reporting](https://github.com/antoniopaya22/go-rest-template/security/advisories/new)
or by email to antoniopaya@outlook.com.

You can expect an acknowledgement within a few days. Once the issue is confirmed, a fix will be prepared and published together with an advisory crediting the reporter (unless you prefer to stay anonymous).

## Verifying releases

Container images published to `ghcr.io/antoniopaya22/go-rest-template` are
signed with [cosign](https://github.com/sigstore/cosign) using keyless signing
and carry a build provenance attestation:

```sh
cosign verify ghcr.io/antoniopaya22/go-rest-template:<version> \
  --certificate-identity-regexp '^https://github.com/antoniopaya22/go-rest-template/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

gh attestation verify oci://ghcr.io/antoniopaya22/go-rest-template:<version> \
  --repo antoniopaya22/go-rest-template
```
