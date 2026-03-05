# Lab Native GitOps & Configuration Specification

This document defines the architecture and manifest schema for the **Native GitOps Engine** in the Lab platform. This system provides a built-in reconciliation loop to manage both infrastructure (VMs/Stacks) and software configuration (Packages/Builds/Services) from a Git repository.

---

## 1. Architectural Overview

The Lab API acts as a **Continuous Reconciliation Controller**. It eliminates the need for external tools like Pulumi or Ansible by baking the "Desired State vs. Actual State" logic directly into the Go backend.

### The Reconciliation Loop
1. **Fetch:** Pulls the target Git repository (via `go-git`).
2. **Parse:** Recursively reads all `.yaml` files in the repository.
3. **Diff:** Compares the YAML definitions against the SQLite database and Libvirt state.
4. **Provision (Hardware):** Creates or updates the VM/Container/Network.
5. **Configure (Software):** Connects via agentless SSH to apply the "Playbook" defined in the YAML.
6. **Report:** Streams live logs and sync status (Healthy/Out-of-Sync/Failed) to the Lab UI.

---

## 2. Manifest Schema

Resources are defined in YAML. The system supports a modular directory structure (e.g., `vms/`, `networks/`, `roles/`).

### Example: `vms/app-server.yaml`
```yaml
kind: VirtualMachine
metadata:
  name: "production-web-01"
  description: "Main application server built from source"

spec:
  # --- Stage 1: Hardware (The "Rig") ---
  hardware:
    cpu: 4
    memory: 8GB
    disks:
      - size: 50GB
        pool: ssd-storage
    network:
      bridge: vmbr0
      type: bridge

  # --- Stage 2: Initial Boot (Cloud-Init) ---
  users:
    - name: "martin"
      groups: ["sudo", "developers"]
      ssh_keys: ["ssh-ed25519 AAA..."]

  # --- Stage 3: Configuration (The "Playbook") ---
  provision:
    # Abstracted Package Management
    packages:
      - name: "build-essential"
      - name: "cmake"
      - name: "git"

    tasks:
      - name: "Clone Source Code"
        module: git
        args:
          repo: "https://github.com/example/app.git"
          dest: "/usr/local/src/app"
        register: "repo_state" # Tracks if the commit hash changed

      - name: "Build from Source"
        module: shell
        args:
          cmd: "mkdir -p build && cd build && cmake .. && make install"
          chdir: "/usr/local/src/app"
          creates: "/usr/local/bin/app-binary"
        when: "repo_state.changed" # Only run if code was updated

      - name: "Setup Systemd Service"
        module: template
        args:
          dest: "/etc/systemd/system/app.service"
          content: |
            [Unit]
            Description=My Custom App
            [Service]
            ExecStart=/usr/local/bin/app-binary
            Restart=always
            [Install]
            WantedBy=multi-user.target
        notify: "Reload Systemd"

    handlers:
      - name: "Reload Systemd"
        module: shell
        args:
          cmd: "systemctl daemon-reload && systemctl enable --now app"
```

---

## 3. The Configuration Engine (Idempotency)

The engine ensures that running the same manifest multiple times results in the same state without redundant work.

### State Tracking (SQLite)
The Lab API tracks every task execution in a `git_task_state` table:
- **Task ID:** Hash of `(VM_ID, Task_Name, Module, Args)`.
- **Config Hash:** Hash of the content/version being applied (e.g., File content or Git Commit).
- **Status:** `success`, `changed`, or `failed`.

### Execution Logic
1. **Task Hashing:** The engine hashes the task definition.
2. **Database Lookup:** If the hash matches a "Success" entry in the DB, the task is **Skipped**.
3. **Success Proofs (`creates`):** For shell tasks, if the file specified in `creates` exists, the task is **Skipped**.
4. **Change Propagation:** If a task returns "Changed", it triggers any tasks that depend on it (via `when`) and queues any `handlers` it `notified`.

---

## 4. Force & Clean Mechanics

To support debugging and "tinkering" workflows, the engine supports three sync modes:

| Mode | Trigger | Behavior |
| :--- | :--- | :--- |
| **Normal** | `lab sync` | Skips all tasks that match the Database hash. Fastest path. |
| **Force** | `lab sync --force` | Ignores the Database hash. Re-runs the "Check" phase of every module (re-calculates checksums/states). |
| **Clean** | `lab sync --clean` | Deletes all task history for the VM and re-runs the entire manifest as if it's the first time. |
| **Task-Level Force** | `force: true` | A property in the YAML that makes a specific task run on every single sync regardless of mode. |

---

## 5. Supported Modules

### Infrastructure
- **`vm`**: Manage CPU, RAM, Disks, and Power State.
- **`network`**: Manage Bridges and NAT networks.
- **`stack`**: Manage Docker Compose stacks (Compose + Env files).

### Configuration (Agentless SSH)
- **`packages`**: Abstracted wrapper for `apt`, `pacman`, and `apk`.
- **`file`**: Manage permissions, ownership, and directories.
- **`template`**: Upload files with Jinja2-style variable injection.
- **`git`**: Clone and update repositories with commit-tracking.
- **`shell`**: Execute arbitrary commands with `creates`/`removes` idempotency guards.
- **`service`**: Manage `systemd` (or `openrc`) unit states.

---

## 6. Implementation Phases

1. **Phase 8.1: GitOps Core** — Repository watcher, YAML parser, and SQLite state tracking.
2. **Phase 8.2: Infrastructure Reconciler** — Mapping YAML `spec` to existing Libvirt/Stack services.
3. **Phase 8.3: Configuration Engine** — SSH orchestrator and the first set of modules (`packages`, `shell`, `file`).
4. **Phase 8.4: Advanced Config** — `template` engine, `handlers`, and `register/when` logic.
