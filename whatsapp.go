package main

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func sendNewsletterMessage(client *whatsmeow.Client, number string, msg string) error {
	jid, _ := types.ParseJID(number)

	msgWhatsApp := &waE2E.Message{
		Conversation: proto.String(msg),
	}

	_, err := client.SendMessage(context.Background(), jid, msgWhatsApp)
	return err
}

func sendImage(client *whatsmeow.Client, number string, imageURL string, caption string) error {
	jid, _ := types.ParseJID(number)

	resp, err := http.Get(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read image data: %w", err)
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	uploadResp, err := client.UploadNewsletter(context.Background(), imageData, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload image: %w", err)
	}

	msgWhatsApp := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imageData))),
			Mimetype:      proto.String(mimeType),
			Caption:       proto.String(caption),
		},
	}

	_, err = client.SendMessage(context.Background(), jid, msgWhatsApp)
	return err
}
