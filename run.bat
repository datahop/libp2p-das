@echo off

set duration=%1
set nodeType=%2

if "%duration%"=="" (
    echo There should be 2 parameters: duration and nodeType. e.g. run.bat 30 builder
    exit /b
) else if "%nodeType%"=="" (
    echo There should be 2 parameters: duration and nodeType. e.g. run.bat 30 builder
    exit /b
)

if not "%nodeType%"=="builder" (
    if not "%nodeType%"=="validator" (
        if not "%nodeType%"=="nonvalidator" (
            echo Invalid nodeType. Valid options are "builder", "validator", or "nonvalidator".
            exit /b
        )
    )
)

if "%nodeType%"=="builder" (
    go run . -debug -seed 1234 -port 61960 -nodeType builder -duration %duration%
    exit /b
) else (
    go run . -debug -duration %duration% -nodeType %nodeType% -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73
    exit /b
)