# Project Instructions — Local-First Academic TUI Manager

## 1. Your role

Act as a senior Go engineer, software architect, TUI developer and technical writer.

You are responsible for designing and implementing a complete, usable first version of a local-first academic management application for Windows, with a terminal user interface (TUI) and a standalone CLI.

The application will help a student manage:

* 9 or more course units.
* Topics and reading progress within each unit.
* Assignments, coursework, tests and examinations.
* Academic documents, research papers, notes and summaries.
* Deadlines, weekly schedules and study progress.
* Academic analytics, coverage and time-sensitive priorities.
* Local files, imports, exports, backups and application configuration.

The application should be built in Go, using Charm libraries where appropriate.

You have permission to research the installed dependencies, design the architecture, create project documentation, implement the application, write tests and improve the implementation as you progress.

**Do not attempt to implement the entire application in one step.** Plan the system, divide it into independently testable stages and implement those stages in sequence.

The first goal is a reliable, functional application, not an unnecessarily sophisticated architecture.

---

## 2. Core development principles

Follow these principles throughout development.

1. **Simplicity first:** Prefer understandable solutions over complex abstractions.
2. **Local-first:** The application must work without an internet connection.
3. **Data safety:** Academic records and user files must never be silently lost or overwritten.
4. **Modularity:** Separate the TUI, application logic, storage, filesystem operations and CLI.
5. **Incremental development:** Implement, test and document small, complete features.
6. **Maintainability:** Use clear functions, structs, interfaces only where justified, and explicit error handling.
7. **Resource awareness:** Keep memory usage, executable size and dependencies reasonable for low-end Windows computers.
8. **Framework discipline:** Use Charm libraries for established TUI functionality, but do not force every interface element into a component that is unsuitable.
9. **Configuration-driven builds:** Versioning, packaging and release procedures must be documented and reproducible.
10. **No premature overengineering:** Design for future extension without implementing speculative features unnecessarily.

Do not add unnecessary frameworks, databases, services or dependencies.

Use Go's standard library wherever it is sufficient.

---

## 3. Technology stack

Use the following as the initial stack.

| Area             | Technology                                               |
| ---------------- | -------------------------------------------------------- |
| Language         | Go                                                       |
| TUI framework    | Bubble Tea                                               |
| TUI styling      | Lipgloss                                                 |
| TUI components   | Bubbles                                                  |
| CLI              | Cobra, if its benefits justify the dependency            |
| Primary database | SQLite                                                   |
| Database driver  | Select a maintained, Windows-compatible Go SQLite driver |
| Configuration    | JSON                                                     |
| Structured data  | JSON                                                     |
| Logging          | Go standard library `log/slog`                           |
| Testing          | Go `testing`                                             |
| Build            | Go toolchain                                             |
| Version control  | Git                                                      |
| Installer        | Separate Go executable                                   |
| Release hosting  | GitHub Releases                                          |
| Documentation    | Markdown                                                 |

Use maintained, compatible dependency versions. Inspect the available Go version before selecting dependencies.

Do not introduce a dependency solely because it is popular.

If Cobra is used, the application should still support a simple, discoverable command structure.

---

## 4. Application identity

The final application name has not yet been decided.

Use a configurable placeholder identity during development.

Create a central application metadata configuration that includes:

* Application name.
* Executable name.
* Application identifier.
* Current version.
* Build number, if needed.
* Database schema version.
* Default storage directory name.
* GitHub repository URL placeholder.
* Release channel.

Do not hardcode the application name in multiple source files.

Keep application metadata and build configuration separate from academic data.

The application must support changing its name and version through a small number of clearly documented configuration files.

---

# PART I — PROJECT PLANNING

## 5. Documentation-first workflow

Before implementing the application, inspect the empty repository and create the complete project documentation.

Create between 12 and 20 Markdown documents in a `docs/` directory.

Use the following initial documentation structure:

```text
docs/
├── 00_PROJECT_OVERVIEW.md
├── 01_REQUIREMENTS.md
├── 02_SYSTEM_ARCHITECTURE.md
├── 03_PROJECT_STRUCTURE.md
├── 04_DATA_MODEL.md
├── 05_DATABASE_AND_MIGRATIONS.md
├── 06_FILESYSTEM_AND_STORAGE.md
├── 07_TUI_DESIGN.md
├── 08_CLI_SPECIFICATION.md
├── 09_TASKS_AND_DEADLINES.md
├── 10_ANALYTICS_AND_COVERAGE.md
├── 11_IMPORT_EXPORT_AND_BACKUP.md
├── 12_CONFIGURATION_AND_INSTALLER.md
├── 13_TESTING_STRATEGY.md
├── 14_BUILD_AND_RELEASE.md
├── 15_IMPLEMENTATION_ROADMAP.md
├── 16_SECURITY_AND_DATA_SAFETY.md
└── 17_FUTURE_EXTENSIONS.md
```

