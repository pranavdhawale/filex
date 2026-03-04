# FileX Setup Guide (Docker)

This guide provides step-by-step instructions to get FileX up and running using Docker. Our setup features **multi-stage builds** and **containerized background workers** to provide the best experience for both development and production.

---

## 🏗 Architecture Overview

FileX consists of several core services:

- **Client**: Next.js 16 application (React 19).
- **API Server**: Go backend handling HTTP requests and orchestration.
- **Workers**: Go background processes (`worker-expiry`, `worker-multipart`, `worker-gc`) for async tasks.
- **Database**: MongoDB for metadata and configuration.
- **Cache & Pub/Sub**: Redis for rate limiting and inter-process communication.
- **Storage**: MinIO for S3-compatible, secure object storage of encrypted files.

---

## 📋 Prerequisites

Ensure you have the following installed:

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (Version 20.10+) or Docker Engine
- [Docker Compose](https://docs.docker.com/compose/install/) (Usually included with Docker Desktop)

---

## 🚀 Development Quick Start

To start developing locally with all backing services:

1.  **Prepare Environment Variables**:
    By default, the `docker-compose.dev.yml` file has all necessary environment variables pre-configured for local testing.

2.  **Spin up the environment**:

    ```bash
    docker-compose -f docker-compose.dev.yml up --build -d
    ```

    - The **Client** will be available at [http://localhost:3000](http://localhost:3000).
    - The **API Server** will be available at [http://localhost:8080](http://localhost:8080).
    - The **MinIO Console** will be available at [http://localhost:9001](http://localhost:9001).

3.  **Hot-Reloading Features**:
    - **Client**: Next.js provides instant hot-reloading for code changes in `client/`.

---

## 📦 Production Deployment

For production, FileX uses minimal, optimized container images and typically delegates routing to an external proxy.

### Starting Production

1.  **Configure Environment**:
    Provide your configuration via the environment file referenced in `docker-compose.prod.yml` (e.g., `/srv/docker/secrets/filex.env`).

2.  **Spin up the environment**:

    ```bash
    docker-compose -f docker-compose.prod.yml up -d --build
    ```

### Using Nginx Proxy Manager (NPM)?

If you use **Nginx Proxy Manager** (or another Nginx reverse proxy) in front of FileX API, you **must** disable request buffering for large multipart uploads to function efficiently without hanging.

1.  Edit your Proxy Host in NPM.
2.  Go to the **Advanced** tab.
3.  Paste the following configuration:
    ```nginx
    client_max_body_size 0;
    proxy_request_buffering off;
    proxy_buffering off;
    ```
4.  Save and apply.

Without this, NPM will attempt to buffer entire large files in memory or temp storage before sending them to the API, heavily degrading performance and potentially dropping connections for large payloads.

---

## 🗄️ Database & Storage Access (Local Dev)

### Connect to MongoDB

You can connect directly to the local MongoDB instance managing FileX metadata:

- **Connection String**: `mongodb://localhost:27017/?replicaSet=rs0`
- **Container Exec**:
  ```bash
  docker exec -it filex-mongo mongosh
  ```

### Accessing MinIO (Object Storage)

MinIO stores the actual encrypted file chunks and complete blobs.

- **Console UI**: [http://localhost:9001](http://localhost:9001)
- **Default Credentials** (from dev compose):
  - **Username**: `minioadmin`
  - **Password**: `minioadmin`

---

## 🛠 Troubleshooting

### API Connection Issues / CORS

Ensure `NEXT_PUBLIC_API_URL` dynamically matches your proxy set up if not using the default Next.js dev server. By default in dev, the client sends requests explicitly to `http://localhost:8080`.

### Background Worker Operations

If files aren't expiring or multipart uploads appear stuck, check the worker logs explicitly:

```bash
# Check garbage collection
docker-compose -f docker-compose.dev.yml logs -f worker-gc

# Check expiry process
docker-compose -f docker-compose.dev.yml logs -f worker-expiry

# Check multipart assembly
docker-compose -f docker-compose.dev.yml logs -f worker-multipart
```

### Complete Reset

If you need to completely wipe the local state (MongoDB data and MinIO storage volumes):

```bash
docker-compose -f docker-compose.dev.yml down -v
docker-compose -f docker-compose.dev.yml up --build -d
```
