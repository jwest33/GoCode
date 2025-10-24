# Task API Walkthrough - Step-by-Step Tutorial

Welcome! This guide will walk you through building a complete REST API using GoCode as your AI coding assistant. Follow each step carefully and watch how the agent helps you build production-ready code.

**Estimated time: 60 minutes**

---

## Prerequisites Checklist

Before starting, verify you have:

- [ ] Go 1.23+ installed (`go version`)
- [ ] gopls installed (`gopls version`)
- [ ] GoCode configured with model path
- [ ] Empty directory for the tutorial
- [ ] `.gocode-example-config.yaml` copied to your directory as `config.yaml`

---

## Phase 1: Setup & Basic Tools (~10 minutes)

**What you'll learn:** Basic GoCode tools (write, read, glob), how the agent creates project structure, todo list management

### Step 1.1: Start GoCode

```bash
cd task-api-tutorial  # Your empty project directory
gocode
```

You should see GoCode start up, possibly with a project initialization prompt. If asked, accept project analysis.

### Step 1.2: Show the Agent Your Requirements

**Prompt to type:**
```
I want to build a Task Management REST API in Go. I have a detailed specification in this file. Please read examples/task-api/initial-spec.md and create a plan for implementing this project.
```

**What to expect:**
- Agent will use the `read` tool to read initial-spec.md
- Agent will use the `todo_write` tool to create a task breakdown
- You'll see a TODO.md file appear in your directory
- Agent will outline the implementation phases

**Checkpoint:** Verify that TODO.md exists and contains a structured plan.

### Step 1.3: Initialize the Go Module

**Prompt to type:**
```
Let's start by initializing the Go module and creating the basic project structure as outlined in the spec.
```

**What to expect:**
- Agent will use `bash` tool to run `go mod init task-api`
- Agent will create directory structure (cmd/api/, internal/database/, etc.)
- Agent will use `write` tool to create a basic main.go
- Agent will mark the first todos as completed

**Checkpoint:**
- Run `ls` or `dir` - you should see: cmd/, internal/, go.mod, main.go
- Run `cat go.mod` - should show module name "task-api"

### Step 1.4: Review What Was Created

**Prompt to type:**
```
Show me the project structure you created. Use glob to list all files and read the main.go to explain what it does.
```

**What to expect:**
- Agent will use `glob` tool to find all files (e.g., `**/*.go`)
- Agent will use `read` tool to read main.go
- Agent will explain the structure

**What you learned:**
- ✅ How to give the agent context (showing it the spec)
- ✅ `read`, `write`, `bash`, `glob` tools in action
- ✅ How the agent breaks down tasks using `todo_write`

---

## Phase 2: API Implementation (~15 minutes)

**What you'll learn:** Multi-file navigation, code editing, how the agent handles complex multi-step tasks

### Step 2.1: Implement the Database Layer

**Prompt to type:**
```
Let's implement the database layer first. Create the database connection, migration system, and query functions in internal/database/ as specified in the requirements.
```

**What to expect:**
- Agent will create multiple files (db.go, migrations.go, queries.go)
- Agent will use `write` tool multiple times
- You may be prompted to approve each file creation
- Agent will update the todo list as it completes each file

**Checkpoint:**
- Check internal/database/ has 3 files
- Read one of them: `cat internal/database/db.go`
- Should see proper Go code with imports, functions, comments

### Step 2.2: Create the Task Model

**Prompt to type:**
```
Now create the Task model in internal/models/task.go with validation functions as described in the spec.
```

**What to expect:**
- Agent creates internal/models/task.go
- Includes Task struct with JSON tags
- Includes Validate() method checking title length, enum values, etc.

**Checkpoint:**
```bash
cat internal/models/task.go
```
Should see Task struct with all fields from spec (id, title, description, status, priority, timestamps)

### Step 2.3: Implement the First API Endpoint

**Prompt to type:**
```
Let's implement the GET /api/tasks endpoint in internal/handlers/tasks.go. Start with just this one endpoint and the necessary HTTP server setup.
```

**What to expect:**
- Agent creates handlers/tasks.go with GetAllTasks handler
- Agent updates main.go to set up HTTP routes
- May create middleware.go for error handling

**Checkpoint:**
```bash
cat cmd/api/main.go
```
Should see HTTP server setup with route registration

### Step 2.4: Implement Remaining CRUD Endpoints

**Prompt to type:**
```
Now implement the remaining CRUD endpoints:
- GET /api/tasks/:id
- POST /api/tasks
- PUT /api/tasks/:id
- DELETE /api/tasks/:id

Follow REST best practices with proper HTTP status codes as specified.
```

**What to expect:**
- Agent will use `edit` tool to add handlers to existing files
- Agent updates routes in main.go
- You'll see confirmations for each edit

