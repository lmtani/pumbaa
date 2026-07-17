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

# Select the FASTQ→uBAM workflow (its task has a real execution log) and open
# its debug view.
Down@550ms 4
Sleep 1.2s
Enter
Sleep 3s

# Drill into the task, then show its command and its execution log.
Down@600ms 1
Sleep 1.2s
Type "3"
Sleep 3.5s
Escape
Sleep 800ms
Type "4"
Sleep 1.2s
Enter
Sleep 4s
Escape
Sleep 1s

# Back to the list, then quit.
Escape
Sleep 1s
Escape
Sleep 600ms
