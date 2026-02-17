# Lab DevOps API

The **Lab DevOps API** is an interactive learning environment for practicing DevOps skills, such as Terraform and other IaC (Infrastructure as Code) tools. The platform allows users to execute "labs" in a simulated and secure environment, receiving real-time feedback.

## Overview

This project is the backend of the platform. It provides a REST API and a WebSocket endpoint to:
- Get lab details.
- Execute lab code in an isolated environment.
- Save user progress.

The execution environment is simulated using **LocalStack** to provision cloud resources (like S3, Lambda, etc.) locally, and **Docker** to isolate each lab's execution.

## Features

- **IaC & Automation Code Execution**: Supports **Terraform** and **Ansible**, with an extensible architecture for other tools.
- **Real-time Feedback**: Execution logs are streamed via WebSocket.
- **Simulated Environment**: Uses LocalStack to simulate cloud services, cost-free and securely.
- **Data Persistence**: User progress, labs, and workspace status are saved in an SQLite database.
- **Docker Isolation**: Each lab execution runs in a temporary Docker container.
- **Workspace Status Tracking**: Tracks the completion status of labs, enabling user progress validation.
- **Automatic Validation**: Automatically triggers solution validation upon successful code execution.

## Architecture

The project follows a layered architecture for separation of concerns:

1.  **Presentation Layer (API)**:
    -   Location: `internal/api/`
    -   Responsibility: Manage HTTP requests and WebSocket communication.
    -   Technologies: `Echo` (web framework), `Gorilla WebSocket`.

2.  **Business Logic Layer (Service)**:
    -   Location: `internal/service/`
    -   Responsibility: Orchestrate business logic, such as fetching a lab, initiating execution, and saving state.

3.  **Data Access Layer (Repository)**:
    -   Location: `internal/repository/`
    -   Responsibility: Abstract database access.
    -   Technologies: `SQLite`.

4.  **Execution Layer (Executor)**:
    -   Location: `internal/executor/`
    -   Responsibility: Handle code execution in external environments using a **Session Manager** pattern — a long-lived Docker container is created and steps (execution + validation) are run via `docker exec`.
    -   Technologies: `Docker Engine SDK`, retry logic, `stdcopy` log demultiplexing.

## Supported Lab Types

The platform currently supports those types of labs:

### Terraform Labs
Execute Infrastructure as Code (IaC) using Terraform. Users can provision cloud resources in the simulated LocalStack environment.

### Ansible Labs
Execute automation playbooks using Ansible. The system:
- Dynamically creates `playbook.yml` with user-provided code
- Generates an `inventory.ini` file for localhost configuration
- Runs playbooks in isolation using the `cytopia/ansible:latest` Docker image
- Enables communication with other services (e.g., LocalStack) on the Docker network
- **Auto Validation**: Runs `ansible-playbook validation.yml` automatically if provided.

### Kubernetes Labs
Execute Kubernetes manifests in a lightweight K3s cluster. The system:
- Provisions a K3s cluster in a Docker container
- Exposes the cluster API securely
- Automatically configures `kubeconfig` for the execution environment
- Supports `kubectl` commands in an isolated environment

## How to Run the Project

The simplest way to run the project is using `docker-compose`.

**Prerequisites**:
- Docker
- Docker Compose

**Steps**:

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/your-username/lab-devops.git
    cd lab-devops
    ```

2.  **Start the services**:
    ```bash
    docker-compose up --build
    ```
    This command will:
    -   Build the API's Docker image.
    -   Start a container for the API.
    -   Start a container for LocalStack (cloud simulator).
    -   Create a Docker network for container communication.

3.  **Access the API**:
    -   The API will be available at `http://localhost:8081` (Created generic config via env vars).
    -   The LocalStack UI will be available at `http://localhost:4566`.

## Configuration (Environment Variables)

The application is configured using environment variables. The following are the supported variables and their default values:

