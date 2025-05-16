# Technical Context

## Backend Technology Stack
- **Language**: Go (Golang)
- **Web Framework**: Gin
- **ORM**: GORM
- **API Documentation**: Swagger

## Frontend Technology Stack
- **Framework**: React
- **Language**: TypeScript/JavaScript

## Database
- Uses GORM for database operations
- Database specifics to be determined based on project requirements

## Architecture
- MVC design pattern for backend
- Service layer with DTOs for data transfer between layers
- RESTful API communication between frontend and backend
- JSON in camel case format for API communication

## Key Components
### Models
- Data models using GORM
- Located in `models/portal/` directory

### Services
- Business logic implementation
- Located in `server/portal/internal/service/` directory
- Uses DTOs for data transfer

### Controllers
- API endpoints and request handling
- Located in `server/portal/internal/routers/` directory
- Uses `pkg/middleware/render/json.go` for responses

### Jobs
- Background tasks and operations
- Located in `job/` directory
- Email sending functionality in `job/email/` directory

## Development Environment
- Go modules for dependency management
- Make for build automation
- Golangci-lint for code quality 