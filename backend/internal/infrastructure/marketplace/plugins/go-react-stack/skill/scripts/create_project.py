#!/usr/bin/env python3
"""
Go+React Full-Stack Project Creation Script

This script creates a Go+React full-stack project scaffold following OpenSpec conventions.
"""

import os
import sys
import json
import shutil
from pathlib import Path
from typing import Dict, Optional

def ask_question(prompt: str, default: Optional[str] = None, validator=None) -> str:
    """Interactively ask user for input"""
    if default:
        full_prompt = f"{prompt} [{default}]: "
    else:
        full_prompt = f"{prompt}: "
    
    while True:
        answer = input(full_prompt).strip()
        if not answer and default:
            return default
        if not answer:
            print("This field cannot be empty, please re-enter")
            continue
        if validator:
            try:
                validator(answer)
            except ValueError as e:
                print(f"Validation failed: {e}")
                continue
        return answer

def validate_kebab_case(value: str) -> None:
    """Validate kebab-case format"""
    if not value.replace('-', '').replace('_', '').isalnum():
        raise ValueError("Project name can only contain letters, numbers, hyphens, and underscores")
    if value[0] == '-' or value[-1] == '-':
        raise ValueError("Project name cannot start or end with a hyphen")

def validate_module_path(value: str) -> None:
    """Validate Go module path"""
    if not value or '/' not in value:
        raise ValueError("Go module path format should be: github.com/user/project")

def create_directory_structure(base_path: Path, project_name: str, include_openspec: bool) -> None:
    """Create project directory structure"""
    dirs = [
        "backend/cmd/server",
        "backend/internal/domain/example",
        "backend/internal/application/example",
        "backend/internal/infrastructure/config",
        "backend/internal/infrastructure/storage",
        "backend/internal/interfaces/http/handler",
        "backend/internal/interfaces/http/response",
        "backend/internal/wire",
        "backend/docs",
        "frontend/src/components/common",
        "frontend/src/hooks",
        "frontend/src/services",
        "frontend/src/utils",
        "frontend/src/types",
        "frontend/src/styles",
        "frontend/public",
    ]
    
    if include_openspec:
        dirs.extend([
            "openspec/specs/go-style",
            "openspec/specs/typescript-style",
            "openspec/specs/api-conventions",
            "openspec/specs/testing",
            "openspec/changes/archive",
        ])
    
    for dir_path in dirs:
        (base_path / dir_path).mkdir(parents=True, exist_ok=True)
    
    print("✓ Created directory structure")

def generate_go_mod(base_path: Path, module_path: str) -> None:
    """Generate go.mod file"""
    content = f"""module {module_path}

go 1.24.0

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.6.0
	github.com/stretchr/testify v1.11.1
	github.com/swaggo/files v1.0.1
	github.com/swaggo/gin-swagger v1.6.0
	github.com/swaggo/swag v1.8.12
)
"""
    (base_path / "backend" / "go.mod").write_text(content, encoding='utf-8')
    print("✓ Generated go.mod")

def generate_package_json(base_path: Path, project_name: str) -> None:
    """Generate package.json file"""
    content = {
        "name": f"{project_name}-frontend",
        "version": "0.1.0",
        "private": True,
        "type": "module",
        "scripts": {
            "dev": "vite",
            "build": "tsc && vite build",
            "preview": "vite preview",
            "lint": "eslint src --ext ts,tsx"
        },
        "dependencies": {
            "react": "^18.2.0",
            "react-dom": "^18.2.0",
            "react-router-dom": "^6.20.0",
            "axios": "^1.6.0"
        },
        "devDependencies": {
            "@types/react": "^18.2.0",
            "@types/react-dom": "^18.2.0",
            "@typescript-eslint/eslint-plugin": "^6.0.0",
            "@typescript-eslint/parser": "^6.0.0",
            "@vitejs/plugin-react": "^4.2.0",
            "eslint": "^8.0.0",
            "eslint-plugin-react": "^7.33.0",
            "eslint-plugin-react-hooks": "^4.6.0",
            "typescript": "^5.0.0",
            "vite": "^5.0.0"
        }
    }
    (base_path / "frontend" / "package.json").write_text(
        json.dumps(content, indent=2, ensure_ascii=False) + "\n",
        encoding='utf-8'
    )
    print("✓ Generated package.json")