| Variable          | Default                                 | Description                                      |
| ----------------- | --------------------------------------- | ------------------------------------------------ |
| `DB_PATH`         | `./data/lab.db`                         | Path to the SQLite database file.                |
| `MIGRATIONS_PATH` | `./db/migrations/001_init_schema.sql`   | Path to the SQL file for database initialization.|
| `DOCKER_NETWORK`  | `minha-rede-lab`                        | Docker network for container communication.      |
| `TEMP_DIR_ROOT`   | `/app/data/temp-exec`                   | Directory for temporary execution files.         |
| `SERVER_PORT`     | `:8080`                                 | Port the Go server listens on (inside container).|

## API Endpoints

### Get Lab Details

Returns the details of a specific lab and the user's last workspace state.

-   **URL**: `/api/v1/labs/:labID`
-   **Method**: `GET`
-   **Example**:
    ```bash
    curl http://localhost:8081/api/v1/labs/lab-tf-01
    ```

### Execute a Lab

Initiates a WebSocket connection to execute lab code and receive real-time logs.

-   **URL**: `/api/v1/labs/:labID/execute`
-   **Protocol**: `WebSocket`
-   **Client Message (to start execution)**:
    ```json
    {
      "action": "execute",
      "user_code": "resource \"aws_s3_bucket\" \"my_bucket\" { ... }"
    }
    ```
    **Note**: Upon successful execution (exit code 0), the server **automatically** initiates the validation process without requiring a separate request.

-   **Client Message (manual validation - optional)**:
    ```json
    {
      "action": "validate"
    }
    ```
-   **Server Messages**:
    -   `{"type": "log", "payload": "..."}`: An execution log line (often with emojis like ✅/❌).
    -   `{"type": "error", "payload": "..."}`: An error message.
    -   `{"type": "complete", "payload": "..."}`: Completion message.

### Health Check

Returns the health status of the application and its dependencies (Database, Disk).

-   **URL**: `/api/v1/health`
-   **Method**: `GET`
-   **Response**:
    ```json
    {
      "status": "ok",
      "checks": {
        "database": "ok",
        "disk": "ok"
      },
      "timestamp": "2026-02-17T10:00:00Z"
    }
    ```

## Database

The project uses **SQLite** as its database. The database file is created at `./data/lab.db`.

Schema migrations are located in `db/migrations/`. The `001_init_schema.sql` file contains the initial structure for the `labs` and `workspaces` tables.

### Database Schema

The `workspaces` table includes:
- `id`: Unique identifier for the workspace
- `lab_id`: Reference to the associated lab
- `user_id`: Reference to the user
- `status`: Workspace completion status (`in_progress` or `complete`)
- `state`: JSON representation of the workspace state

This schema enables tracking of user progress and validation of lab submissions.

## Recent Updates

### February 14, 2026
- **Session Manager Executor**: Complete rewrite of the execution engine. Containers are now long-lived, with execution and validation steps running via `docker exec` instead of one-shot runs.
- **Retry Logic**: Container creation now retries up to 3 times with progressive delays, handling Docker Desktop + WSL2 filesystem sync race conditions.
- **K8s Validation Retry**: Kubernetes lab validation now polls for up to 30 seconds (every 2s), waiting for resources to become ready.
- **Handler Simplification**: The WebSocket handler no longer orchestrates validation — it's a passive consumer of execution results. Removed two-phase `isValidation`/`shouldValidateAfter` flow.
- **Expanded Domain Contract**: `ExecutionFinalState` now carries separate `ExecutionResult` and `ValidationResult` fields for granular inspection.

### December 07, 2025
- **Automatic Validation**: The system now automatically triggers validation after a successful execution request.
- **Environment Variables**: Full configuration via environment variables for better portability.
- **API Port Change**: Docker Compose now exposes the API on port `8081` by default.

### November 29, 2025
- **Lab & Track Management**: Added full CRUD capabilities for Labs and Tracks.
    - New API endpoints for updating (`PATCH`) and deleting (`DELETE`) labs and tracks.
    - Enhanced `LabService` to handle these operations securely.
- **Bug Fixes**: Resolved issues in the Docker executor to ensure reliable lab execution.
- **Infrastructure**: Initial configuration improvements and Dockerfile optimizations (including LocalStack network aliases).


## How to Contribute

Contributions are welcome! Feel free to open an *issue* to report bugs or suggest new features. If you wish to contribute code, please open a *Pull Request*.