# Files

- [Deploy, Backup & Restore](deploy-restore.md) - Production deploy via make deploy, x86_64/amd64 architecture constraint, the docker-compose layout (api-server + MySQL + Redis + nginx), migration discipline, and the backup/restore flow.
- [Local Development](local-dev.md) - Bring-up the dev stack — docker-compose.dev.yml (MySQL + Redis + adminer + 9Pay mock + OnePay mock), air hot-reload for the backend, vite for the frontend, ports, and admin tooling.
- [Observability](observability.md) - Structured logging (slog), Prometheus metrics, error code conventions, request id correlation, and the system-health and cron-health UI surfaces used by admins.
