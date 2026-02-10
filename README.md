# 🗄️ Code Vault

A powerful, self-hosted code snippet manager with versioning, deduplication, and expiration.

**"Pastebin for serious developers"**

![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## ✨ Features

- **📝 Snippet Management** - Store code snippets, configs, SQL queries, YAML files
- **🔄 Version Control** - Full version history with diff support
- **♻️ Smart Deduplication** - Content-addressed storage saves space automatically
- **⏰ Auto-Expiration** - Temporary snippets self-delete
- **🎨 Syntax Highlighting** - Beautiful colored output in terminal
- **🔍 Interactive UI** - Fuzzy finder with live preview
- **🏷️ Tags & Search** - Organize and find snippets easily - WIP ⚠
- **📋 Clipboard Support** - Copy snippets with one command

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 14+
- Docker & Docker Compose (recommended)

### Installation

1. **Clone the repository:**
```bash
git clone https://github.com/yourusername/code-vault.git
cd code-vault
```

2. **Start infrastructure (PostgreSQL + SeaweedFS):**
```bash
docker-compose up -d
```

3. **Build the CLI:**
```bash
go build -o vault ./cmd/vault
sudo mv vault /usr/local/bin/  # Optional: install globally
```

4. **Initialize:**
```bash
vault init
```

## 📖 Usage

### Basic Commands

**Create a snippet:**
```bash
vault create script.go --title "Hello World" --tags go,example
```

**List snippets:**
```bash
vault list
```

**Get a snippet (interactive):**
```bash
vault get
# Opens fuzzy finder with preview
```

**Get specific snippet:**
```bash
vault get <snippet-id>
vault get <snippet-id> --copy  # Copy to clipboard
vault get <snippet-id> -o file.go  # Save to file
```

### Versioning

**Update snippet (creates new version):**
```bash
vault update <snippet-id> script-v2.go
```

**View version history:**
```bash
vault versions <snippet-id>
```

**Compare versions:**
```bash
vault diff <snippet-id> --from 1 --to 3
```

**Get specific version:**
```bash
vault get <snippet-id> --version 2
```

### Expiration

**Create with expiration:**
```bash
vault create temp.sh --expires 24h
vault create config.yaml --expires 7d
```

**Manage expiration:**
```bash
vault expire <snippet-id> --set 30d
vault expire <snippet-id> --extend 7d
vault expire <snippet-id> --remove
```

**List expiring snippets:**
```bash
vault list --expiring-soon
vault list --expired
```

**Run cleanup worker:**
```bash
# Run once
vault worker --once

# Run continuously (checks every hour)
vault worker --interval 1h

# Background (recommended)
nohup vault worker --interval 1h > worker.log 2>&1 &
```

### Advanced

**Delete snippet:**
```bash
vault delete <snippet-id>
vault delete <snippet-id> --hard  # Also delete from storage
```

**Browse interactively:**
```bash
vault browse
```

## 🏗️ Architecture
```
┌─────────────┐
│   CLI Tool  │
└───┬─────────┘
    │
    ├────────────────┐
    │                │
┌───▼────────┐   ┌───▼──────┐
│ PostgreSQL │   │SeaweedFS │
│ (Metadata) │   │ (Content)│
└────────────┘   └──────────┘
```

- **PostgreSQL**: Stores metadata (titles, tags, versions, users)
- **SeaweedFS**: Stores actual file content (S3-compatible)
- **Deduplication**: SHA-256 content addressing

## ⚙️ Configuration

Configuration file: `~/.config/code-vault/config.yaml`
```yaml
seaweedfs:
  s3_endpoint: "http://localhost:8333"
  bucket_name: "code-vault"

database:
  host: "localhost"
  port: 5432
  user: "vault_user"
  password: "your_password"
  dbname: "code_vault"

user:
  username: "your_username"
```

## 🐳 Docker Setup

The included `docker-compose.yml` provides:
- PostgreSQL 16
- SeaweedFS (master, volume, filer, s3)
```bash
# Start services
docker-compose up -d

# Stop services
docker-compose down

# View logs
docker-compose logs -f
```

## 🔧 Development

**Run tests:**
```bash
go test ./...
```

**Build:**
```bash
go build -o vault ./cmd/vault
```

**Install dependencies:**
```bash
go mod download
```

## 📊 Storage Efficiency

Example savings with deduplication:
```
3 snippets with identical content:
Without deduplication: 3 × 10 KB = 30 KB
With deduplication:    1 × 10 KB = 10 KB
Savings: 66% 🎉
```

## 🤝 Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details

## 📮 Support

- GitHub Issues: [Report bugs or request features](https://github.com/yourusername/code-vault/issues)
- Discussions: [Ask questions](https://github.com/yourusername/code-vault/discussions)

---

**Made with ❤️ by [Your Name](https://github.com/yourusername)**