Each document must be useful and specific to this project, not generic software documentation.

### Documentation requirements

**00 — Project overview**

* Explain the purpose and scope.
* Describe the target user and environment.
* Establish the first usable version's boundaries.
* Describe the relationship between the TUI, CLI, database and local files.

**01 — Requirements**

* Functional and non-functional requirements.
* Explicitly distinguish essential features from future extensions.
* Define acceptance criteria for every major feature.

**02 — System architecture**

* Explain the application's components and responsibilities.
* Describe dependencies between modules.
* Document how the CLI and TUI access the same application services.
* Include a Mermaid architecture diagram.

**03 — Project structure**

* Define the complete source-code folder structure.
* Explain what each package owns.
* Establish package dependency rules.
* Avoid circular dependencies and oversized packages.

**04 — Data model**

* Define course units, topics, tasks, deadlines, files, progress and other relevant entities.
* Define relationships and unique identifiers.
* Explain which information belongs in SQLite and which belongs in the filesystem.

**05 — Database and migrations**

* Define the SQLite schema.
* Explain database initialization, indexing, transactions and migrations.
* Document schema versioning and recovery considerations.

**06 — Filesystem and storage**

* Define the academic folder hierarchy.
* Explain safe file creation, deletion, renaming and moving.
* Define storage path resolution and file metadata.
* Document how the database remains consistent with filesystem changes.

**07 — TUI design**

* Define the screens, navigation, layouts, styling and keyboard shortcuts.
* Include textual wireframes.
* Explain the persistent command input and dynamic display areas.
* Define responsive behaviour for different terminal sizes.

**08 — CLI specification**

* Define the root command, subcommands, flags and examples.
* Include commands for academic records, file management, imports, exports and maintenance.
* Define non-interactive behaviour and error reporting.

**09 — Tasks and deadlines**

* Define assignments, coursework, tests, examinations and other deadlines.
* Explain how dates, completion status, changes and reminders are managed.
* Define how deadlines appear on the timeline.

**10 — Analytics and coverage**

* Define the coverage calculation for course units and topics.
* Explain completion percentages, pending work and overdue items.
* Define how academic progress and upcoming deadlines are compared.
* Ensure calculations are transparent and reproducible.

**11 — Import, export and backup**

* Define file imports, exports and backup formats.
* Specify how academic records and associated files are exported.
* Define a portable plain-text status report for phone access.
* Explain restoration and backup verification.

**12 — Configuration and installer**

* Define configuration files, storage location selection and installation behaviour.
* Specify the two-executable distribution.
* Document PATH integration and application updates.
* Establish safe handling of existing installations and user data.

**13 — Testing strategy**

* Define unit, integration, CLI, TUI-model and installer tests.
* Establish test fixtures and temporary directories.
* Include data integrity and migration testing.
* Define release acceptance tests.

**14 — Build and release**

* Define development builds, production builds and version injection.
* Explain Windows packaging, release artifacts and GitHub Releases.
* Define version tags, release notes and build verification.
* Make the project suitable for future integration with an external build manager.

**15 — Implementation roadmap**

* Divide the project into sequential milestones.
* Identify dependencies between milestones.
* Define acceptance criteria and completion checks.
* Record which features are explicitly deferred.

**16 — Security and data safety**

* Cover file operation safety, path validation, database transactions and backups.
* Address accidental deletion, partial operations, malformed imports and interrupted builds.
* Define safe behaviour for CLI commands operating on external folders.

**17 — Future extensions**

* Document potential features without implementing them prematurely.
* Include richer analytics, advanced scheduling, remote build integration if useful, and potential AI-assisted academic workflows.
* Describe how the architecture could accommodate them.

Cross-reference documents where necessary. Avoid repeating large sections of content.

After creating the documents, review them for contradictions, missing requirements and unnecessary complexity.

Do not start the main implementation until the documentation and roadmap have been created and reviewed.

---

# PART II — APPLICATION STRUCTURE

## 6. Initial project layout

Use the following structure as a starting point. Modify it when a clear architectural reason exists and document any significant changes.

