Sentinel Auth Server

Sentinel is a production-grade, stateless authentication server built in Go. It implements high-security standards including RBAC (Role-Based Access Control), MFA (TOTP), and Audit Logging, following Clean Architecture principles.

Architecture

The system is designed as a Modular Monolith, decoupling business logic from external frameworks.

graph TD
    Client[Client App/Frontend] -->|JSON/HTTP| Handler[HTTP Delivery Layer]
    
    subgraph "Sentinel Auth Core"
        Handler --> Usecase[Auth Usecase]
        Usecase -->|Interface| UserRepo[User Repository]
        Usecase -->|Interface| TokenRepo[Token Repository]
        Usecase -->|Utility| Security[Security Pkg (Argon2/JWT)]
    end

    subgraph "Infrastructure"
        UserRepo -->|SQL| Postgres[(PostgreSQL)]
        TokenRepo -->|RESP| Redis[(Redis Cache)]
    end

    Usecase --> MFA[MFA Provider (TOTP)]


-->Key Features

-->Robust Authentication: Uses Argon2id for password hashing (resistant to GPU/ASIC attacks).

-->Hybrid Token System:

Access Tokens: Short-lived JWTs (Stateless) for microservices authorization.

Refresh Tokens: Opaque strings stored in Redis (Stateful) allowing immediate revocation.

-->Multi-Factor Authentication: Full support for TOTP (Google Authenticator, Authy).

-->RBAC: Granular permission system (Roles -> Permissions).

-->Audit Logs: Immutable history of all security events (Login successes, failures, MFA challenges).

-->High Performance: Optimized SQL queries avoiding N+1 problems.

-->Getting Started

Prerequisites

Docker & Docker Compose

Go 1.21+ (Optional, for local debugging)

1. Clone the Repository

git clone [https://github.com/FilipeAphrody/sentinel-auth.git](https://github.com/FilipeAphrody/sentinel-auth.git)
cd sentinel-auth


2. Start Infrastructure

We use Docker to orchestrate the API, PostgreSQL, and Redis.

docker-compose up -d


3. Initialize Database

Apply the schema to create tables and seed default roles (admin, user).

# Windows (PowerShell)
docker exec -i sentinel-db psql -U user -d sentinel < schema.sql

# Linux/Mac
cat schema.sql | docker exec -i sentinel-db psql -U user -d sentinel


4. Verify Installation

The server will start on port 8080.

curl http://localhost:8080/health
# Output: {"status":"healthy","version":"1.0.0",...}


-->Project Structure

This project follows strict Clean Architecture:

sentinel-auth/
├── cmd/api/            # Application Entry Point (Wiring)
├── internal/
│   ├── domain/         # Entities & Interface Definitions (Pure Go)
│   ├── usecase/        # Business Logic (Auth flows, MFA rules)
│   ├── repository/     # Database Implementations (Postgres/Redis)
│   └── delivery/http/  # HTTP Handlers (Echo Framework)
├── pkg/security/       # Cryptographic Utilities (Argon2, TOTP, JWT)
└── schema.sql          # Database Migration


-->Technical Decisions

Decision

Choice

Justification

Language

Go

Native concurrency, type safety, and small footprint.

Architecture

Clean Arch

Allows switching DBs or Frameworks without touching business logic.

Database

PostgreSQL

ACID compliance is non-negotiable for identity management.

Hashing

Argon2id

Superior to BCrypt against hardware-based brute force attacks.

Session

Redis + Opaque

Allows instant global logout (security requirement) which pure JWTs cannot do.

-->API Endpoints

Method

Path

Description

POST

/v1/login

Authenticate user. Returns tokens or 202 Accepted if MFA required.

POST

/v1/mfa/verify

Verify TOTP code to complete login.

POST

/v1/mfa/setup

Generate a new QR Code for 2FA enrollment.

POST

/v1/mfa/enable

Confirm and enable 2FA for the account.

GET

/health

Liveness check for Kubernetes/Load Balancers.

Developed by Filipe Aphrody