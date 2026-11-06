## 1. Derive source-specific review states

- [x] 1.1 Reuse effective current reviews to compute the Human and Bot states independently with the existing precedence and incomplete-history handling.
- [x] 1.2 Update existing table coverage to exercise distinct human and bot outcomes while retaining the shared review-state rendering mappings.

## 2. Render two review columns

- [x] 2.1 Replace the combined Review column with adjacent Human review and Bot review columns, using matching state renderers.
- [x] 2.2 Add descriptive text headers and compact person/robot emoji headers, with width calculations for both display modes and narrow terminals.
- [x] 2.3 Update table-layout coverage for column order, source-specific values, display modes, and compact widths.

## 3. Validate the change

- [x] 3.1 Run the relevant Go tests and full repository checks, then run strict OpenSpec validation and resolve any failures.
- [x] 3.2 Review the table in text and emoji modes at standard and narrow terminal sizes, including no-color mode.