```text
academic-manager/
├── AGENTS.md
├── README.md
├── CHANGELOG.md
├── LICENSE
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
│
├── cmd/
│   ├── academic/
│   │   └── main.go
│   └── installer/
│       └── main.go
│
├── internal/
│   ├── app/
│   ├── config/
│   ├── database/
│   ├── domain/
│   ├── services/
│   ├── filesystem/
│   ├── tasks/
│   ├── analytics/
│   ├── timeline/
│   ├── importer/
│   ├── exporter/
│   ├── backup/
│   ├── cli/
│   ├── tui/
│   │   ├── app/
│   │   ├── components/
│   │   ├── screens/
│   │   ├── styles/
│   │   └── navigation/
│   ├── installer/
│   └── logging/
│
├── migrations/
│
├── configs/
│   ├── app.json
│   └── build.json
│
├── scripts/
│   ├── build.bat
│   ├── test.bat
│   ├── package.bat
│   └── release.bat
│
├── docs/
│
├── assets/
│
├── testdata/
│
└── dist/
```

Do not create empty packages just to match the diagram. Create packages when they have a clear responsibility.

Keep reusable application logic outside `cmd/` and `internal/tui/`.

The TUI and CLI must call the same underlying application services.

---

# PART III — ACADEMIC DATA AND STORAGE

## 7. Course unit management

The application must support a configurable number of course units. The initial user has nine, but the application must not impose a permanent limit of nine.

Each course unit should have:

* Unique ID.
* Name.
* Optional course code.
* Description.
* Creation date.
* Optional colour or visual identifier.
* Active or archived status.
* Associated topics, tasks and deadlines.

Users must be able to create, rename, archive and delete units.

Deletion must be safe. Where associated records or files exist, explain the consequences and require explicit confirmation.

Prefer archiving when it can preserve useful academic history.

## 8. Topics and reading progress

Each course unit can contain multiple topics.

A topic should have:

* Unique ID.
* Name.
* Description.
* Course unit association.
* Reading status.
* Priority, if needed.
* Creation and modification timestamps.
* Optional notes.
* Associated files and deadlines.

Use the following reading statuses:

```text
Unread
Pending
Read
```

Users must be able to transition topics between these statuses.

The TUI should make it easy to:

* View all topics within a course unit.
* Filter topics by reading status.
* Change topic status.
* View associated reading materials.
* See topic-level progress.
* Identify topics with upcoming assessments or deadlines.

Do not automatically mark a topic as read merely because a user opens its folder.

## 9. Assignments and coursework

Assignments and coursework must be first-class records rather than ordinary text files.

Each item should support:

* Title.
* Description.
* Course unit.
* Type, such as assignment, coursework, test, examination or project.
* Status.
* Creation date.
* Deadline.
* Completion date, where applicable.
* Priority.
* Associated documents.
* Notes and progress history, where useful.

Suggested statuses:

```text
Not Started
In Progress
Completed
Submitted
Overdue
```

Overdue should preferably be derived from the deadline and completion state rather than being a manually maintained status.

Users must be able to update deadlines, change progress, attach files and review past work.

A completed assignment must remain available in the academic history.

## 10. Physical academic folder structure

The application must maintain a real folder structure on the user's chosen storage drive.

Example:

```text
AcademicData/
├── database/
│   └── academic.db
│
├── config/
│   └── user.json
│
├── backups/
│
├── exports/
│
├── reports/
│
├── units/
│   ├── unit-001/
│   │   ├── unit.json
│   │   ├── topics/
│   │   │   ├── topic-001/
│   │   │   │   ├── topic.json
│   │   │   │   ├── reading/
│   │   │   │   ├── research/
│   │   │   │   └── summaries/
│   │   │   └── topic-002/
│   │   │       ├── topic.json
│   │   │       ├── reading/
│   │   │       └── summaries/
│   │   │
│   │   ├── assignments/
│   │   │   ├── assignment-001/
│   │   │   │   ├── task.json
│   │   │   │   ├── materials/
│   │   │   │   ├── drafts/
│   │   │   │   └── submissions/
│   │   │   └── coursework-001/
│   │   │
│   │   └── notes/
│   │
│   └── unit-002/
│
└── inbox/
```

This is an illustrative structure. Refine it in the storage specification before implementing it.

Use stable IDs for folder names instead of relying exclusively on editable titles. This avoids breaking references when a user renames a unit or topic.

SQLite is the primary source of truth for relationships, status, deadlines and analytics.

JSON sidecar files can hold useful portable metadata, but avoid maintaining two independent sources of truth for the same information. Define how sidecars are updated and recovered.

## 11. File management

The application must support:

