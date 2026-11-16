package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"

	"github.com/mckelvj/SendEmailFunction/internal/email"
)

type EmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

var sender *email.Sender

func response(status int, message string) events.APIGatewayV2HTTPResponse {
	body, _ := json.Marshal(map[string]string{
		"message": message,
	})
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}
}

func handler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (
	events.APIGatewayV2HTTPResponse, error,
) {
	if request.RequestContext.HTTP.Method != "POST" {
		return response(405, "Method not allowed"), nil
	}

	// Require a verified JWT authorizer on the API route.
	if request.RequestContext.Authorizer == nil ||
		request.RequestContext.Authorizer.JWT == nil {
		return response(403, "Forbidden"), nil
	}

	var input EmailRequest
	if err := json.Unmarshal([]byte(request.Body), &input); err != nil {
		return response(400, "Invalid JSON"), nil
	}

	input.To = strings.TrimSpace(input.To)
	input.Subject = strings.TrimSpace(input.Subject)

	// Never allow the browser to choose arbitrary recipients.
	if input.To != os.Getenv("ALLOWED_RECIPIENT") {
		return response(403, "Recipient not permitted"), nil
	}

	if len(input.Subject) > 200 || len(input.Body) > 10000 {
		return response(400, "Message too large"), nil
	}

	_, err := sender.Send(ctx, input.To, input.Subject, input.Body)
	if err != nil {
		log.Printf("Email send failed: %v", err)
		return response(500, "Unable to send email"), nil
	}

	return response(200, "Email accepted by SES"), nil
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("AWS configuration failed: %v", err)
	}

	from := os.Getenv("SES_FROM_EMAIL")
	if from == "" || os.Getenv("ALLOWED_RECIPIENT") == "" {
		log.Fatal("Required email environment variables missing")
	}

	sender = &email.Sender{
		Client: sesv2.NewFromConfig(cfg),
		From:   from,
	}

	lambda.Start(handler)
}
