# ASF Free Games Claimer (Golang Version)

A super lightweight, containerized Go application that automatically claims free Steam games posted by [/u/ASFinfo](https://www.reddit.com/user/ASFinfo) via [ArchiSteamFarm (ASF)](https://github.com/JustArchiNET/ArchiSteamFarm)'s IPC interface.

Compared to the Node.js version which creates a ~1.6 GB container image, this Golang rewrite results in a Docker image of only **~15 MB** (over a 99% reduction in size) with minimal RAM and CPU footprints.

---

## Features

- **Extremely Lightweight:** Compiled statically and runs inside a tiny Alpine Linux Docker container.
- **State Persistence:** Avoids re-claiming already processed licenses by storing the claim history count in a local file (`lastlength`).
- **Interactive Color Logs:** Beautiful ANSI-colored output (`[INFO]`, `[SUCCESS]`, `[WARN]`, `[ERROR]`) for easier monitoring.
- **Zero External Dependencies:** Built entirely using Go's standard library.

---

## How It Works

1. On startup (and every 6 hours thereafter), the program queries a public GitHub Gist maintained by `/u/ASFinfo` to fetch new Steam license/activation codes.
2. It compares the number of total available codes with the number saved in the local `lastlength` file.
3. If new codes are found, it generates a bulk `addlicense` command (limited to a maximum of the 40 most recent games to prevent rate limits) and sends it to ArchiSteamFarm's IPC.
4. If ASF reports a successful command execution, `lastlength` is updated. Otherwise, the changes are rolled back to try again during the next run.

---

## Prerequisites

- An active [ArchiSteamFarm (ASF)](https://github.com/JustArchiNET/ArchiSteamFarm) instance.
- **IPC Enabled** in ArchiSteamFarm (enabled by default; see the [ASF IPC documentation](https://github.com/JustArchiNET/ArchiSteamFarm/wiki/IPC) for more info).

---

## Deployment & Usage

### Method 1: Docker Compose (Recommended)

Create a `docker-compose.yml` file using the configuration below. This will use the pre-built image from Docker Hub:

```yaml
version: "3"

services:
  asf-claim:
    image: duvydev/asf-freegamesclaimer:latest
    container_name: asfclaim-go
    restart: unless-stopped
    volumes:
      # Persists the state file on the host to avoid re-claiming old licenses on container restarts
      - ./lastlength:/root/lastlength
    environment:
      - ASF_PORT=1242
      - ASF_HOST=your-asf-host-or-ip
      - ASF_PASSWORD=your_asf_ipc_password
      - ASF_COMMAND_PREFIX=!
      - ASF_HTTPS=false
      - ASF_BOTS=asf
```

Run the container:
```bash
docker compose up -d
```

### Method 2: Running Locally

1. Install Go (1.26 or later).
2. Clone the repository and navigate to the directory:
   ```bash
   git clone https://github.com/Pablo/ASF-FreeGamesClaimer.git
   cd ASF-FreeGamesClaimer
   ```
3. Copy the template `.env` file and configure your variables:
   ```bash
   cp .env.template .env
   # Edit .env file with your ASF IPC settings
   ```
4. Build and run:
   ```bash
   go build -o asfclaim main.go
   ./asfclaim
   ```

---

## Configuration Variables

You can configure the claimer using environment variables or a `.env` file in the working directory:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `ASF_PORT` | `1242` | The port your ASF IPC server is listening on. |
| `ASF_HOST` | `localhost` | Host name or IP address of your ASF IPC server. |
| `ASF_PASSWORD` | *(empty)* | IPC password configured in ASF (`IPCPassword`). Leave blank if not set. |
| `ASF_COMMAND_PREFIX` | `!` | Prefix for ASF commands (e.g. `!`, `/`). |
| `ASF_HTTPS` | `false` | Set to `true` if your ASF IPC endpoint uses HTTPS. |
| `ASF_BOTS` | `asf` | Target bots to run the command on (comma-separated or `asf` for all bots). |

---

## License

This project is licensed under the [MIT License](LICENSE).