* Creating folders and files.
* Renaming files and folders.
* Moving files between academic folders.
* Importing files from any accessible local folder.
* Viewing file metadata.
* Opening a file using the operating system's default application.
* Searching academic files.
* Safely deleting files and folders.
* Identifying missing or externally moved files where practical.

Use Go's filesystem APIs and validate paths carefully.

Never allow a user-supplied path to escape the intended application storage directory through traversal or unsafe path resolution.

When moving or deleting files:

* Check whether the operation affects linked academic records.
* Handle name collisions explicitly.
* Report failures clearly.
* Avoid silently replacing existing files.
* Keep the database and filesystem consistent as far as possible.
* Use temporary files, backups or recoverable operations where appropriate.

The application should support importing multiple files from a selected folder.

Provide a preview of bulk operations when they may affect many files.

---

# PART IV — TUI EXPERIENCE

## 12. General visual direction

Build a clean, practical and compact terminal interface inspired by modern interactive CLI applications, including OpenCode, Claude Code and Gemini CLI.

Do not copy their complete interfaces.

The visual design should use:

* A light-blue or grey-inspired visual identity, adapted to terminal colour capabilities.
* Subtle borders and separators.
* Consistent spacing.
* Clear status indicators.
* Restrained colour usage.
* Readable text and tables.
* A persistent command input at the bottom.
* A changing content area above the input.
* Context-sensitive keyboard shortcuts.

Support terminals with limited dimensions and colour capabilities.

Use Lipgloss for styling and Bubbles for suitable reusable components.

Avoid unnecessary animations, excessive borders and complex layouts that make the TUI difficult to navigate.

## 13. Main TUI layout

The general layout should resemble:

```text
┌──────────────────────────────────────────────────────────────┐
│ ACADEMIC MANAGER                     Date | Week | Overview  │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  DYNAMIC CONTENT AREA                                        │
│                                                              │
│  Dashboard, course units, topics, files, timeline,           │
│  assignments, analytics, search results, logs, etc.          │
│                                                              │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│ Status: 9 units | 42 topics | 3 pending tasks                │
├──────────────────────────────────────────────────────────────┤
│ > Type a command or use /help                         [Input]│
└──────────────────────────────────────────────────────────────┘
```

This is a conceptual wireframe. Adapt it to actual terminal dimensions.

The command input should remain persistent while the content area changes between screens.

Avoid losing partially typed commands or unsaved input when changing views.

Provide keyboard navigation as well as command-based interaction.

## 14. Main screens

Implement the screens incrementally.

### Home dashboard

Display:

* Total active course units.
* Total topics.
* Read, pending and unread topic counts.
* Course unit coverage.
* Active assignments and coursework.
* Upcoming deadlines.
* Overdue items.
* Current week's study overview.
* Important items requiring attention.

Use compact progress bars, lists and indicators appropriate for a terminal.

The dashboard should be useful without requiring the user to navigate through multiple screens.

### Course units

Display the course unit list with:

* Name and code.
* Topic count.
* Reading coverage.
* Active assignments.
* Upcoming deadlines.

Allow selection of a unit to view its topics, tasks, files and analytics.

### Topic manager

Support topic creation, editing, status transitions, filtering, sorting and access to topic folders.

### Assignment manager

Provide a consolidated view of assignments, coursework, tests and examinations.

Support filtering by unit, status, deadline and priority.

### File explorer

Display the academic storage hierarchy.

Allow navigation, creation, importing, moving, renaming, opening and deletion.

### Timeline

Provide a scrollable, week-oriented timeline.

Display:

* Days of the week.
* Dates.
* Assignment deadlines.
* Tests and examinations.
* Planned study activities, when supported.
* Completed and pending items.
* Overdue items.

The user should be able to move between previous and upcoming weeks.

Selecting an item should open its details, allow editing and provide navigation to its associated course unit or folder.

### Analytics

Show course coverage, topic completion, assignment status, deadlines and progress over time.

Do not use analytics that cannot be explained or reproduced from the underlying data.

### Settings

Provide access to:

* Application configuration.
* Storage path.
* Display preferences.
* Backup and export.
* Database maintenance.
* Application version.
* Help and shortcuts.

---

# PART V — COMMAND SYSTEM

## 15. Persistent command input

The TUI must provide a persistent input box for commands and searches.

Support:

* Command history.
* Basic input editing.
* Autocomplete where practical.
* Help for available commands.
* Clear validation errors.
* Consistent output and status messages.

Do not make every action dependent on remembering commands. Important operations must also be accessible through navigation and keyboard shortcuts.

