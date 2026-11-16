## Context

The thread view already has an outdated flag and displays the original line number in its location line. The visual theme defines a warning role that is bold and adapts to light, dark, and no-color modes.

## Goals / Non-Goals

**Goals:**
- Make the existing outdated state easy to spot near the top of the selected thread.
- Keep the status legible without color.

**Non-Goals:**
- Change how GitHub outdated state is fetched or determined.
- Change historical diff context or thread navigation.

## Decisions

- Add a warning-symbol prefix to the existing “Outdated” location text and paint that complete label with the existing warning role. This keeps the marker in the current location summary and reuses the theme's established warning palette.
- Leave the label text and symbol present in no-color mode so color is not its only meaning.

## Risks / Trade-offs

- [Unicode symbol width can vary by terminal] → Keep the marker short and use the existing width-aware rendering path.
