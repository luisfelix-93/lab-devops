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
    -   Responsibility: Handle code execution in external environments (in this case, Docker containers).
    -   Technologies: `Docker Engine`.

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
    -   The API will be available at `http://localhost:8080`.
    -   The LocalStack UI will be available at `http://localhost:4566`.

## Project Structure

```
.
├── cmd/lab-api/main.go     # Application entry point
├── data/                     # Persistent data (SQLite database, temporary files)
├── db/migrations/            # Database migrations
├── internal/                 # Main application source code
│   ├── api/                  # API handlers and routes
│   ├── domain/               # Domain data structures
│   ├── executor/             # Logic for executing code in containers
│   ├── repository/           # Database access logic
│   └── service/              # Business logic
├── docker-compose.yaml       # Service orchestration (API + LocalStack)
└── Dockerfile                # Instructions for building the API image
```

## API Endpoints

### Get Lab Details

Returns the details of a specific lab and the user's last workspace state.

-   **URL**: `/api/v1/labs/:labID`
-   **Method**: `GET`
-   **Example**:
    ```bash
    curl http://localhost:8080/api/v1/labs/lab-tf-01
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
-   **Server Messages**:
    -   `{"type": "log", "payload": "..."}`: An execution log line.
    -   `{"type": "error", "payload": "..."}`: An error message.
    -   `{"type": "complete", "payload": "..."}`: Completion message.

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

### November 17, 2025
- **Workspace Status Tracking**: Added the ability to track workspace completion status. After successful lab execution, the workspace is marked as `"complete"`.
- **Enhanced State Management**: The `WorkspaceID` is now properly captured and used when saving the final state of a lab execution.
- **Ansible Support**: Full support for executing Ansible playbooks with dynamic inventory configuration and Docker isolation.

### November 22, 2025
- **Kubernetes Support**: Added support for running Kubernetes labs using K3s.
- **CI/CD Pipelines**: Implemented GitHub Actions for:
    - Automatic Pull Request creation on branch push.
    - Automated Docker build and push on PR merge.
- **Stability Improvements**: Fixed bugs in the Docker executor and improved logging.

### November 20, 2025
- **Learning Tracks**: Introduced structured learning paths (Tracks) to organize labs sequentially.
- **Performance**: Optimized Docker builds using BuildKit cache mounts for faster iteration.


## How to Contribute

Contributions are welcome! Feel free to open an *issue* to report bugs or suggest new features. If you wish to contribute code, please open a *Pull Request*.