@echo off

del *.csv

set builderCount=%1
set validatorCount=%2
set nonValidatorCount=%3
set parcelSize=%4

if "%builderCount%"=="" (
    echo There should be 4 parameters: builderCount, validatorCount, nonValidatorCount, and parcelSize. e.g. test.bat 1 2 1 512
    exit /b
) else if "%validatorCount%"=="" (
    echo There should be 4 parameters: builderCount, validatorCount, nonValidatorCount, and parcelSize. e.g. test.bat 1 2 1 512
    exit /b
) else if "%nonValidatorCount%"=="" (
    echo There should be 4 parameters: builderCount, validatorCount, nonValidatorCount, and parcelSize. e.g. test.bat 1 2 1 512
    exit /b
) else if "%parcelSize%"=="" (
    echo There should be 4 parameters: builderCount, validatorCount, nonValidatorCount, and parcelSize. e.g. test.bat 1 2 1 512
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
    start /B "" run.bat builder %parcelSize%
)

for /L %%i in (1,1,%validatorCount%) do (
    start /B "" run.bat validator %parcelSize%
)

for /L %%i in (1,1,%nonValidatorCount%) do (
    start /B "" run.bat nonvalidator %parcelSize%
)