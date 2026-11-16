package email

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type SESClient interface {
	SendEmail(
		ctx context.Context,
		input *sesv2.SendEmailInput,
		opts ...func(*sesv2.Options),
	) (*sesv2.SendEmailOutput, error)
}

type Sender struct {
	Client SESClient
	From   string
}

func (s *Sender) Send(
	ctx context.Context,
	to, subject, body string,
) (string, error) {
	if s.Client == nil || s.From == "" {
		return "", errors.New("email sender not configured")
	}

	address, err := mail.ParseAddress(to)
	if err != nil || address.Address != to {
		return "", errors.New("invalid recipient address")
	}

	if strings.TrimSpace(subject) == "" ||
		strings.TrimSpace(body) == "" {
		return "", errors.New("subject and body are required")
	}

	result, err := s.Client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: &s.From,
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    &subject,
					Charset: awsString("UTF-8"),
				},
				Body: &types.Body{
					Text: &types.Content{
						Data:    &body,
						Charset: awsString("UTF-8"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("SES send failed: %w", err)
	}

	if result.MessageId == nil {
		return "", errors.New("SES returned no message ID")
	}
	return *result.MessageId, nil
}

func awsString(s string) *string {
	return &s
}
