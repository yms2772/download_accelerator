## About
The program opens the TCP server on the PC, and each client communicates 1:N with the PC, dividing the requested file by the number of clients and downloading it.

## GUI
- [fyne-io/fyne](https://github.com/fyne-io/fyne)

## Installation
You must run [yms2772/download_accelerator_agent](https://github.com/yms2772/download_accelerator_agent), before run this program.
```
git clone https://github.com/yms2772/download_accelerator.git
cd download_accelerator
go build -v
```

## Options
|Name|Description|
|------|---|
|Port|Port to open TCP socket|
|URL|URL you want to download|
|Fileanme|Filled in automatically when entering URL|
|Parallel|Number of downloads per client at the same time|
|Chunk Size|Size to split when sending a file from client to PC|
|Chunk Parallel|Number of chunks sent at the same time|

## Platform
### On Windows
![windows](https://user-images.githubusercontent.com/6222645/219873230-d1bed6e8-6144-4948-8027-72a1160cd299.png)

### On MacOS (M1)
![macos](https://user-images.githubusercontent.com/6222645/219873286-0fd4d8bd-a1e4-41f0-8c16-045288d6d76f.png)