Use a predictable command structure.

Illustrative commands:

```text
/help
/dashboard
/units
/units add
/units list
/units open <unit-id>
/units rename <unit-id>
/topics add
/topics list
/topics status <topic-id> <status>
/tasks add
/tasks list
/tasks deadlines
/timeline
/timeline next
/timeline previous
/files
/files import <path>
/files move <source> <destination>
/search <query>
/analytics
/export
/backup
/settings
/quit
```

Refine this command set in the CLI specification.

The command parser should not become a large collection of hardcoded string comparisons.

Use Cobra for the standalone CLI if it simplifies command parsing, help generation and argument validation.

For the TUI, reuse the same command definitions or service functions where practical, without coupling the TUI to Cobra internals unnecessarily.

## 16. Standalone CLI

The application must be usable from any terminal directory.

Examples:

```powershell
academic
academic .
academic /help
academic units list
academic topics list
academic tasks deadlines
academic analytics
academic export --format txt
```

The command:

```powershell
academic .
```

should open or invoke the application's context-aware file import workflow for the current working directory.

Do not silently import files merely because the command was executed.

Show the files to be imported, the target academic folder and any potential conflicts before executing a bulk import.

Support explicit non-interactive flags for scripting, such as a destination and confirmation option, while retaining safe defaults.

CLI operations must use the same storage configuration, database and application services as the TUI.

When invoked from another directory, do not accidentally initialise a second database or change the configured storage location.

---

# PART VI — DEADLINES, TIMELINE AND ANALYTICS

## 17. Deadline management

Deadlines should be central to the application's data model.

Support:

* Assignment deadlines.
* Coursework deadlines.
* Test dates.
* Examination dates.
* Other academic events.
* User-defined deadlines.

Every deadline must be editable.

Changing a deadline must update all affected views and calculations.

Use local dates and times consistently. Store timestamps in a clearly defined format and document the timezone policy.

Allow deadlines to have:

* A title.
* An associated academic item.
* A date and optional time.
* An optional description.
* An optional priority.
* A completion or submission state.

Avoid hardcoding assumptions about the academic calendar.

## 18. Progress and deadline awareness

The dashboard should help the user understand the relationship between their academic coverage and upcoming deadlines.

For example:

* A topic with an approaching test and an unread status should be visibly highlighted.
* An overdue assignment that is incomplete should be identifiable.
* A course unit with many unread topics and a nearby examination should receive a clear attention indicator.
* Completed or submitted work should not continue to appear as incomplete.

Use simple, documented rules for attention indicators.

Suggested labels:

```text
On Track
Attention
At Risk
Overdue
Completed
```

These are descriptive planning indicators, not predictions of academic performance.

Do not present arbitrary percentages as evidence that the user will pass or fail an assessment.

If a risk heuristic is used, explain its inputs and limitations. Prefer transparent rules that the user can understand.

## 19. Course coverage

Calculate topic coverage from topic statuses.

For example:

```text
Total topics = 20
Read = 8
Pending = 5
Unread = 7

Reading coverage = 8 / 20 × 100 = 40%
```

Clearly distinguish reading completion from general academic preparedness.

A topic being marked as read does not prove mastery.

Show:

* Total topics.
* Read topics.
* Pending topics.
* Unread topics.
* Reading coverage.
* Active assignments.
* Completed assignments.
* Upcoming deadlines.

Define how empty units are represented. Do not divide by zero or misleadingly report 100% coverage for a unit with no topics.

Make the coverage formula consistent throughout the application.

## 20. Timeline calculations

The timeline must combine deadlines from different academic categories.

Provide:

* Weekly views.
* Navigation between weeks.
* Grouping by date.
* Sorting by time and priority.
* Indicators for overdue and completed items.
* A detailed view for each event.

Allow deadlines to be edited directly from the timeline.

Keep date calculations in a dedicated package rather than duplicating date logic across TUI screens.

---

# PART VII — CONFIGURATION, INSTALLER AND UPDATES

## 21. Local storage selection

On first launch, the application must guide the user through initial configuration.

Allow the user to select:

* Application data directory.
* Database location, preferably within that directory.
* Academic materials directory, preferably within the same storage root.
* Optional backup location.

The application should work with paths on different local drives.

Do not require administrator privileges for normal installation.

Keep the selected storage path in a clearly defined configuration location.

The application must distinguish between:

* Program installation directory.
* Application configuration directory.
* Academic data directory.
* Temporary working directory.

Never delete or replace academic data during an application update.

## 22. Separate installer executable

The repository must produce two Windows executables:

