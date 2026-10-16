
You would Draw it inside 

Explain as if Teaching a Child
Remove Unnecessary Details
Use Everyday Language 
Break the Topic into Small Parts

---

# Sage Skill: Architecture Mapping

## Mission & Rules
Map codebase architecture into `SAGE.md` so human users can visually inspect structure and technical debt in the `sage` browser UI.
- **Never Assume Anything**: The AI must never assume facts about the codebase. Do not guess file paths, dependencies, or patterns. Verify directly via `git status`, file reads, and imports.
- **Pedagogical Clarity**: Explain as if teaching a child. Remove unnecessary details, use everyday language, break topics into small parts.

## SAGE.md Syntax Cheat-Sheet
```sage
// group: GroupName                  // Change group name to categorize nodes into a cluster
// path: path/to/source              // Change path to map this cluster to actual source directory
rect id "Simple Label" [blue]       // Change shape (rect/circle/diamond/triangle/cylinder) or [color] to alter rendering
cylinder db "Database" [purple]     // Change [purple] to [blue/green/red/yellow/orange/gray] for different palette color
id -> db "Reads data"               // Change arrow -> to <-> for bidirectional sync; change label for protocol
id <-> db "Bi-directional sync"
```
See [lang-docs.md](../../lang-docs.md) for full language rules.

## Incremental Update Protocol
When code changes after `SAGE.md` exists:
1. **Never regenerate the whole file.**
2. Run `git status` and `git diff` to locate modified files and packages.
3. Locate corresponding `// group:` or node IDs in `SAGE.md`.
4. Surgically update or append only the touched nodes and connections.
5. Remind the user to run `sage` in their terminal to view the updated diagram.
