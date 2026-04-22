# FileX v2🔐

**FileX** is a privacy-first file infrastructure system engineered for secure and encrypted file exchange 🛡️.

Designed with privacy and security at its core 🔒, FileX ensures that every file transfer and storage operation is protected through strong end-to-end encryption 🔐 and controlled, secure transfer mechanisms ⚙️. Whether you're sharing sensitive documents 📁, distributing internal assets 🏢, or building secure workflows 🔄, FileX keeps your data protected at every stage 🚀.

At its core, FileX represents **File Infrastructure for Locked Encrypted eXchange** 🔏 — where security is not an add-on, but the foundation 🧱.

## 🚀 Why FileX?

We built FileX on the principle of **Zero-Compromise Security**.

- **Privacy First 🛡️**: Files are encrypted client-side, and access is tightly controlled.
- **Scale Seamlessly 📈**: Built to handle massive files seamlessly using multipart uploads and S3-compatible object storage.
- **Automated Lifecycle ⏳**: Auto-expiring links and automated garbage collection ensure no orphaned data is left behind.
- **High Performance ⚡**: Powered with a Go backend and a React frontend with Vite for blazing-fast experiences.
- **Engineering Excellence 🛠️**: Utilizing background workers (goroutines) to handle the heavy lifting off the main API thread.

## ✨ Features

### 🔐 Uncompromising Security

- **End-to-End Encryption**: Files are encrypted client-side using AES-256-GCM before storage. The server never sees plaintext or encryption keys.
- **Argon2id Key Derivation**: Passphrases are strengthened with Argon2id (WASM) for GPU-resistant key derivation.
- **Chunk-Level AEAD**: Each chunk includes file ID and chunk index in additional authenticated data, preventing reordering attacks.
- **Privacy-focused**: Complete peace of mind knowing unauthorized users can't access your sensitive files.

### 📦 Robust File Handling

- **Multipart Uploads**: Efficiently handles huge files by chunking them into 10MB parts for reliable transfer and zero timeouts.
- **Concurrent Uploads**: Chunks are uploaded in parallel with progress tracking and integrity verification.
- **Seamless Downloads**: High-speed, streaming decryption directly to your device via presigned URLs.

### ⚙️ Automated Data Management

- **Link Expiry**: Automatically invalidate access to files after a set Time-to-Live (TTL: 30min, 1hr, 24hr).
- **Garbage Collection**: Background workers safely clean up orphaned or incomplete uploads to save storage.
- **Background Processing**: Dedicated goroutines handle expiry, GC, and multipart assemblies asynchronously.

### 🎨 Modern & Fast UI

- **Vite + React 19**: Fast development and optimized production builds with the latest React features.
- **TanStack Router**: Type-safe, file-based routing for excellent DX.
- **Tailwind CSS v4**: Utility-first CSS framework for rapid styling.
- **OGL**: Ultra-lightweight WebGL library for dynamic particle UI effects.

## 🛠️ The Tech Stack

FileX isn't just a file host; it's an architectural showcase of modern Go and React built for security and scale.

### **Frontend** (The Interface) 🎨

- **Vite 6**: Next-generation frontend build tool with hot module replacement.
- **React 19**: Latest React with concurrent features and improved rendering.
- **TanStack Router**: Type-safe routing with file-based route generation.
- **TanStack Query**: Powerful server state management.
- **Zustand**: Lightweight client state management.
- **Tailwind CSS v4**: Utility-first CSS framework for rapid styling.
- **OGL**: Ultra-lightweight WebGL library for dynamic UI elements.
- **Lucide React**: Beautiful, consistent icon set.
- **Argon2-browser**: WASM-based password hashing for key derivation.
- **fflate**: Compression utilities.
- **tus-js-client**: Resumable upload protocol support.

### **Backend** (The Engine) 🦍

- **Go 1.26**: Raw performance and robust concurrency for API handling.
- **MongoDB**: Primary database for file metadata, shares, and multipart sessions.
- **MinIO**: High-performance, S3-compatible object storage for securely preserving encrypted file data.
- **In-Memory Rate Limiting**: Sharded token bucket rate limiter (no Redis dependency for single-node deployments).
- **Background Workers**: Goroutines for Expiry, Multipart jobs, and Garbage Collection.

### **Infrastructure** 🏗️

- **Docker & Docker Compose**: Fully containerized setup for reproducible development and production environments.

## ⚡ Quick Start

