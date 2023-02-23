## About
The program opens the TCP server on the PC, and each client communicates 1:N with the PC, dividing the requested file by the number of clients and downloading it.

If 4 clients download a 1GB file, divide them into 250MB each and transfer them to the PC.

## GUI
- [fyne-io/fyne](https://github.com/fyne-io/fyne)

## Installation
You must run [yms2772/download_accelerator_agent](https://github.com/yms2772/download_accelerator_agent), before run this program.
```bash
go install github.com/yms2772/download_accelerator@latest
```
### or
```bash
git clone https://github.com/yms2772/download_accelerator.git
cd download_accelerator
go build -v
```
The result is saved in the `downloaded` folder.

## Options
|      Name      | Description                                                                |
|:--------------:|:---------------------------------------------------------------------------|
|      Port      | Port to open TCP socket (port forwarding is required if using a public IP) |
|      URL       | URL to download                                                            |
|    Filename    | Filled in automatically when entering URL                                  |
|    Parallel    | Number of downloads per client at the same time                            |
|   Chunk Size   | Size to split when sending a file from client to PC                        |
| Chunk Parallel | Number of chunks sent at the same time                                     |

## YouTube
#### Supported URLs: `youtube.com`, `youtu.be`, `shorts`
* Set the `Parallel` option to more than `50` when you download any YouTube video. Because the YouTube server is super slow.
* The `Audio Included` option is enabled when the `ffmpeg` is in `PATH` or `./bin`. (default has no audio)

![windows_youtube](https://user-images.githubusercontent.com/6222645/220840488-02d62d9d-a7ef-455b-9d23-b321f53fa723.png)


## Supported Platforms
### On Windows 11
![windows](https://user-images.githubusercontent.com/6222645/219873230-d1bed6e8-6144-4948-8027-72a1160cd299.png)

### On MacOS (M1)
![macos](https://user-images.githubusercontent.com/6222645/219873286-0fd4d8bd-a1e4-41f0-8c16-045288d6d76f.png)

