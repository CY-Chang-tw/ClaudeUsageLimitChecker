@echo off
cd /d "%~dp0"
start /B ClaudeUsageLimitChecker.exe >> monitor.log 2>&1
