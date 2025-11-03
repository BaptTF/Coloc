package main

import (
	"context"
	"log"

	"github.com/lrstanley/go-ytdlp"
)

func main() {
	log.Println("Installing yt-dlp and ffmpeg...")
	ytdlp.MustInstall(context.TODO(), nil)
	ytdlp.MustInstallFFmpeg(context.TODO(), nil)
	ytdlp.MustInstallFFprobe(context.TODO(), nil)
	log.Println("Tools installed successfully")
}