```text
academic.exe
academic-installer.exe
```

The installer must be built from the same repository using Go.

The installer should provide a simple interactive setup experience.

Initial installer responsibilities:

* Display application identity and version.
* Allow selection of the installation directory.
* Allow selection of the academic storage directory.
* Create the required configuration and folder structure.
* Initialise the database and migrations.
* Create application shortcuts where practical.
* Offer optional PATH integration.
* Detect an existing installation.
* Preserve existing academic data.
* Provide a clear completion message.

The installer should not embed a complete duplicate copy of the application source or unnecessarily duplicate large resources.

Use a documented, reproducible packaging method for distributing the application and installer together.

The application should support a portable installation mode in the future, but this should not delay the first usable release.

## 23. Updating

Design the installer and application for future updates.

An update must:

* Detect the current application version.
* Replace program files without overwriting academic data.
* Apply compatible database migrations.
* Preserve the storage location.
* Provide a recoverable path if an update fails.
* Report the installed and target versions.

Do not implement a complex automatic update service in the first version.

Prioritise a reliable manual update workflow through a new installer.

## 24. PATH integration

Allow the user to add the application's CLI executable to the user's PATH.

Do not modify system-wide environment variables unless explicitly requested and authorised.

Provide clear instructions for refreshing the terminal session.

Support running commands from any directory after PATH integration.

The installer must be safe to run more than once.

---

# PART VIII — EXPORTS, BACKUPS AND PORTABILITY

## 25. Academic status reports

The application must be able to export a plain-text summary of the user's academic situation.

This is important because the user may carry a phone to school instead of a PC.

Example report:

```text
ACADEMIC STATUS REPORT
Generated: 2026-10-02

COURSE UNITS
----------------------------

Unit: Course Unit A
Topics: 20
Read: 8
Pending: 5
Unread: 7
Coverage: 40%

Active assignments: 2
Upcoming deadlines: 1

TOPICS REQUIRING ATTENTION
----------------------------

1. Topic A - Unread
2. Topic B - Pending

UPCOMING DEADLINES
----------------------------

2026-10-05 - Assignment A
2026-10-09 - Test A

OVERDUE ITEMS
----------------------------

None

GENERAL SUMMARY
----------------------------

Total units: 9
Total topics: 120
Read: 48
Pending: 30
Unread: 42
Overall reading coverage: 40%
```

This is only an illustrative report.

The export must be generated from current database records, not manually maintained text.

Support:

* Overall status.
* Per-unit reports.
* Upcoming deadlines.
* Overdue items.
* Reading progress.
* Assignment status.

Use a human-readable format suitable for copying or sending to a phone.

Also support JSON export for future integrations.

## 26. Backups

Implement reliable local backups.

At minimum:

* Back up the SQLite database consistently.
* Include academic files when performing a full backup.
* Record backup creation time and application version.
* Provide a restoration workflow.
* Avoid overwriting an existing backup without confirmation.
* Verify backup integrity where practical.

Use SQLite's appropriate backup facilities or another documented consistent backup method rather than copying an actively changing database blindly.

Make backup and restore operations available through both the TUI and CLI.

---

# PART IX — IMPLEMENTATION ROADMAP

## 27. Development phases

Use the following phases as the initial implementation plan. Refine them in `docs/15_IMPLEMENTATION_ROADMAP.md` before implementation.

### Phase 0 — Repository and documentation

* Inspect the repository.
* Create the Markdown documentation.
* Establish project scope and architecture.
* Create the implementation roadmap.
* Review for consistency.

**Acceptance:** Documentation is complete, internally consistent and actionable.

### Phase 1 — Go foundation

* Initialise the Go module.
* Establish application metadata.
* Create the initial package structure.
* Add logging and error handling conventions.
* Implement the executable entry point.
* Add basic tests.

**Acceptance:** The application compiles and starts with a basic help or welcome screen.

### Phase 2 — Configuration and storage

* Implement configuration loading and saving.
* Support configurable data directories.
* Initialise SQLite.
* Implement database migrations.
* Create storage directories.
* Add data integrity tests.

**Acceptance:** The application can initialise, reopen and use its configured storage without losing data.

### Phase 3 — Core academic entities

* Implement course units.
* Implement topics.
* Implement assignments and coursework.
* Implement deadlines.
* Implement basic CRUD operations through application services.

**Acceptance:** The core entities can be created, viewed, edited, archived or safely deleted through tests and CLI operations.

### Phase 4 — Filesystem integration

