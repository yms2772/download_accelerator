## About
The program opens the TCP server on the PC, and each client communicates 1:N with the PC, dividing the requested file by the number of clients and downloading it.

If 4 clients download a 1GB file, divide them into 250MB each and transfer them to the PC.

## GUI
- [fyne-io/fyne](https://github.com/fyne-io/fyne)

## Installation
You must run [yms2772/download_accelerator_agent](https://github.com/yms2772/download_accelerator_agent), before run this program.
```
git clone https://github.com/yms2772/download_accelerator.git
cd download_accelerator
go build -v
```
The result is saved in the `downloaded` folder.

## Options
|Name|Description|
|------|---|
|Port|Port to open TCP socket (port forwarding is required if using a public IP)|
|URL|URL to download|
|Fileanme|Filled in automatically when entering URL|
|Parallel|Number of downloads per client at the same time|
|Chunk Size|Size to split when sending a file from client to PC|
|Chunk Parallel|Number of chunks sent at the same time|

## YouTube
#### Supported URL: `www.youtube.com`, `youtube.com`, `youtu.be`
*Note: Audio is only included in resolutions below `360p`. If you want to download with audio in high resolution, select `audio/mp4` or `audio/webm` in `Quality` and merge with `ffmpeg`.*

![windows_youtube](https://user-images.githubusercontent.com/6222645/220338480-39738426-f40c-4495-9465-2959709ae2a3.png)

## Supported Platforms
### On Windows
![windows](https://user-images.githubusercontent.com/6222645/219873230-d1bed6e8-6144-4948-8027-72a1160cd299.png)

### On MacOS (M1)
![macos](https://user-images.githubusercontent.com/6222645/219873286-0fd4d8bd-a1e4-41f0-8c16-045288d6d76f.png)

