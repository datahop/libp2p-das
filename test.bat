@echo off

if "%~1"=="" (
    echo Please enter at least 1 parameter. The first parameter is a number and the second parameter is optional and can be 'debug'.
    exit /b
)

if "%~2"=="debug" (
    echo Starting libp2p-das with %~1 peers in debug mode.
) else (
    echo Starting libp2p-das with %~1 peers.
)

start "" go run . -debug -seed 1234 -port 61960

timeout /t 2

for /l %%i in (1,1,%~1) do (
    if "%~2"=="debug" (
        start "" go run . -debug -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73
        timeout /t 1
    ) else (
        start "" go run . -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73
        timeout /t 1
    )
)

timeout /t 20