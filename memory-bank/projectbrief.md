# Navy-NG Project Brief

## Project Overview
Navy-NG is a front-end and back-end integrated project. The backend is implemented in Golang using the Gin web framework, while the frontend is implemented in React.

## Key Components
- Backend: Golang with Gin framework
- Frontend: React
- Database: Uses GORM for database operations

## Project Structure
The project follows a specific directory structure as outlined in the project requirements:

```
navy-ng/
├── models/                # Data model layer - using GORM for database operations
│   └── portal/            # Portal module data models
│       └── object.go      # Core data model definitions
│
├── pkg/                   # Public library code - can be referenced by external projects
│   └── middleware/        # Middleware
│       └── render         # Controller layer rendering related methods
│            └── json.go   # Rendering methods
│
├── server/                # Backend service layer
│   └── portal/            # Portal module services
│       └── internal/      # Internal implementation (not exposed externally)
│           ├── main.go    # Service startup entry
│           ├── conf/      # Configuration management (environment variables/config files)
│           ├── docs/      # Swagger documentation
│           ├── routers/   # Gin route definitions and controllers
│           └── service/   # Business logic implementation
│
├── job/                   # Inspection job tasks
│   └── email/             # Task job collection, each job corresponds to sending one type of email
├── web/                   # Frontend application layer
│   └── navy-fe/           # React frontend project
│
├── scripts/               # Development and operations scripts
```

## Development Guidelines
- Backend code follows Java MVC design pattern
- Frontend parameter model and database layer model should be separated
- Service layer should build separate DTO models with filenames using the template `${service}_dto`
- All response code in controller classes must use pkg/middleware/render/json.go
- JSON for front-end and back-end communication uses camel case format
- Method complexity should not exceed 15 