**Important:** Watch how the agent uses `edit` tool vs `write` tool. Edit is for modifying existing files, write is for new files.

**Checkpoint:**
```bash
grep -n "func" internal/handlers/tasks.go
```
Should see 5 handler functions (GetAllTasks, GetTask, CreateTask, UpdateTask, DeleteTask)

**What you learned:**
- ✅ How agent handles multi-file tasks
- ✅ `edit` tool for modifying existing code
- ✅ Agent's ability to maintain code consistency
- ✅ How to break complex tasks into steps

---

## Phase 3: Database Integration (~10 minutes)

**What you'll learn:** Bash tool for dependencies, testing the code, fixing errors

### Step 3.1: Add Database Driver

**Prompt to type:**
```
Add the SQLite driver dependency and update the database code to use it properly.
```

**What to expect:**
- Agent runs `go get github.com/mattn/go-sqlite3`
- Agent may edit database code to import the driver
- Updates go.mod and go.sum

**Checkpoint:**
```bash
cat go.mod
```
Should see github.com/mattn/go-sqlite3 in require section

### Step 3.2: Test Database Connection

**Prompt to type:**
```
Let's test if the database connection works. Add some basic logging to main.go and try to build and run the application.
```

**What to expect:**
- Agent edits main.go to add initialization logging
- Agent runs `go build ./cmd/api`
- May encounter errors - this is expected!

**If errors occur** (common scenario):

**Prompt to type:**
```
There are build errors. Please read the error output and fix them.
```

**What to expect:**
- Agent reads the error messages
- Agent uses `edit` tool to fix issues (missing imports, syntax errors, etc.)
- Agent runs build again to verify

**Checkpoint:**
```bash
go build ./cmd/api
./api  # On Windows: api.exe
```
Should start without errors. You may see "Server listening on :8080" or similar.

Press Ctrl+C to stop the server.

### Step 3.3: Test the API Manually

**Prompt to type:**
```
The server is running. Help me test the POST /api/tasks endpoint using curl. First, create a sample curl command, then I'll run it to create a task.
```

**What to expect:**
- Agent provides a curl command like:
  ```bash
  curl -X POST http://localhost:8080/api/tasks \
    -H "Content-Type: application/json" \
    -d '{"title": "Test task", "priority": "high"}'
  ```

**Your action:**
1. Start the server in another terminal: `./api`
2. Run the curl command the agent provided
3. Report results to the agent

**Prompt to type after testing:**
```
The curl command worked / didn't work. Here's the output: [paste output]
```

**What you learned:**
- ✅ Using bash tool for Go commands (go get, go build)
- ✅ How agent debugs and fixes errors
- ✅ Agent can generate test commands for you

---

## Phase 4: LSP-Powered Refactoring (~10 minutes)

**What you'll learn:** How GoCode uses gopls for intelligent code navigation, finding definitions and references

**Note:** This phase requires `lsp.enabled: true` in config.yaml and gopls installed.

### Step 4.1: Verify LSP is Working

**Prompt to type:**
```
I want to refactor the task validation logic. First, use LSP to find all places where the Validate function is called.
```

**What to expect:**
- Agent uses `lsp_find_references` tool to find all calls to Validate()
- Agent lists the locations (file:line)
- Shows you the code context

**What you're seeing:** This is much more accurate than grep because it understands Go semantics (types, scopes, etc.)

### Step 4.2: Find Definition

**Prompt to type:**
```
Now find the definition of the Task struct and show me its fields.
```

**What to expect:**
- Agent uses `lsp_find_definition` tool
- Jumps directly to the struct definition in models/task.go
- Reads and explains the struct

**Checkpoint:** Agent should correctly identify internal/models/task.go as the location.

### Step 4.3: List Symbols in a File

**Prompt to type:**
```
List all the functions and types defined in internal/handlers/tasks.go
```

**What to expect:**
- Agent uses `lsp_list_symbols` tool
- Shows you all functions, types, constants in the file
- Organized by symbol type

**This is useful for:** Getting an overview of a file's structure without reading the whole thing.

### Step 4.4: LSP-Guided Refactoring

**Prompt to type:**
```
I want to extract the error response logic into a helper function since it's repeated in multiple handlers. Use LSP to find all the error response code, then refactor it into a reusable function.
```

**What to expect:**
- Agent uses LSP tools to find error handling patterns
- Agent creates a new helper function (e.g., `respondWithError`)
- Agent uses `edit` tool to replace duplicated code with calls to the helper
- Agent uses LSP to verify the refactoring didn't break anything

**Checkpoint:**
```bash
go build ./cmd/api
```
Should build successfully after refactoring.

