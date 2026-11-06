## 1. Stabilize quota observations

- [x] 1.1 Reject out-of-order quota observations within the same resource and reset window, while accepting a later reset window.
- [x] 1.2 Add model coverage for late lower-used responses, equal counters, and reset-window rollover.

## 2. Render a relative reset countdown

- [x] 2.1 Format reset duration with rounded-up seconds, minutes, hours, or days, using full wording in normal layout and compact wording in narrow layout.
- [x] 2.2 Refresh the countdown until reset and show `reset now` after the reset time passes.
- [x] 2.3 Add UI coverage for unit boundaries, singular/plural wording, compact output, and elapsed resets.

## 3. Verify the change

- [x] 3.1 Run the Go tests, build, formatting checks, diff checks, and strict OpenSpec validation.
