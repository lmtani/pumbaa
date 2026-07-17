# Terminal demo for the README, recorded with vhs (https://github.com/charmbracelet/vhs).
#
# Prerequisites:
#   - `vhs` installed and `pumbaa` on PATH
#   - a reachable Cromwell server with a few workflows (CROMWELL_HOST)
#
# Record from the repository root:
#   vhs docs/assets/demo.tape
#
# Then embed docs/assets/demo.gif in the README (there is a TODO marker).

Output docs/assets/demo.gif

Set FontSize 16
Set Width 1200
Set Height 700
Set Padding 20

Type "pumbaa dashboard"
Sleep 500ms
Enter
Sleep 4s

# Walk the workflow list
Down@400ms 3
Sleep 1s

# Open the selected workflow
Enter
Sleep 4s

# Back to the list and quit
Escape
Sleep 1s
Type "q"
Sleep 1s
