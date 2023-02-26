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
* You can download the thumbnail image of the YouTube video from `Menu -> Others -> Download Thumbnail`.

![windows_youtube](https://user-images.githubusercontent.com/6222645/221404955-4fb87e03-873d-49e3-88e9-51c4bb88982b.png)
![windows_youtube_download](https://user-images.githubusercontent.com/6222645/221405594-28aae628-66e6-40c5-a8c5-04fee49c1a64.gif)


## Supported Platforms
### On Windows 11
![windows](https://user-images.githubusercontent.com/6222645/221404834-9dc50dfb-f03a-447f-992a-d5a0898b40ab.png)

### On MacOS (M1)