**What you learned:**
- ✅ LSP provides semantic code understanding
- ✅ More accurate than text search (grep)
- ✅ Essential for safe refactoring
- ✅ How agent uses `lsp_find_definition`, `lsp_find_references`, `lsp_list_symbols`

---

## Phase 5: Testing & Debugging (~10 minutes)

**What you'll learn:** Test-driven development with GoCode, debugging errors, bash tool for running tests

### Step 5.1: Create Tests for Handlers

**Prompt to type:**
```
Create comprehensive unit tests for all the handler functions in tests/handlers_test.go. Include tests for success cases, validation errors, and not found errors.
```

**What to expect:**
- Agent creates tests/handlers_test.go
- Includes table-driven tests for each endpoint
- Tests success and error cases
- Uses httptest package for HTTP testing

**Checkpoint:**
```bash
cat tests/handlers_test.go
```
Should see test functions like TestGetAllTasks, TestCreateTask, etc.

### Step 5.2: Run the Tests

**Prompt to type:**
```
Run the tests and show me the results. If any tests fail, help me fix them.
```

**What to expect:**
- Agent runs `go test ./tests -v`
- Shows test output
- If failures occur, agent reads the error messages and fixes the code

**Common issues:**
- Database not initialized in tests
- Missing test data
- Handler expecting different response format

**Agent will:**
- Use `read` tool to examine failing tests
- Use `edit` tool to fix the handlers or tests
- Re-run tests to verify fixes

**Checkpoint:** All tests should pass. You should see output like:
```
PASS
ok      task-api/tests  0.234s
```

### Step 5.3: Add Database Tests

**Prompt to type:**
```
Create integration tests for the database layer in tests/database_test.go. Test the CRUD operations with an in-memory SQLite database.
```

**What to expect:**
- Agent creates database_test.go
- Sets up test database with migrations
- Tests all query functions

### Step 5.4: Check Test Coverage

**Prompt to type:**
```
Run the tests with coverage reporting and show me the results.
```

**What to expect:**
- Agent runs `go test ./... -cover`
- Shows coverage percentage
- May suggest areas that need more tests

**Checkpoint:**
```bash
go test ./... -cover
```
Should show >80% coverage (as required by spec).

**What you learned:**
- ✅ Agent can write comprehensive tests
- ✅ Agent debugs failing tests automatically
- ✅ Using bash tool for `go test`
- ✅ Test-driven development workflow

---

## Phase 6: Session Management (~5 minutes)

**What you'll learn:** Checkpointing (resume work later), long-term memory (agent remembers context)

**Note:** This phase requires `checkpoint.enabled: true` and `memory.enabled: true` in config.yaml.

### Step 6.1: Checkpoint Your Session

**Prompt to type:**
```
I need to take a break. Please summarize what we've accomplished so far and save the session state.
```

**What to expect:**
- Agent summarizes the completed work
- Session is automatically saved to checkpoints.db
- Agent may store key facts in long-term memory

**Your action:** Exit GoCode
```
exit
```

### Step 6.2: Resume the Session

**Your action:** Restart GoCode in the same directory
```bash
gocode
```

**What to expect:**
- GoCode detects existing checkpoint
- May ask if you want to resume the previous session
- If you say yes, conversation history is restored

**Prompt to type:**
```
What were we working on? What's left to do?
```

**What to expect:**
- Agent recalls the project context
- May reference the TODO.md file
- Can continue exactly where you left off

**This demonstrates:** The agent maintained context across sessions!

### Step 6.3: Test Long-term Memory

**Prompt to type:**
```
Remind me why we chose to use SQLite instead of PostgreSQL for this project.
```

**What to expect:**
- Agent recalls the decision (simple, embedded, no external dependencies)
- May reference the spec or earlier conversation
- Shows that memory persists across sessions

### Step 6.4: Continue Development

**Prompt to type:**
```
Let's add a new feature: filtering tasks by status. Add a query parameter ?status=todo to the GET /api/tasks endpoint.
```

**What to expect:**
- Agent picks up where you left off
- Understands the existing codebase
- Implements the feature using edit tool
- Updates tests

**What you learned:**
- ✅ Checkpointing lets you pause and resume work
- ✅ Long-term memory helps agent remember context
- ✅ Essential for multi-day projects
- ✅ Agent maintains understanding across sessions

---

## Phase 7: Final Polish & Documentation (~10 minutes)

**What you'll learn:** Code review, documentation, final deployment

### Step 7.1: Request a Code Review

**Prompt to type:**
```
Review all the code we've written. Check for:
- Error handling completeness
- Code organization and clarity
- Potential bugs or edge cases
- Best practices violations
Suggest improvements.
```

**What to expect:**
- Agent uses `glob` to find all .go files
- Agent uses `read` to review each file
- Agent provides a structured review with suggestions
- May suggest specific improvements

