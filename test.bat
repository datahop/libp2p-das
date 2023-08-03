@echo off

del *.csv

set duration=%1
set builderCount=%2
set validatorCount=%3
set nonValidatorCount=%4

if "%duration%"=="" (
    echo There should be 4 parameters: duration, builderCount, validatorCount, and nonValidatorCount. e.g. test.bat 30 1 2 1
    exit /b
) else if "%builderCount%"=="" (
    echo There should be 4 parameters: duration, builderCount, validatorCount, and nonValidatorCount. e.g. test.bat 30 1 2 1
    exit /b
) else if "%validatorCount%"=="" (
    echo There should be 4 parameters: duration, builderCount, validatorCount, and nonValidatorCount. e.g. test.bat 30 1 2 1
    exit /b
) else if "%nonValidatorCount%"=="" (
    echo There should be 4 parameters: duration, builderCount, validatorCount, and nonValidatorCount. e.g. test.bat 30 1 2 1
    exit /b
)

set /a builderCount=%builderCount%
if %builderCount% leq 0 (
    echo builderCount should be greater than 0.
    exit /b
)

set /a nonBuilderCount=%validatorCount%+%nonValidatorCount%

if %nonBuilderCount% leq 0 (
    echo The sum of validatorCount and nonValidatorCount should be greater than 0.
    exit /b
)

for /L %%i in (1,1,%builderCount%) do (
    start /B "" run.bat %duration% builder
)

for /L %%i in (1,1,%validatorCount%) do (
    start /B "" run.bat %duration% validator
)

for /L %%i in (1,1,%nonValidatorCount%) do (
    start /B "" run.bat %duration% nonvalidator
)