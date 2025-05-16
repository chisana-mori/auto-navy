# System Patterns

## Architecture Patterns
- **MVC Pattern**: Backend follows the Model-View-Controller pattern
- **Service Layer**: Business logic is encapsulated in service layers
- **Repository Pattern**: Data access is abstracted through repositories (via GORM)
- **DTO Pattern**: Data Transfer Objects for communication between layers

## Code Organization
- **Package by Feature**: Code is organized by feature/module rather than by layer
- **Clean Architecture**: Dependencies point inward, with domain at the center
- **Separation of Concerns**: Clear separation between models, services, and controllers

## Naming Conventions
- **Go Naming**: Follow standard Go naming conventions (camelCase for unexported, PascalCase for exported)
- **File Naming**: Service DTOs follow `${service}_dto.go` pattern
- **API Endpoints**: RESTful naming conventions for API endpoints
- **JSON Format**: Camel case for JSON field names in API communication

## Error Handling
- **Centralized Error Handling**: Common error handling mechanisms
- **Error Types**: Structured error responses with types and messages
- **Logging**: Consistent logging patterns for errors and operations

## Testing Patterns
- **Unit Testing**: For individual components
- **Integration Testing**: For API endpoints and service interactions
- **Mock Objects**: For isolating dependencies in tests

## Frontend Patterns
- **Component-Based Architecture**: React components for UI building blocks
- **State Management**: Local component state and global state management
- **API Service Layer**: Centralized API communication

## Security Patterns
- **Authentication**: JWT or similar token-based authentication
- **Authorization**: Role-based access control
- **Input Validation**: Thorough validation of all inputs
- **Security Checks**: Regular security configuration checks

## Performance Patterns
- **Caching**: Strategic caching for frequently accessed data
- **Pagination**: For large data sets
- **Asynchronous Processing**: For long-running tasks via job system 