### Step 7.2: Implement Suggested Improvements

**Prompt to type:**
```
Please implement the top 3 most important improvements you suggested.
```

**What to expect:**
- Agent uses `edit` tool to make changes
- Explains each change
- Runs tests to verify nothing broke

### Step 7.3: Generate Documentation

**Prompt to type:**
```
Create a comprehensive README.md for this project. Include:
- Project description
- How to build and run
- API endpoint documentation
- How to run tests
- Example curl commands
```

**What to expect:**
- Agent creates README.md with all requested sections
- Includes code examples
- Professional formatting

**Checkpoint:**
```bash
cat README.md
```
Should be a complete, professional README.

### Step 7.4: Final Test

**Prompt to type:**
```
Let's do a final end-to-end test. Start the server, then guide me through testing all 5 endpoints with curl commands.
```

**What to expect:**
- Agent provides curl commands for each endpoint
- You run them manually and report results
- Agent verifies everything works

**Your action:**
1. Start server: `go run cmd/api/main.go`
2. Run each curl command in another terminal
3. Verify responses

**Checkpoint:** All endpoints should work correctly.

---

## Congratulations!

You've completed the GoCode Task API walkthrough!

### What You Built

✅ Complete REST API with 5 endpoints
✅ SQLite database with migrations
✅ Comprehensive validation and error handling
✅ Unit and integration tests (>80% coverage)
✅ Production-ready code structure
✅ Full documentation

### What You Learned

**GoCode Tools:**
- `read` - Reading files and code
- `write` - Creating new files
- `edit` - Modifying existing code
- `glob` - Finding files by pattern
- `grep` - Searching code content
- `bash` - Running commands (go build, go test, curl)
- `todo_write` - Task breakdown and tracking
- `lsp_find_definition` - Jump to code definitions
- `lsp_find_references` - Find where code is used
- `lsp_list_symbols` - List functions and types

**GoCode Features:**
- How to prompt the agent effectively
- Breaking down complex tasks
- Debugging and fixing errors
- Test-driven development
- Code refactoring with LSP
- Session checkpointing
- Long-term memory

**Software Engineering:**
- REST API design
- Go project structure
- Database migrations
- Error handling patterns
- Unit testing strategies

---

## Next Steps

### Extend This Project

Try asking the agent to add:

1. **Pagination**
   ```
   Add pagination to GET /api/tasks with ?page=1&limit=10 parameters
   ```

2. **Filtering**
   ```
   Add filtering by priority and status to GET /api/tasks
   ```

3. **Authentication**
   ```
   Add JWT-based authentication to protect the endpoints
   ```

4. **Search**
   ```
   Add full-text search on task title and description
   ```

5. **Sorting**
   ```
   Add sorting by created_at, priority, or status
   ```

### Apply to Your Own Projects

Now that you understand GoCode's capabilities, try:
- Starting a new project from scratch
- Refactoring existing code
- Adding tests to legacy code
- Debugging complex issues
- Learning a new framework

### Share Your Experience

- What worked well?
- What was confusing?
- What features would you like to see?
- Create your own examples and share them!

---

## Troubleshooting

### LSP Not Working
- Verify: `gopls version`
- Check config: `lsp.enabled: true`
- Ensure go.mod exists in project root

### Tests Failing
- Check database initialization
- Verify imports are correct
- Run `go mod tidy`

### Checkpointing Issues
- Verify: `checkpoint.enabled: true`
- Check checkpoints.db was created
- Ensure you're in the same directory

### Agent Seems Confused
- Be more specific in prompts
- Break complex requests into smaller steps
- Show the agent relevant files with "read"

### Build Errors
- Run `go mod tidy`
- Check Go version: `go version`
- Verify all imports exist

---

## Additional Resources

- [GoCode Documentation](../../README.md)
- [Go Documentation](https://go.dev/doc/)
- [REST API Design](https://restfulapi.net/)
- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [Testing in Go](https://go.dev/doc/tutorial/add-a-test)

---

**Questions or Issues?**

If you encounter problems with this walkthrough:
1. Check the Troubleshooting section above
2. Review the prerequisites
3. Try the prompt variations below
4. Open an issue on the GoCode repository

---

## Prompt Variations

If the exact prompts don't work as expected, try these variations:

**Instead of:** "Create the database layer"
**Try:** "Create internal/database/db.go with functions to open a SQLite connection and run migrations"

**Instead of:** "Implement the API endpoints"
**Try:** "Implement the GetAllTasks handler function in internal/handlers/tasks.go that queries the database and returns JSON"

**Instead of:** "Fix the errors"
**Try:** "Read the build error output and fix the missing import in handlers/tasks.go"

**Key insight:** More specific prompts often get better results!

---

**Enjoy building with GoCode!** 🚀
