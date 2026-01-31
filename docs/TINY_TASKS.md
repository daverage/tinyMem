# tinyTasks: Autonomous Task Tracking

tinyTasks is a built-in task management system for tinyMem that lives alongside your code in `tinyTasks.md`. It acts as an **autonomous ledger** that ensures AI agents and human operators stay in sync regarding project goals and progress.

## 🧠 The Philosophy: "Intent lives in the File"

Unlike standard project memory, which can be fuzzy, tinyTasks is **authoritative**. If it's not in `tinyTasks.md`, it's not a prioritized task for the agent.

1.  **Human-Led Intent**: tinyMem may automatically create the `tinyTasks.md` file when it detects multi-step work, but it will **refuse to act** until a human edits the file and adds unchecked tasks (`- [ ]`).
2.  **Explicit Sync**: Every time an agent starts a task or finishes a subtask, it MUST update the file. This ensures the "Project Memory" is always grounded in reality.
3.  **Automatic Visualization**: The `tinymem dashboard` reads `tinyTasks.md` and provides a visual progress report.

---

## 🚀 How it Works

### 1. Initial Creation
If you ask an agent to perform a complex task and `tinyTasks.md` is missing, the agent will create a template:

```markdown
# Tasks — NOT STARTED
> No work is authorised until a human edits this file and defines tasks.

## Tasks
<!-- No tasks defined yet -->
```

### 2. Activating the Agent
To authorize the agent to work, you edit the file:

```markdown
# Tasks — Implement User Authentication
- [ ] Create database schema for users
- [ ] Implement JWT login endpoint
- [ ] Add unit tests for auth service
```

### 3. Execution & Tracking
As the agent works, it updates the checkboxes:

```markdown
# Tasks — Implement User Authentication
- [x] Create database schema for users
- [ ] Implement JWT login endpoint
- [ ] Add unit tests for auth service
```

When all checkboxes are marked `[x]`, the task is considered complete.

---

## 📊 Dashboard Integration

The tinyMem dashboard (`tinymem dashboard`) provides a real-time view of your task ledger:

- **Completion Rate**: Total percentage of finished tasks.
- **Section Breakdown**: Progress per major goal.
- **Trend Analysis**: How fast tasks are being completed.

---

## 🛠 CLI Commands

| Command | Action |
|---------|--------|
| `tinymem stats` | Shows high-level task metrics in the terminal. |
| `tinymem dashboard` | Opens/Shows the visual status of all tasks. |
| `tinymem health` | Verifies that the task ledger is correctly synchronized with the memory DB. |

---

## 📜 Safety Rules for Agents

- **Never infer state**: If the file says a task is unchecked, it IS unchecked.
- **Mark completion only when done**: Never "pre-check" a task before executing the code.
- **Respect the Ledger**: The file is the single source of truth. If the database and the file disagree, the file wins.

---

## FAQ

**Q: Can I use multiple task files?**
A: No, tinyMem specifically looks for `tinyTasks.md` in the project root to maintain a single, unambiguous source of truth.

**Q: What happens if I delete a completed task from the file?**
A: tinyMem will eventually remove it from its internal analytics as well, but it's better to keep completed tasks for historical context until the goal is fully achieved.

**Q: Does it support hierarchical tasks?**
A: Yes! You can use nested lists to group atomic subtasks under a main task.
