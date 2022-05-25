package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

const input = "rtmp://localhost/live/test"

func main() {
	apiKey := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if len(apiKey) == 0 {
		log.Fatal("No GOOGLE_APPLICATION_CREDENTIALS env variable.")
	}
	err := runFF()
	if err != nil {
		log.Fatal(err)
	}
}

func runFF() error {
	cmd := exec.Command(
		"/bin/ffmpeg",
		"-i", input,
		"-vn",
		"-c:a", "libopus",
		"-b:a", "64k",
		"-ac", "1", // mono
		"-ar", "16k", // sample rate
		"-f", "ogg",
		"-",
	)

	log.Println("Run command:", cmd)

	out, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	_, err = cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	// Send the contents of the audio file with the encoding and
	// and sample rate information to be transcripted.
	stream, err := client.StreamingRecognize(ctx)
	if err != nil {
		log.Fatal(err)
	}
	stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				SingleUtterance: false,
				InterimResults:  true, // We need this to have GCP correct and send the detected text continuously.
				Config: &speechpb.RecognitionConfig{
					Encoding:        speechpb.RecognitionConfig_OGG_OPUS,
					SampleRateHertz: 16000,
					LanguageCode:    "fr-FR",
					UseEnhanced:     true,      // Better, but costs more $$$
					Model:           "default", // Switch to "video" when fr-FR is supported
				},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Let's begin")

	go func() {
		log.Println("Results:")
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				log.Println("Recv EOF")
				return
			}
			if err != nil {
				log.Println("GCP Error:", err)
				time.Sleep(1 * time.Second)
				continue
			}
			log.Printf("Waiting for results....")
			for _, result := range resp.Results {
				log.Printf("Result: %+v\n", result)
			}
		}
	}()

	cmd.Start()
	for {
		buf := make([]byte, 1024)
		n, err := out.Read(buf)
		if n == 0 {
			log.Println("Too soon: sleep 100ms")
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if err == io.EOF {
			log.Println("EOF!")
			stream.CloseSend()
			return nil
		}
		if err != nil {
			log.Println(err)
			time.Sleep(1 * time.Second)
			continue
		}
		err = stream.Send(&speechpb.StreamingRecognizeRequest{
			StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
				AudioContent: buf[:n],
			},
		})
		if err != nil {
			log.Println("Send error:", err)
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}