* Create academic folder structures.
* Associate folders with academic records.
* Implement import, move, rename and deletion operations.
* Handle conflicts and filesystem failures.
* Add filesystem integration tests.

**Acceptance:** Academic files can be managed without breaking the database or losing existing files during normal operations.

### Phase 5 — Initial TUI

* Implement the main Bubble Tea model.
* Build navigation and reusable components.
* Create the persistent command input.
* Implement the home dashboard.
* Add course unit, topic and task screens.
* Establish the application's visual style.

**Acceptance:** The user can navigate the application and manage core academic records through the TUI.

### Phase 6 — Timeline and analytics

* Implement coverage calculations.
* Implement deadline-aware indicators.
* Build the weekly timeline.
* Add dashboard analytics.
* Support timeline editing.

**Acceptance:** Dashboard and timeline values match the underlying records and update correctly after changes.

### Phase 7 — CLI and command system

* Implement CLI commands.
* Integrate command parsing with application services.
* Support working-directory context.
* Implement safe bulk import previews.
* Add help, validation and command history.

**Acceptance:** Core functionality can be accessed through both the TUI and standalone CLI.

### Phase 8 — Export, import and backup

* Implement plain-text reports.
* Implement JSON exports.
* Add backups and restore.
* Test recovery and data integrity.

**Acceptance:** Academic information can be exported, backed up and restored successfully.

### Phase 9 — Installer

* Build the separate installer executable.
* Implement first-run setup.
* Add directory selection.
* Add PATH integration.
* Support repeat installation.
* Preserve existing data.

**Acceptance:** A fresh installation can be completed on a Windows machine without requiring the Go toolchain.

### Phase 10 — Release preparation

* Add version injection.
* Finalise build scripts.
* Test packaging.
* Verify CLI and TUI operations.
* Run migration, backup and restoration tests.
* Create release notes and a sample release package.

**Acceptance:** Both executables build reproducibly, and the application passes the documented release checks.

Do not mark a phase complete merely because its code has been written. Run the relevant tests and verify the acceptance criteria.

If a feature proves too complex for the first usable version, document the issue, implement a simpler safe alternative and update the roadmap.

---

# PART X — TESTING AND CODE QUALITY

## 28. Testing requirements

Write tests as features are implemented.

Use:

* Unit tests for calculations and domain logic.
* Integration tests for SQLite operations.
* Filesystem tests using temporary directories.
* CLI tests for command behaviour.
* TUI model tests for state transitions and navigation.
* Installer tests for configuration and storage setup.
* End-to-end tests for essential workflows.

At minimum, test the following workflows:

1. Create a course unit.
2. Add topics to it.
3. Change topic statuses.
4. Verify reading coverage.
5. Create an assignment.
6. Add and modify its deadline.
7. Import files into its folder.
8. View the unit's dashboard and timeline entries.
9. Export an academic status report.
10. Back up and restore the database and associated files.

Test failure scenarios:

* Invalid configuration.
* Unavailable storage directory.
* Database migration failure.
* Duplicate filenames.
* Missing files.
* Interrupted file operations.
* Invalid command arguments.
* Malformed imports.
* Invalid or past deadlines where relevant.

Run:

```powershell
go test ./...
go vet ./...
go build ./...
```

Use Windows-specific integration testing for the installer and filesystem operations where necessary.

Document any test that cannot be run in the current environment and why.

## 29. Code conventions

* Use idiomatic Go.
* Prefer small, cohesive functions.
* Use explicit error handling.
* Avoid unnecessary interfaces.
* Keep business logic outside the TUI.
* Avoid global mutable application state.
* Use context cancellation for long-running processes and operations.
* Avoid blocking the Bubble Tea event loop.
* Use transactions for related database changes.
* Keep package dependencies clear.
* Add comments to explain non-obvious behaviour, not obvious code.
* Avoid speculative generic frameworks and premature plugin systems.

Do not suppress errors simply to keep the interface running.

Present understandable user-facing errors while preserving useful diagnostic information in logs.

---

# PART XI — GIT AND AGENT WORKFLOW

## 30. Git checkpoints

The user will create Git commits after major stages are completed.

Do not automatically commit, push or tag changes unless explicitly instructed.

At the end of each major phase:

1. Run the relevant tests.
2. Check formatting and static analysis.
3. Inspect the changed files.
4. Update the relevant documentation.
5. Confirm that the acceptance criteria are satisfied.
6. Provide a concise summary of completed work.
7. List known limitations and the next phase.
8. Stop at the milestone so the user can inspect and commit the work.

Avoid accumulating unrelated changes across phases.