Want to see it in action? You only need [Docker](https://www.docker.com/).

```bash
# Clone the repository
git clone https://github.com/pranavdhawale/bytefile.git
cd bytefile

# Start development environment 🚀
docker-compose -f docker-compose.dev.yml up --build -d
```

That's it! Everything boots up automatically.

- 🎨 **Frontend**: [http://localhost:3000](http://localhost:3000)
- ⚙️ **Backend API**: [http://localhost:8080](http://localhost:8080)
- 🪣 **MinIO Console**: [http://localhost:9001](http://localhost:9001)

## 🏗️ Architecture

FileX follows a robust **API Server + Background Workers** architecture for responsive, scalable file operations.

```
┌──────────────────────────────────────────────────────────────┐
│                    CLIENT (Vite/React 19)                    │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────────┐ │
│  │ crypto.ts │  │ upload.ts │  │ api.ts   │  │ TanStack     │ │
│  │ AES-GCM  │  │ multipart │  │ REST     │  │ Router/Zustand│ │
│  └──────────┘  └──────────┘  └──────────┘  └───────────────┘ │
└──────────────────────────────────────────────────────────────┘
                            │ HTTP/REST
                            ▼
┌──────────────────────────────────────────────────────────────┐
│                    API SERVER (Go 1.26)                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────────┐ │
│  │ handler/ │  │ ratelimit/│  │ counter/ │  │ middleware    │ │
│  │ endpoints│  │ in-memory │  │ download │  │ CORS, logging │ │
│  └──────────┘  └──────────┘  └──────────┘  └───────────────┘ │
│         │              │              │                       │
│         ▼              ▼              ▼                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐          │
│  │ MongoDB  │  │ MinIO    │  │ In-Memory Rate   │          │
│  │ (metadata)│ │ (blobs)  │  │ Limiter (64 shard)│          │
│  └──────────┘  └──────────┘  └──────────────────┘          │
└──────────────────────────────────────────────────────────────┘
                            │ goroutines
                            ▼
┌──────────────────────────────────────────────────────────────┐
│                  BACKGROUND WORKERS (same process)            │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐   │
│  │ worker/expiry │  │ worker/gc    │  │ worker/multipart   │   │
│  │ Reap expired │  │ Cleanup      │  │ Assemble chunks    │   │
│  │ shares/files │  │ orphan files │  │ into complete      │   │
│  └──────────────┘  └──────────────┘  └────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

### Key Components

- **Client**: Vite SPA handling encryption, chunking, and user experience.
- **API Server**: Go server handling request validation, routing, rate limiting, and access control.
- **MinIO Storage**: Stores the encrypted file blobs securely with presigned URL downloads.
- **Background Workers**: Run as goroutines in the same process for:
  - `worker-expiry`: Actively reaps expired shares.
  - `worker-gc`: Identifies and removes orphaned chunks or failed uploads.
  - `worker-multipart`: Constructs multipart chunks into complete file objects.

## 📁 Project Structure

```
filex/
├── client/                 # Vite + React Front-End
│   ├── src/
│   │   ├── components/    # Reusable UI components
│   │   ├── hooks/         # React hooks (useUpload, useDownload)
│   │   ├── lib/           # Client utilities (crypto, api, upload, download)
│   │   ├── routes/        # TanStack Router file-based routes
│   │   ├── store/         # Zustand state management
│   │   └── main.tsx       # Application entry
│   ├── public/            # Static assets (argon2.wasm)
│   ├── package.json
│   └── Dockerfile
│
├── server/                # Go 1.26 Backend
│   ├── cmd/api/main.go    # Entrypoint for API server
│   └── internal/          # Core application logic
│       ├── api/           # HTTP handlers & routes
│       ├── config/        # Environment configuration
│       ├── counter/       # In-memory download counter
│       ├── database/      # MongoDB connection
│       ├── handler/       # Request handlers (file, share, health)
│       ├── models/        # Domain entities (File, Share, MultipartSession)
│       ├── ratelimit/     # Sharded token bucket rate limiter
│       ├── repository/    # Data access layer
│       ├── server/        # HTTP server setup
│       ├── slug/          # NanoID generation
│       ├── storage/       # MinIO S3 operations
│       └── worker/        # Background job processors
│
├── docs/                   # Documentation
├── docker-compose.dev.yml # Development orchestration
├── docker-compose.prod.yml# Production orchestration
├── SETUP_GUIDE.md         # Setup instructions
└── README.md              # This file
```

## 👩💻 Development

We use a modern dockerized workflow to spin up all moving parts smoothly and reliably.

### Commands

```bash
# Start development environment
docker-compose -f docker-compose.dev.yml up

# Rebuild containers
docker-compose -f docker-compose.dev.yml up --build

# Stop all services
docker-compose -f docker-compose.dev.yml down

# View logs
docker-compose -f docker-compose.dev.yml logs -f

# View specific service logs
docker-compose -f docker-compose.dev.yml logs -f filex
docker-compose -f docker-compose.dev.yml logs -f mongo
docker-compose -f docker-compose.dev.yml logs -f minio
```

### Local Development (without Docker)

```bash
# Backend
cd server
go mod download
go run ./cmd/api

# Frontend
cd client
npm install
npm run dev
```

## 🔒 Security Model

FileX implements **client-side end-to-end encryption**:

1. **Key Generation**: Client generates a random 256-bit File Encryption Key (FEK)
2. **Key Wrapping**: FEK is wrapped with a key derived from the user's passphrase using Argon2id
3. **Chunk Encryption**: Each 10MB chunk is encrypted with AES-256-GCM using:
   - Unique IV per chunk (based on chunk index)
   - AEAD = `fileId || chunkIndex` (prevents reordering attacks)
4. **Server Storage**: Server only stores encrypted data + wrapped FEK + salt — never plaintext or keys
5. **Download Flow**: Client unwraps FEK with passphrase, streams decryption chunk-by-chunk

The server is **zero-knowledge** — it cannot decrypt files even if compromised.

## 🔒 Privacy & Data

- **Encrypted at Rest**: Files are secured cryptographically before upload.
- **Ephemeral Access**: Expiry limits mean your files never sit available forever.
- **Self-Hosted Complete Control**: Keeps data in your hands entirely.
- **No Tracking**: Your transfers are your business alone. We don't track your behavior.
- **Open Source**: Full transparency in what code runs on your hardware.

## 🤝 Contributing

We ❤️ open source! If you have ideas, suggestions, or bug fixes, feel free to contribute.

1. Fork the repo 🍴
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request 📩

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Built with ❤️ for the community.**