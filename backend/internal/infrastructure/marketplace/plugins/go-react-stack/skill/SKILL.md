---
name: go-react-stack
description: Create a new Go+React full-stack project with DDD architecture, OpenSpec specifications, and React best practices. Includes complete project scaffolding with backend (Go with Gin), frontend (React with TypeScript), OpenSpec structure, and example code following project conventions. Use this skill when users request creating a new full-stack project, scaffolding a Go+React application, or setting up a project with OpenSpec specifications.
---

# Go+React Full-Stack Project Scaffold

Create a Go+React full-stack project scaffold following OpenSpec conventions, with DDD layered architecture and React best practices.

## Workflow

### 1. Interactive Project Information Gathering

Ask the user for the following information before creating the project:

1. **Project name** (kebab-case):
   - Examples: `my-app`, `todo-app`
   - Validation: Only lowercase letters, numbers, and hyphens

2. **Go module path**:
   - Example: `github.com/user/my-app`
   - Default: Suggest `github.com/your-username/{project-name}` if project name is provided

3. **Project description**:
   - Brief description of project purpose

4. **Include OpenSpec** (default: yes):
   - If included, creates complete OpenSpec directory structure and specification files

5. **Frontend build tool** (default: Vite):
   - Options: Vite or Webpack
   - Recommend Vite for faster development experience

6. **Project location**:
   - Default: Current working directory
   - Can specify absolute path

### 2. Create Project Directory Structure

Create the following directory structure based on user input:

```
<project-name>/
├── backend/                    # Go backend
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── domain/            # Domain layer
│   │   │   └── example/       # Example domain
│   │   ├── application/       # Application layer
│   │   │   └── example/       # Example application service
│   │   ├── infrastructure/    # Infrastructure layer
│   │   │   ├── config/
│   │   │   └── storage/
│   │   └── interfaces/        # Interface layer
│   │       └── http/
│   │           ├── handler/
│   │           ├── response/
│   │           └── server.go
│   ├── internal/wire/         # Wire dependency injection
│   ├── docs/                  # Swagger docs (auto-generated)
│   ├── go.mod
│   ├── go.sum
│   ├── .golangci.yml
│   └── Makefile
├── frontend/                   # React frontend
│   ├── src/
│   │   ├── components/        # React components
│   │   │   └── common/       # Common components
│   │   ├── hooks/             # Custom hooks
│   │   ├── services/          # API services
│   │   ├── utils/             # Utility functions
│   │   ├── types/             # TypeScript types
│   │   ├── styles/            # Style files
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── public/
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts         # or webpack.config.js
│   └── .eslintrc.json
├── openspec/                   # OpenSpec specifications (if enabled)
│   ├── AGENTS.md
│   ├── project.md
│   ├── specs/
│   │   ├── go-style/
│   │   ├── typescript-style/
│   │   ├── api-conventions/
│   │   └── testing/
│   └── changes/
│       └── archive/
├── .gitignore
└── README.md
```

### 3. Generate Configuration Files

#### Go Backend Configuration

**go.mod**: Use `go mod init <module-path>` to create, include basic dependencies:
- `github.com/gin-gonic/gin` - HTTP framework
- `github.com/google/wire` - Dependency injection
- `github.com/stretchr/testify` - Testing framework
- `github.com/swaggo/swag` - Swagger documentation

**Makefile**: Include common commands (build, test, run, wire, swagger, etc.)

**.golangci.yml**: Code linting configuration (see `references/go-configs.md`)

#### React Frontend Configuration

**package.json**: Include React, TypeScript, Vite, routing, state management dependencies

**tsconfig.json**: TypeScript configuration (strict mode)

**vite.config.ts**: Vite configuration with proxy settings (proxy to backend API)

**.eslintrc.json**: ESLint configuration (see `references/typescript-configs.md`)

#### OpenSpec Configuration (if enabled)

Copy template files from `assets/openspec-templates/` to `openspec/` directory and customize them based on project information:

1. **project.md**: Replace `[PROJECT_DESCRIPTION]` placeholder with actual project description
2. **specs/go-style/spec.md**: Copy as-is (generic Go style guide)
3. **specs/typescript-style/spec.md**: Copy as-is (generic TypeScript style guide)
4. **specs/api-conventions/spec.md**: Update API title, host, and base path based on project
5. **specs/testing/spec.md**: Copy as-is (generic testing guide)

When copying these files, replace placeholders like:
- `[project-name]` → actual project name
- `[module-path]` → actual Go module path
- `[description]` → actual project description
- `localhost:8080` → actual backend port (if different)

### 4. Generate Example Code

#### Backend Examples

1. **Domain layer** (`internal/domain/example/`):
   - `entity.go` - Entity definition
   - `repository.go` - Repository interface

2. **Application layer** (`internal/application/example/`):
   - `service.go` - Application service
   - `dto.go` - Data transfer objects

3. **Infrastructure layer** (`internal/infrastructure/storage/`):
   - `example_repository.go` - Repository implementation

4. **Interface layer** (`internal/interfaces/http/handler/`):
   - `example_handler.go` - HTTP handler (with Swagger annotations)

5. **Wire configuration** (`internal/wire/`):
   - `wire.go` - Wire provider sets

6. **Test files**:
   - `*_test.go` files for each layer

#### Frontend Examples

1. **Components** (`src/components/`):
   - `Home.tsx` - Home page component
   - `common/Button.tsx` - Common button component
   - `common/Loading.tsx` - Loading component

2. **Hooks** (`src/hooks/`):
   - `useApi.ts` - API call hook

3. **Services** (`src/services/`):
   - `api.ts` - API client wrapper

4. **Routing** (`src/App.tsx`):
   - React Router configuration example

5. **Types** (`src/types/`):
   - `index.ts` - Type definitions

### 5. Generate Documentation

**README.md**: Include:
- Project introduction
- Tech stack description
- Quick start guide
- Project structure description
- Development guide

### 6. Initialize Git (Optional)

Ask user if they want to initialize Git repository. If yes:
- Run `git init`
- Create `.gitignore` file
- Create initial commit

## Using the Script

Use the provided Python script to quickly create a project:

```bash
python3 scripts/create_project.py
```

The script will interactively ask for project information and automatically create the project structure.

## References

- **Go configuration templates**: See `references/go-configs.md`
- **TypeScript/React configuration templates**: See `references/typescript-configs.md`
- **Example code**: See `references/example-code.md`
- **OpenSpec templates**: See `assets/openspec-templates/`
- **Project creation script**: See `scripts/create_project.py`

## Important Notes

1. **Module path**: Ensure Go module path is correct, as import paths in code will be based on this
2. **Port configuration**: Backend default port 8080, frontend dev server default 3000
3. **API proxy**: Frontend Vite configuration includes proxy settings to backend API
4. **Code conventions**: All generated code follows OpenSpec convention requirements
5. **Test coverage**: Example code includes test files demonstrating TDD practices

## Next Steps

After project creation, prompt the user:
1. Navigate to backend: `cd backend && go mod download`
2. Navigate to frontend: `cd frontend && npm install`
3. Start backend: `cd backend && make run`
4. Start frontend: `cd frontend && npm run dev`
5. View API docs: Visit `http://localhost:8080/swagger/index.html`
