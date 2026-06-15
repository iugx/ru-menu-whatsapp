package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func handler(ctx context.Context, raw json.RawMessage) (string, error) {
	log.Printf("RAW EVENT: %s", string(raw))

	var event EventLambda
	if err := json.Unmarshal(raw, &event); err != nil {
		return "", fmt.Errorf("failed to parse event: %w", err)
	}

	bucketName := os.Getenv("S3_BUCKET_NAME")
	dbFileName := os.Getenv("DB_FILE_NAME")
	localPath := "/tmp/" + dbFileName

	dbLog := waLog.Stdout("Database", "DEBUG", true)

	cfgAws, err := config.LoadDefaultConfig(ctx)
	s3Client := NewS3Client(cfgAws)

	if err != nil {
		return "", fmt.Errorf("unable to load SDK config: %w", err)
	}

	fmt.Println("Downloading database from S3...")

	if err := s3Client.downloadFromS3(ctx, bucketName, dbFileName, localPath); err != nil {
		fmt.Printf("Could not download DB (normal for first run): %v\n", err)
	}

	dbString := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&cache=shared",
		localPath,
	)

	container, err := sqlstore.New(ctx, "sqlite", dbString, dbLog)
	if err != nil {
		return "", fmt.Errorf("sqlstore.New: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return "", fmt.Errorf("GetFirstDevice: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "DEBUG", true))

	if client.Store.ID == nil {
		return "", fmt.Errorf("no session found in DB — run QR auth locally first, then upload the DB to S3")
	}

	if err := client.Connect(); err != nil {
		return "", fmt.Errorf("Connect: %w", err)
	}

	formattedMenu := fmtMenu(event)

	if event.ResponsePayload.ImgMenu != nil && *event.ResponsePayload.ImgMenu != "" {
		fmt.Printf("Sending image: %s\n", *event.ResponsePayload.ImgMenu)
		if err := sendImage(client, event.RequestPayload.WhatsAppNumber, *event.ResponsePayload.ImgMenu, formattedMenu); err != nil {
			fmt.Printf("Image send error: %v\n", err)
		}
	} else {
		if err := sendNewsletterMessage(client, event.RequestPayload.WhatsAppNumber, formattedMenu); err != nil {
			fmt.Printf("Send error: %v\n", err)
		}
	}

	client.Disconnect()

	if err := container.Close(); err != nil {
		fmt.Printf("container.Close error: %v\n", err)
	}

	fmt.Println("Uploading updated database to S3...")
	if err := s3Client.uploadToS3(ctx, bucketName, dbFileName, localPath); err != nil {
		return "", fmt.Errorf("upload to S3: %w", err)
	}

	fmt.Println("Success! Session saved.")
	return "Success", nil
}
