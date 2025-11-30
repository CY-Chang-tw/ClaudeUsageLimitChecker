Set WshShell = CreateObject("WScript.Shell")
WshShell.Run chr(34) & WScript.ScriptFullName & "\..\ClaudeUsageLimitChecker.exe" & Chr(34), 0
Set WshShell = Nothing
