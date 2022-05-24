package main

import (
	"context"
	"encoding/binary"
	"io"
	"log"
	"os"
	"os/exec"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

const input = "/path/to/input"

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
				Config: &speechpb.RecognitionConfig{
					Encoding:        speechpb.RecognitionConfig_OGG_OPUS,
					SampleRateHertz: 48000,
					LanguageCode:    "fr-FR",
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
				log.Println("EOF")
			}
			for _, result := range resp.Results {
				log.Printf("Result: %+v\n", result)
			}
		}
	}()

	cmd.Start()
	buf := make([]byte, 512)
	for {
		n, err := out.Read(buf)
		if err == io.EOF {
			stream.CloseSend()
		}
		if err != nil {
			break
		}
		binary.Write(os.Stdout, binary.LittleEndian, buf)
		err = stream.Send(&speechpb.StreamingRecognizeRequest{
			StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
				AudioContent: buf[:n],
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}
