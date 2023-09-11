@echo off

set duration=%1
set nodeType=%2
set parcelSize=%3

if "%duration%"=="" (
    echo There should be 3 parameters: duration, nodeType, and parcelSize. e.g. run.bat 30 builder 256
    exit /b
) else if "%nodeType%"=="" (
    echo There should be 3 parameters: duration, nodeType, and parcelSize. e.g. run.bat 30 builder 256
    exit /b
)else if "%parcelSize%"=="" (
    echo There should be 3 parameters: duration, nodeType, and parcelSize. e.g. run.bat 30 builder 256
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
    go run . -debug -seed 1234 -port 61960 -nodeType builder -duration %duration% -parcelSize %parcelSize%
    exit /b
) else (
    go run . -debug -duration %duration% -nodeType %nodeType% -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 -parcelSize %parcelSize%
    exit /b
)