def copy_openspec_templates(base_path: Path, project_name: str, description: str, module_path: str) -> None:
    """Copy and customize OpenSpec template files"""
    import shutil
    
    # Get the script directory to find templates
    # Script is in skill/scripts/, templates are in skill/assets/openspec-templates/
    script_dir = Path(__file__).parent
    template_dir = script_dir.parent / "assets" / "openspec-templates"
    target_dir = base_path / "openspec"
    
    if not template_dir.exists():
        print(f"Warning: OpenSpec templates not found at {template_dir}, skipping...")
        return
    
    # Copy AGENTS.md and project.md
    for file in ["AGENTS.md", "project.md"]:
        src = template_dir / file
        if src.exists():
            dst = target_dir / file
            content = src.read_text(encoding='utf-8')
            # Replace placeholders in project.md
            if file == "project.md":
                content = content.replace("[PROJECT_DESCRIPTION]", description)
            dst.write_text(content, encoding='utf-8')
            print(f"✓ Generated openspec/{file}")
    
    # Copy specs directory
    specs_src = template_dir / "specs"
    specs_dst = target_dir / "specs"
    
    if specs_src.exists():
        # Copy all spec files
        for spec_dir in specs_src.iterdir():
            if spec_dir.is_dir():
                spec_dst = specs_dst / spec_dir.name
                spec_dst.mkdir(parents=True, exist_ok=True)
                
                for spec_file in spec_dir.glob("*.md"):
                    content = spec_file.read_text(encoding='utf-8')
                    # Customize api-conventions/spec.md
                    if spec_dir.name == "api-conventions":
                        content = content.replace("Project API", f"{project_name} API")
                        content = content.replace("Project API service", description)
                        content = content.replace("github.com/user/project", module_path)
                    (spec_dst / spec_file.name).write_text(content, encoding='utf-8')
                    print(f"✓ Generated openspec/specs/{spec_dir.name}/{spec_file.name}")

def generate_readme(base_path: Path, project_name: str, description: str, module_path: str) -> None:
    """Generate README.md"""
    content = f"""# {project_name}

{description}

## Tech Stack

### Backend
- Go 1.24+
- Gin (HTTP framework)
- Wire (Dependency injection)
- Swagger (API documentation)

### Frontend
- React 18+
- TypeScript
- Vite
- React Router
- Axios

## Quick Start

### Backend

```bash
cd backend
go mod download
make run
```

Backend service will run on `http://localhost:8080`

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Frontend dev server will run on `http://localhost:3000`

## API Documentation

After starting the backend service, visit `http://localhost:8080/swagger/index.html` to view API documentation.

## Project Structure

```
{project_name}/
├── backend/          # Go backend
├── frontend/         # React frontend
└── openspec/         # OpenSpec specifications (if enabled)
```

## Development Guide

Refer to documentation in each directory and OpenSpec specifications for detailed development guide.
"""
    (base_path / "README.md").write_text(content, encoding='utf-8')
    print("✓ Generated README.md")

def main():
    """Main function"""
    print("=" * 60)
    print("Go+React Full-Stack Project Creation Tool")
    print("=" * 60)
    print()
    
    # Interactive questions
    project_name = ask_question("Project name (kebab-case)", validator=validate_kebab_case)
    module_path = ask_question("Go module path", f"github.com/your-username/{project_name}", validator=validate_module_path)
    description = ask_question("Project description", "A Go+React full-stack application")
    include_openspec = ask_question("Include OpenSpec? (y/n)", "y").lower() == 'y'
    build_tool = ask_question("Frontend build tool (vite/webpack)", "vite").lower()
    project_dir = ask_question("Project location (absolute path)", str(Path.cwd() / project_name))
    
    # Create project directory
    base_path = Path(project_dir)
    if base_path.exists():
        response = input(f"Directory {base_path} already exists, continue? (y/n): ").lower()
        if response != 'y':
            print("Cancelled")
            return
    else:
        base_path.mkdir(parents=True, exist_ok=True)
    
    print(f"\nCreating project: {project_name}")
    print(f"Location: {base_path}")
    print()
    
    # Create directory structure
    create_directory_structure(base_path, project_name, include_openspec)
    
    # Generate configuration files
    generate_go_mod(base_path, module_path)
    generate_package_json(base_path, project_name)
    generate_readme(base_path, project_name, description, module_path)
    
    # Generate OpenSpec files if enabled
    if include_openspec:
        copy_openspec_templates(base_path, project_name, description, module_path)
    
    # TODO: Generate more configuration files (.golangci.yml, Makefile, tsconfig.json, vite.config.ts, .eslintrc.json)
    # TODO: Generate example code files
    
    print()
    print("=" * 60)
    print("Project creation completed!")
    print("=" * 60)
    print()
    print("Next steps:")
    print(f"1. cd {base_path}/backend && go mod download")
    print(f"2. cd {base_path}/frontend && npm install")
    print(f"3. cd {base_path}/backend && make run")
    print(f"4. cd {base_path}/frontend && npm run dev")

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nCancelled")
        sys.exit(1)
    except Exception as e:
        print(f"\nError: {e}")
        sys.exit(1)
