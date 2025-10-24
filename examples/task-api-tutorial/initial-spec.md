# Task Management API - Requirements Specification

## Project Overview

Build a RESTful API for managing tasks using Go and SQLite. The API should support creating, reading, updating, and deleting tasks with proper validation and error handling.

## Functional Requirements

### 1. Data Model

A **Task** should have the following fields:

- `id` (integer, primary key, auto-increment)
- `title` (string, required, max 200 characters)
- `description` (string, optional, max 1000 characters)
- `status` (enum: "todo", "in_progress", "done", default: "todo")
- `priority` (enum: "low", "medium", "high", default: "medium")
- `created_at` (timestamp, auto-generated)
- `updated_at` (timestamp, auto-updated)

### 2. API Endpoints

#### GET /api/tasks
- **Description**: Retrieve all tasks
- **Response**: 200 OK with array of tasks
- **Example Response**:
  ```json
  {
    "tasks": [
      {
        "id": 1,
        "title": "Implement user authentication",
        "description": "Add JWT-based auth to the API",
        "status": "in_progress",
        "priority": "high",
        "created_at": "2025-01-15T10:00:00Z",
        "updated_at": "2025-01-15T14:30:00Z"
      }
    ]
  }
  ```

#### GET /api/tasks/:id
- **Description**: Retrieve a specific task by ID
- **Response**:
  - 200 OK with task object
  - 404 Not Found if task doesn't exist
- **Example Response**:
  ```json
  {
    "id": 1,
    "title": "Implement user authentication",
    "description": "Add JWT-based auth to the API",
    "status": "in_progress",
    "priority": "high",
    "created_at": "2025-01-15T10:00:00Z",
    "updated_at": "2025-01-15T14:30:00Z"
  }
  ```

#### POST /api/tasks
- **Description**: Create a new task
- **Request Body**:
  ```json
  {
    "title": "Fix database connection pool",
    "description": "Connection pool is leaking connections",
    "priority": "high"
  }
  ```
- **Validation**:
  - `title` is required and must be 1-200 characters
  - `description` is optional, max 1000 characters
  - `status` must be one of: "todo", "in_progress", "done"
  - `priority` must be one of: "low", "medium", "high"
- **Response**:
  - 201 Created with created task
  - 400 Bad Request if validation fails
- **Example Response**:
  ```json
  {
    "id": 2,
    "title": "Fix database connection pool",
    "description": "Connection pool is leaking connections",
    "status": "todo",
    "priority": "high",
    "created_at": "2025-01-15T15:00:00Z",
    "updated_at": "2025-01-15T15:00:00Z"
  }
  ```

#### PUT /api/tasks/:id
- **Description**: Update an existing task
- **Request Body**: Any combination of updatable fields
  ```json
  {
    "status": "done",
    "description": "Updated description"
  }
  ```
- **Response**:
  - 200 OK with updated task
  - 404 Not Found if task doesn't exist
  - 400 Bad Request if validation fails

#### DELETE /api/tasks/:id
- **Description**: Delete a task
- **Response**:
  - 204 No Content on success
  - 404 Not Found if task doesn't exist

### 3. Error Handling

All errors should return proper HTTP status codes with JSON error messages:

```json
{
  "error": "Task not found",
  "code": 404
}
```

Common error codes:
- `400` - Bad Request (validation errors, malformed JSON)
- `404` - Not Found (task doesn't exist)
- `500` - Internal Server Error (database errors, unexpected failures)

### 4. Database

- Use **SQLite** for simplicity
- Database file: `tasks.db`
- Implement a migration system to create the schema on first run
- Use prepared statements to prevent SQL injection

## Non-Functional Requirements

### Code Organization

```
task-api/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── database/
│   │   ├── db.go                # Database connection and initialization
│   │   ├── migrations.go        # Schema migrations
│   │   └── queries.go           # SQL query functions
│   ├── handlers/
│   │   ├── tasks.go             # HTTP request handlers
│   │   └── middleware.go        # Error handling, logging
│   └── models/
│       └── task.go              # Task struct and validation
├── tests/
│   ├── handlers_test.go         # API endpoint tests
│   └── database_test.go         # Database operation tests
├── go.mod
└── README.md
```

### Testing

- Write unit tests for all handlers
- Write integration tests for database operations
- Test all error cases (validation failures, not found, etc.)
- Achieve at least 80% code coverage

### Performance

- Support at least 100 concurrent requests
- Database queries should use prepared statements
- Implement connection pooling

### Code Quality

- Follow Go best practices and idioms
- Use meaningful variable and function names
- Add comments for exported functions
- Handle errors properly (no silent failures)
- Use proper HTTP status codes

## Acceptance Criteria

The project is complete when:

1. ✅ All 5 API endpoints are implemented and working
2. ✅ Database schema is created automatically on first run
3. ✅ All validation rules are enforced
4. ✅ Errors return proper HTTP status codes and JSON messages
5. ✅ Unit tests pass with >80% coverage
6. ✅ API can be tested with curl or Postman
7. ✅ Code is well-organized following the specified structure
8. ✅ README.md documents how to build and run the API

## Example Usage

Once implemented, you should be able to interact with the API like this:

```bash
# Start the server
go run cmd/api/main.go

# Create a task
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Learn GoCode", "priority": "high"}'

# Get all tasks
curl http://localhost:8080/api/tasks

# Update a task
curl -X PUT http://localhost:8080/api/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"status": "done"}'

# Delete a task
curl -X DELETE http://localhost:8080/api/tasks/1

# Run tests
go test ./...
```

## Notes for Implementation

- Start with the basic structure (go.mod, main.go)
- Implement the database layer first (connection, migrations, queries)
- Then build the models with validation
- Implement handlers one endpoint at a time
- Add tests after each handler is working
- Refactor and improve error handling at the end

This specification provides a clear, achievable scope for demonstrating GoCode's capabilities while building a real, usable API.
