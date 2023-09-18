@echo off

set nodeType=%1
set parcelSize=%2

if "%nodeType%"=="" (
    echo There should be 2 parameters: nodeType, and parcelSize. e.g. run.bat 30 builder 256
    exit /b
)else if "%parcelSize%"=="" (
    echo There should be 2 parameters: nodeType, and parcelSize. e.g. run.bat 30 builder 256
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
    go run . -seed 1234 -port 61960 -nodeType builder -parcelSize %parcelSize%
    exit /b
) else (
    go run . -nodeType %nodeType% -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 -parcelSize %parcelSize%
    exit /b
)