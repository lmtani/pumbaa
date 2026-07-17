# Terminal demo for the README, recorded with vhs (https://github.com/charmbracelet/vhs).
#
# Prerequisites:
#   - `vhs`, `ttyd` and `ffmpeg` installed, and `pumbaa` on PATH
#   - a reachable Cromwell server with a few workflows. The easiest way to get
#     one is the showcase: see example/showcase/README.md — start the local
#     Cromwell and run ./run.sh. This tape targets it on :8010 (edit below to
#     point elsewhere).
#
# Record from the repository root:
#   vhs docs/assets/demo.tape

Output docs/assets/demo.gif

Set Shell bash
Set FontSize 16
Set Width 1200
Set Height 720
Set Padding 20

# Point pumbaa at the showcase Cromwell without showing the export in the GIF.
Hide
Type "export CROMWELL_HOST=http://localhost:8010"
Enter
Type "clear"
Enter
Show

Type "pumbaa dashboard"
Sleep 800ms
Enter
Sleep 3.5s

# Move the selection down to a two-task workflow.
Down@600ms 1
Sleep 1.2s

# Open the debug view: its call tree, statuses and timings.
Enter
Sleep 3.5s
Down@500ms 2
Sleep 1.5s

# Back to the list, then quit.
Escape
Sleep 1.2s
Down@500ms 2
Sleep 1.2s
Escape
Sleep 800ms
