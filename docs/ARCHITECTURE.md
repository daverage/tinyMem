tinyMem Architecture

    Overview

    tinyMem is a persistent memory system for Large Language Models (LLMs) that provides evidence-based truth validation. It operates as both a proxy server and a Model Context
    Protocol (MCP) server, allowing LLMs to access historical information and validate claims.

    Core Architecture

      1 ┌─────────────────────────────────────────────────────────────────────────────┐
      2 │                              tinyMem System                                 │
      3 ├─────────────────────────────────────────────────────────────────────────────┤
      4 │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐            │
      5 │  │   Proxy Server  │  │     MCP         │  │   CLI Tools     │            │
      6 │  │                 │  │   Server        │  │                 │            │
      7 │  │  /v1/chat/      │  │  stdin/stdout   │  │  tinymem cmd   │            │
      8 │  │  completions    │  │                 │  │                 │            │
      9 │  └─────────────────┘  └─────────────────┘  └─────────────────┘            │
     10 │         │                       │                       │                  │
     11 │         ▼                       ▼                       ▼                  │
     12 │  ┌─────────────────────────────────────────────────────────────────────┐   │
     13 │  │                         Main Application                          │   │
     14 │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │   │
     15 │  │  │   Core Module   │  │  Project        │  │  Server         │   │   │
     16 │  │  │                 │  │  Module         │  │  Module         │   │   │
     17 │  │  │ - Config        │  │  - Path         │  │  - Mode        │   │   │
     18 │  │  │ - Logger        │  │  - ID           │  │                 │   │   │
     19 │  │  │ - DB            │  │                 │  │                 │   │   │
     20 │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘   │   │
     21 │  └─────────────────────────────────────────────────────────────────────┘   │
     22 │                             │                                               │
     23 │                             ▼                                               │
     24 │  ┌─────────────────────────────────────────────────────────────────────┐   │
     25 │  │                        Services Layer                             │   │
     26 │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │   │
     27 │  │  │  Memory         │  │  Evidence       │  │  Recall         │   │   │
     28 │  │  │  Service        │  │  Service        │  │  Engine         │   │   │
     29 │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘   │   │
     30 │  │         │                       │                       │         │   │
     31 │  │         ▼                       ▼                       ▼         │   │
     32 │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │   │
     33 │  │  │  Storage        │  │  Verification   │  │  Semantic       │   │   │
     34 │  │  │  (SQLite)       │  │  Logic          │  │  Engine         │   │   │
     35 │  │  └─────────────────┘  └─────────────────┘  └─────────────────┘   │   │
     36 │  └─────────────────────────────────────────────────────────────────────┘   │
     37 └─────────────────────────────────────────────────────────────────────────────┘

    Key Components

    1. Main Application Structure

     - App: The central application object that coordinates all components
       - CoreModule: Contains configuration, logger, and database connection
       - ProjectModule: Manages project-specific information
       - ServerModule: Tracks server mode (proxy, MCP, or standalone)

    2. Memory System

     - Memory Types:
       - Fact (requires evidence)
       - Claim (unverified assertion)
       - Plan (future intentions)
       - Decision (choices made)
       - Constraint (limitations)
       - Observation (noted facts)
       - Note (miscellaneous information)
       - Task (action items)

     - Recall Tiers:
       - Always (facts and constraints)
       - Contextual (decisions and claims)
       - Opportunistic (observations, notes, plans)

     - Truth States:
       - Verified (confirmed with evidence)
       - Asserted (claimed but not verified)
       - Tentative (provisional)

    3. tinyTasks System

    One of the key features of tinyMem is the tinyTasks system, which stores and manages task information in a dedicated file called tinyTasks.md. This system allows for
    persistent tracking of tasks across sessions:

      1 ┌─────────────────────────────────────────────────────────────────────────────┐
      2 │                              tinyTasks System                               │
      3 ├─────────────────────────────────────────────────────────────────────────────┤
      4 │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐            │
      5 │  │  Task Parser    │  │  Task Manager   │  │  Task Tracker   │            │
      6 │  │                 │  │                 │  │                 │            │
      7 │  │  Reads/Writes   │  │  Creates/       │  │  Updates/       │            │
      8 │  │  tinyTasks.md   │  │  Updates Tasks  │  │  Queries Tasks  │            │
      9 │  └─────────────────┘  └─────────────────┘  └─────────────────┘            │
     10 │         │                       │                       │                  │
     11 │         ▼                       ▼                       ▼                  │
     12 │  ┌─────────────────────────────────────────────────────────────────────┐   │
     13 │  │                    tinyTasks.md File                              │   │
     14 │  │  # Tasks – <Goal>                                                 │   │
     15 │  │  - [ ] Top-level task                                             │   │
     16 │  │    - [ ] Atomic subtask                                           │   │
     17 │  │    - [ ] Atomic subtask                                           │   │
     18 │  │  - [x] Completed task                                             │   │
     19 │  │    - [x] Completed subtask                                        │   │
     20 │  └─────────────────────────────────────────────────────────────────────┘   │
     21 └─────────────────────────────────────────────────────────────────────────────┘

     - File Location: Stored in tinyTasks.md in the project root
     - Structure: Hierarchical task lists with checkboxes indicating completion status
     - Integration: Tasks can be stored as special memory type in the database
     - Persistence: Survives application restarts and provides continuity
     - Format:

     1   # Tasks – <Goal>
     2
     3   - [ ] Top-level task
     4     - [ ] Atomic subtask
     5     - [ ] Atomic subtask
     6   - [x] Completed task
     7     - [x] Completed subtask
     - Safety Filtering: Unfinished dormant tasks are filtered out unless explicitly requested

    4. Ralph Mode (Autonomous Repair System)

    The Ralph system is an autonomous repair loop that can execute evidence-gated repairs with bounded retries:

      1 ┌─────────────────────────────────────────────────────────────────────────────┐
      2 │                                Ralph Mode                                 │
      3 ├─────────────────────────────────────────────────────────────────────────────┤
      4 │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐            │
      5 │  │   Execute       │  │   Evidence      │  │   Recall        │            │
      6 │  │   Phase         │  │   Phase         │  │   Phase         │            │
      7 │  │                 │  │                 │  │                 │            │
      8 │  │  Run command    │  │  Verify        │  │  Retrieve       │            │
      9 │  │  and capture    │  │  evidence       │  │  relevant       │            │
     10 │  │  output         │  │  predicates     │  │  memories       │            │
     11 │  └─────────────────┘  └─────────────────┘  └─────────────────┘            │
     12 │         │                       │                       │                  │
     13 │         ▼                       ▼                       ▼                  │
     14 │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐            │
     15 │  │   Repair        │  │   Human         │  │   Loop          │            │
     16 │  │   Phase         │  │   Gate          │  │   Control       │            │
     17 │  │                 │  │                 │  │                 │            │
     18 │  │  Apply fixes    │  │  Approval      │  │  Max iterations│            │
     19 │  │  based on       │  │  if needed      │  │  and safety     │            │
     20 │  │  memories       │  │                 │  │  checks         │            │
     21 │  └─────────────────┘  └─────────────────┘  └─────────────────┘            │
     22 └─────────────────────────────────────────────────────────────────────────────┘

     - Purpose: Executes evidence-gated repair loops with bounded autonomous retries
     - Phases:
       - Execute: Runs a verification command and captures output
       - Evidence: Checks if evidence predicates are satisfied
       - Recall: Retrieves relevant memories to inform repairs
       - Repair: Applies fixes based on memories and previous output
     - Safety Features:
       - Maximum iteration limits
       - Forbidden path protection
       - Command whitelisting
       - Human approval gates
     - Evidence Predicates:
       - cmd_exit0: Command exits with code 0
       - test_pass: Test passes successfully
       - file_exists: File exists
       - grep_hit: Pattern found in file

    5. Available Tools (MCP Interface)

    tinyMem provides a rich set of tools accessible through the MCP interface:

      1 ┌─────────────────────────────────────────────────────────────────────────────┐
      2 │                               MCP Tools                                     │
      3 ├─────────────────────────────────────────────────────────────────────────────┤
      4 │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐            │
      5 │  │  memory_query   │  │  memory_recent  │  │  memory_write   │            │
      6 │  │                 │  │                 │  │                 │            │
      7 │  │  Search        │  │  Retrieve       │  │  Create new     │            │
      8 │  │  memories      │  │  recent         │  │  memory         │            │
      9 │  └─────────────────┘  └─────────────────┘  └─────────────────┘            │
     10 │         │                       │                       │                  │
     11 │         ▼                       ▼                       ▼                  │
     12 │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐            │
     13 │  │  memory_stats   │  │  memory_health  │  │  memory_doctor  │            │
     14 │  │                 │  │                 │  │                 │            │
     15 │  │  Get memory    │  │  Check health   │  │  Run           │            │
     16 │  │  statistics    │  │  status         │  │  diagnostics    │            │
     17 │  └─────────────────┘  └─────────────────┘  └─────────────────┘            │
     18 │         │                       │                                          │
     19 │         ▼                       ▼                                          │
     20 │  ┌─────────────────┐  ┌─────────────────┐                                  │
     21 │  │  memory_ralph   │  │  Other tools   │                                  │
     22 │  │                 │  │                 │                                  │
     23 │  │  Execute       │  │  Future         │                                  │
     24 │  │  autonomous    │  │  extensions     │                                  │
     25 │  │  repair loop   │  │                 │                                  │
     26 │  └─────────────────┘  └─────────────────┘                                  │
     27 └─────────────────────────────────────────────────────────────────────────────┘

     - memory_query: Search memories using full-text or semantic search
     - memory_recent: Retrieve the most recent memories
     - memory_write: Create a new memory entry with optional evidence
     - memory_stats: Get statistics about stored memories
     - memory_health: Check the health status of the memory system
     - memory_doctor: Run diagnostics on the memory system
     - memory_ralph: Execute an evidence-gated repair loop with memory-assisted recall

    6. Storage Layer

     - SQLite Database: Stores memories, evidence, embeddings, and recall metrics
     - Schema Versioning: Automatic migration system with version tracking
     - Triggers: Maintains Full-Text Search (FTS) index synchronization
     - Constraints: Enforces business rules at the database level

    7. Evidence System

     - Evidence Types:
       - file_exists (checks if a file exists)
       - grep_hit (searches for patterns in files)
       - cmd_exit0 (runs commands and checks exit codes)
       - test_pass (runs tests and verifies success)

     - Verification Process: Validates evidence before promoting claims to facts

    8. Recall Engine

     - Search Methods: Combines FTS5 and traditional LIKE-based search
     - Scoring Algorithm: Ranks memories based on relevance to query
     - Tier-Based Filtering: Applies recall limits based on memory importance
     - Truth State Prioritization: Prefers verified memories over tentative ones

    9. Chain-of-Verification (CoVe)

     - Candidate Verification: Filters memory candidates based on confidence scores
     - Recall Filtering: Removes irrelevant memories from recall results
     - Confidence Threshold: Configurable threshold for accepting memories
     - Statistics Tracking: Monitors verification effectiveness

    Data Flow

      1 ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
      2 │   LLM Request   │───▶│  Proxy/MCP      │───▶│  Memory         │
      3 │                 │    │  Server         │    │  Injection      │
      4 └─────────────────┘    └─────────────────┘    └─────────────────┘
      5                             │                           │
      6                             ▼                           ▼
      7                     ┌─────────────────┐    ┌─────────────────┐
      8                     │  Recall         │───▶│  Memories       │
      9                     │  Engine         │    │  Retrieved      │
     10                     └─────────────────┘    └─────────────────┘
     11                             │                           │
     12                             ▼                           ▼
     13                     ┌─────────────────┐    ┌─────────────────┐
     14                     │  CoVe           │───▶│  Evidence        │
     15                     │  Verification   │    │  Validation      │
     16                     └─────────────────┘    └─────────────────┘
     17                             │                           │
     18                             ▼                           ▼
     19                     ┌─────────────────┐    ┌─────────────────┐
     20                     │  tinyTasks      │───▶│  Task Memory     │
     21                     │  Integration    │    │  Storage         │
     22                     └─────────────────┘    └─────────────────┘
     23                             │                           │
     24                             ▼                           ▼
     25                     ┌─────────────────┐    ┌─────────────────┐
     26                     │  Ralph Mode     │───▶│  Autonomous      │
     27                     │  (Repair Loop)  │    │  Operations      │
     28                     └─────────────────┘    └─────────────────┘

    Operational Pathways

    1. Proxy Server Pathway

     1 1. LLM sends request to /v1/chat/completions
     2 2. Proxy extracts user message
     3 3. Recall engine searches for relevant memories (including tasks)
     4 4. CoVe filters memories (if enabled)
     5 5. Task safety filtering applied
     6 6. Memories injected into system message
     7 7. Request forwarded to backend LLM
     8 8. Response captured for memory extraction
     9 9. Auto-extraction identifies new memories

    2. MCP Server Pathway

     1 1. MCP client sends tool call
     2 2. Server routes to appropriate handler
     3 3. Memory operations performed (including task management)
     4 4. Tools list returned when requested
     5 5. Results formatted as MCP response
     6 6. Response sent via stdout

    3. CLI Operations Pathway

     1 1. User runs tinymem command
     2 2. App initialized with config
     3 3. Services instantiated
     4 4. Operation performed (query, write, etc.)
     5 5. Results displayed to user

    4. Memory Creation Pathway

     1 1. Memory submitted (via proxy, MCP, or CLI)
     2 2. Type validation occurs
     3 3. Evidence verification (for facts)
     4 4. Supersession logic applied
     5 5. Stored in database with appropriate tier/state
     6 6. FTS index updated

    5. Memory Retrieval Pathway

     1 1. Query received (empty = recent, non-empty = search)
     2 2. FTS5 search performed (fallback to LIKE)
     3 3. Results scored by relevance
     4 4. Tier-based filtering applied
     5 5. Truth state prioritization
     6 6. Token and item limits enforced
     7 7. CoVe filtering applied (if enabled)
     8 8. Task safety filtering applied
     9 9. Results returned

    6. tinyTasks Management Pathway

     1 1. Task created/updated via CLI or MCP
     2 2. Task stored as special memory type
     3 3. Task state synchronized with tinyTasks.md file
     4 4. Task recall prioritized based on completion status
     5 5. Task safety filtering applied (unfinished dormant tasks filtered out)

    7. Ralph Mode Pathway

     1 1. memory_ralph tool called with task, command, and evidence
     2 2. Execute phase: Run verification command
     3 3. Evidence phase: Check if evidence predicates are satisfied
     4 4. If not satisfied:
     5    a. Recall phase: Retrieve relevant memories
     6    b. Repair phase: Apply fixes based on memories
     7    c. Repeat until evidence satisfied or max iterations reached
     8 5. Return results with iteration log and final diff

    Key Features

     1. Persistent Memory: Memories stored in SQLite database survive application restarts
     2. Evidence-Based Validation: Facts require verified evidence to be created
     3. Tiered Recall: Different types of memories recalled based on importance
     4. CoVe Integration: Chain-of-Verification filters memories for quality
     5. Autonomous Repair: Ralph system fixes issues with evidence verification
     6. Multiple Interfaces: Proxy, MCP, and CLI access methods
     7. Safety Mechanisms: Path validation, command whitelisting, and iteration limits
     8. Metrics Collection: Tracks recall effectiveness and system performance
     9. Task Management: Integrated tinyTasks system for persistent task tracking
     10. Task Safety: Filters out unfinished dormant tasks unless explicitly requested
     11. Rich Toolset: MCP provides multiple tools for memory operations
     12. Ralph Mode: Autonomous repair loop with evidence-gated execution
     13. Configurable: Extensive configuration options for all features

    This architecture enables tinyMem to serve as a persistent, verifiable memory system for LLMs, enhancing their ability to maintain context and learn from past interactions
    while maintaining data integrity through evidence-based validation. The tinyTasks integration provides a persistent task management system that allows for long-term project
    continuity and task tracking. The Ralph mode offers an autonomous repair capability with safety controls, and the rich toolset provides flexible access to memory operations
    through the MCP interface.
