# Usage Guide

This guide covers the high-level workflow for Winops. Screenshots referenced here can be placed in `docs/assets/`.

## First Launch

1. Launch `Winops.exe` as Administrator for full remediation capabilities.
2. The Dashboard loads a live snapshot of your system (CPU, memory, disk, network, alerts).
3. If you plan to use the AI Assistant, ensure [Ollama](https://ollama.com) is running locally.

## Modules at a Glance

### SysOps
- **Overview/Hardware**: Inspect CPU, motherboard, RAM, storage, GPU, battery, and sensors.
- **Processes/Services**: View running processes, manage Windows services, and inspect startup entries.
- **Logs/Diagnostics**: Review system/application logs and run quick diagnostics.

### NetOps
- **Interfaces/Routes/DNS**: Inventory adapters, routing table, DNS configuration.
- **Discovery/Scan**: Discover devices on your LAN and run targeted port scans.
- **Connectivity**: Ping, traceroute/multi-ping, and Wi-Fi details.
- **Topology/Health**: Visualize network layout and run network health checks.

### SecOps
- **Security Score**: Get a high-level posture score with actionable recommendations.
- **Firewall/Defender**: Inspect rules, toggle allowed/blocked states, and check Defender status.
- **Hardening/Identity**: Review hardening checks, password policy, and local accounts.
- **Forensics/Response**: Capture forensic snapshots and execute response actions (use with caution).

### DevOps
- **Containers**: Manage Docker containers/images/networks/volumes where available.
- **Kubernetes**: Inspect clusters, pods, services, and common resources.
- **Packages/Environment**: Review installed packages and environment variables.

### AIOps
- **Chat**: Ask grounded questions about your system; the assistant can incorporate live context.
- **Persona/Models**: Create or switch models/personas. With Ollama, you can pull models and build a custom `winops` persona.
- **Reports**: Generate AI-assisted operational reports from captured data.

## Tips & Safety

- **Admin recommended**: Many SecOps/DevOps actions require elevated privileges.
- **Local-only by design**: Avoid exporting sensitive logs to untrusted locations.
- **Workflows**: Use the Workflows/Reports tabs to standardize recurring checks.
- **Ollama offline**: If Ollama is unreachable, AIOps will report status; other modules remain fully functional.
- **AI is optional**: Winops works fully without any AI backend. To enable AI, install Ollama or a llama.cpp `llama-server.exe` yourself (see README "Optional: local AI") and select a `.gguf` model in Settings → Models. Winops never downloads a runtime or model automatically.
