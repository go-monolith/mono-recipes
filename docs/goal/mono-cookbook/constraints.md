# Constraints

These constraints define the hard boundaries for achieving the goal "Mono Cookbook" defined in [goal.md](./goal.md).

## Tech Stack & Libraries

| Category | Allowed | Not Allowed |
|----------|---------|-------------|
| Language | Go | Python, TypeScript, Java, etc. |
| Framework | `mono` framework, Standard library, popular Go libraries | - |
| Database | PostgreSQL, MongoDB, Redis, etc. | - |
| Libraries | As appropriate for each recipe | - |

## Git & Cadence

- **Git Branching Strategy**: Create a feature branch `feat/*` for each recipe separately from `main` branch (latest), commit and submit a PR back to `main` branch for review.
- **Git Commit Frequency**: Commit after completing each logical unit of work. Must use "#2 <recipe-name>: <commit-message>" for commit messages format.
- **OKR Review Cadence**: Must review Goal's Success Criteria after completing each recipe
- **Human Executive Check-in Frequency**: Require Human Executive check-in after completing every 5 recipes
- **Git Constraints**: Do not stage changes in `tasks.md`, `goal.md` or `constraints.md` nor commit these changes in your feature branches. These files are managed separately.

## Security & Compliance

- All examples should follow security best practices
- No hardcoded secrets or credentials in code examples
- Use environment variables or config files for sensitive data

## Performance Targets

N/A - This is a documentation/reference project.

## "Don't Touch" Areas

- Existing sample projects in the repository - do not modify them

## Definition of "Safe Change"

- ✅ Example works as intended and meets the recipe requirements. Application must meet the recipe's description and functionality.
- ✅ Example should follow mono framework conventions and patterns (e.g., module structure, service registration)
- ✅ Code follows Go best practices and idioms. No anti-patterns. Prefer to use hexagonal architecture where applicable.
- ✅ Code is properly documented with comments and README files

## Additional Constraints

- Each recipe directory must have its own `README.md` explaining the pattern
- All code examples must be runnable without modification
- Each recipe must include a `demo.sh` bash script or `demo.py` python script to demonstrate the application
- Recipes should be self-contained and not depend on other recipes
- Submit PRs for each recipe separately for review and move on to the next recipe immediately without waiting for approval

---

*These constraints are part of the Goal Driven Development (GDD) process.*
