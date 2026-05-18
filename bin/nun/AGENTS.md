# nun Agent Notes

`nun` is allowed to change this Nix config, so every command that writes files must use the same confirmation flow:

1. Collect all required inputs first.
2. Print a short summary before writing anything.
3. List every file that will be added, modified, or deleted.
4. Ask for explicit `y` confirmation.
5. Abort without writing files for any other answer.

Keep file-generation logic small and shared. Prefer common helpers for prompts, summaries, confirmation, and file writes instead of one-off command code.