Use descriptive commit message suggestions, such as:

```text
docs: establish project architecture and roadmap
feat: implement configuration and SQLite storage
feat: add academic unit and topic management
feat: implement academic TUI dashboard
feat: add deadline timeline and analytics
feat: add installer and release packaging
```

These are suggestions for the user, not commands to commit automatically.

## 31. Working with limited context

The project may be developed by a long-running coding agent with limited context.

Maintain these files:

```text
AGENTS.md
docs/15_IMPLEMENTATION_ROADMAP.md
docs/IMPLEMENTATION_STATUS.md
docs/DECISIONS.md
```

Create `IMPLEMENTATION_STATUS.md` and `DECISIONS.md` when implementation begins.

`IMPLEMENTATION_STATUS.md` should contain:

* Current phase.
* Completed tasks.
* Incomplete tasks.
* Test status.
* Current blockers.
* Next exact implementation step.

`DECISIONS.md` should record:

* Important architecture decisions.
* Why a decision was made.
* Alternatives considered.
* Relevant trade-offs.
* Any decision that may need revisiting.

At the beginning of each new working session, read `AGENTS.md`, the roadmap, implementation status and relevant architecture documents.

Update implementation status at each major checkpoint.

Do not repeat completed work unless a defect requires it.

## 32. Handling uncertainty

Do not invent requirements when the specification is ambiguous.

For minor uncertainties:

* Choose the simplest safe default.
* Document the assumption.
* Keep the design easy to change.

For significant uncertainties that could affect the database, data safety, architecture or user data, document the issue and ask the user before proceeding.

Do not introduce irreversible operations based on an assumption.

Do not wait for user input for every minor design decision.

## 33. Managing scope

The first usable version must prioritise:

1. Reliable academic data storage.
2. Course unit and topic management.
3. Reading progress.
4. Assignment and deadline management.
5. File organisation.
6. A functional TUI.
7. Dashboard and timeline.
8. CLI operations.
9. Plain-text reports.
10. Backup and restoration.
11. Installer and reproducible packaging.

Advanced analytics, extensive customisable dashboards, AI integration, calendar synchronisation, remote access, cloud storage and sophisticated scheduling are future extensions unless they are essential to the initial implementation.

Do not allow future features to delay a reliable first version.

---

# PART XII — BUILD AND RELEASE CONFIGURATION

## 34. Build configuration

Keep build-related configuration centralised.

Include:

* Application name.
* Version.
* Git commit identifier.
* Build timestamp, if useful.
* Go build flags.
* Target operating system and architecture.
* Output directory.
* Executable names.
* Packaging settings.

Support reproducible development and release builds.

Provide simple Windows commands:

```powershell
.\scripts\test.bat
.\scripts\build.bat
.\scripts\package.bat
```

Use Go linker flags where appropriate to inject version metadata.

Do not require Python, Node.js, a web frontend toolchain or a heavy IDE to build this application.

Keep the basic development workflow compatible with a Windows 11 machine with limited RAM and storage.

## 35. Versioning

Use semantic versioning for stable releases:

```text
MAJOR.MINOR.PATCH
```

For example:

```text
0.1.0
0.2.0
0.2.1
1.0.0
```

Document how to:

* Change the application version.
* Update the changelog.
* Run release tests.
* Build both executables.
* Package release files.
* Create a Git tag.
* Publish a GitHub release.

Do not assume the user wants every build published automatically.

## 36. Release package

The release package should include:

```text
release/
├── academic.exe
├── academic-installer.exe
├── README.txt
├── CHANGELOG.txt
└── checksums.txt
```

Adjust the exact package contents based on the installation model selected during planning.

Ensure the user can distinguish the installer from the standalone CLI/TUI executable.

Provide a clear first-run experience and avoid requiring manual configuration for ordinary use.

---

# FINAL INSTRUCTIONS

Begin with **Phase 0: Repository inspection and documentation**.

Do not immediately generate the entire codebase.

First:

1. Inspect the repository and the available Go toolchain.
2. Create the requested Markdown documentation.
3. Refine the folder structure and architecture.
4. Define the MVP and implementation milestones.
5. Create the implementation status and decision-tracking documents.
6. Review the documentation for consistency.

Then report the completed planning phase, the finalised architecture and the exact next implementation step.

After the user has reviewed and committed the documentation, continue with Phase 1 and implement the project incrementally.

The objective is to deliver a reliable, usable local academic management application, not simply to produce a large volume of source code.

**Prioritise working features, data safety, clear architecture and a development process that the user can understand, maintain and extend independently.**
