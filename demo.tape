Output demo.gif

Set Shell zsh
Set FontSize 15
Set Width 820
Set Height 380
Set Theme "Dracula"
Set Padding 24
Set TypingSpeed 80ms
Set PlaybackSpeed 1.0
Set CursorBlink false

# Prompt — already in the project dir
Hide
Type "cd ~/AiGameSave && clear"
Enter
Sleep 300ms
Show

# Step 1: save the session
Type "ags save"
Enter
Sleep 3s

# Pause so viewers read the token savings line
Sleep 1200ms

# Step 2: load → clipboard
Type "ags load"
Enter
Sleep 2s

# Step 3: show what was copied
Type "pbpaste"
Enter
Sleep 3500ms

Sleep 1s
