## Context

See proposal.md for motivation. `Binding` already holds the executable, arguments, working directory, and detach mode; the help view currently uses the executable as the binding label. Existing user TOML files will not all have the new field.

## Goals / Non-Goals

**Goals:** Let config authors give bindings concise labels in help while keeping older config files valid.

**Non-Goals:** Change command execution, placeholder expansion, or help navigation.

## Decisions

- Add an optional, static `summary` string to each binding. Do not expand placeholders in it; it is display text, not a command argument.
- Display the summary when set and keep the executable as a compatibility fallback when it is empty.
- Give the default opener a summary, document the field in the commented example, and add short summaries to the live bindings.

## Risks / Trade-offs

- Empty or overly long summaries can be less helpful than executable paths → Keep the field optional, use the existing executable as fallback, and use concise labels in shipped examples.
