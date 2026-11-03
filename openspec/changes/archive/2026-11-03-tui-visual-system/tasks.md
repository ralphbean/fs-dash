## 1. Theme and configuration

- [x] 1.1 Add validated dark/light theme and auto/never color preferences with defaults that preserve existing config files.
- [x] 1.2 Create a central semantic palette and phase/log-level style mapping, including `NO_COLOR` behavior.
- [x] 1.3 Test that every status has a distinct treatment and that no-color output retains labels and focus markers.

## 2. Framed views

- [x] 2.1 Build reusable header, bordered panel, divider, selected-row, badge, context, and footer rendering helpers.
- [x] 2.2 Apply the visual system to PR list sections and selected-item context without hiding field status on the selected row.
- [x] 2.3 Apply the same system to the preliminary details, help, and log views, including severity treatments.

## 3. Responsive behavior and verification

- [x] 3.1 Measure styled cell width and height, keep the selected row in view after resize, and add a compact layout for narrow terminals.
- [x] 3.2 Test 80x24 and narrow-terminal rendering for unintended wraps, clipped panels, and selected-row visibility.
- [x] 3.3 Run the live TUI in dark, light, and no-color modes and verify contrast, borders, navigation, and log readability.
