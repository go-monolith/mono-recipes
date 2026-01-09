# Constraints

These constraints define the hard boundaries for achieving the goal "Use Latest Framework Version" defined in [goal.md](./goal.md).

## Tech Stack & Libraries

No specific version constraint. Use the latest stable version of go-monolith/mono framework available at the time of upgrade.

## Git & Cadence

- **Git Branching Strategy**: Work directly on main branch
- **Git Commit Frequency**: Commit after each module update is completed and tested
- **OKR Review Cadence**: N/A
- **Human Executive Check-in Frequency**: As needed

## Security & Compliance

- All tests (unit-test and integration-test) must pass before each commit
- No commits should break existing functionality

## Performance Targets

No specific performance targets for this upgrade. Accept the performance characteristics of the new framework version.

## "Don't Touch" Areas

- **Modules with failing tests**: Do not update any module that currently has failing tests. Fix the failing tests first before attempting the framework upgrade for that module.

## Definition of "Safe Change"

- ✅ All existing tests pass (both unit-test and integration-test) before committing
- ✅ Code compiles without errors with the new framework version
- ✅ No new deprecation warnings introduced (or existing deprecated API usage is fixed)
- ✅ Documentation updated to reflect new framework version
- ✅ Each module is updated incrementally and tested independently

## Additional Constraints

- Each commit should represent a complete, working state (all tests pass)
- If a module upgrade reveals bugs or issues, fix them before proceeding to the next module

---

*These constraints are part of the Goal Driven Development (GDD) process.*
