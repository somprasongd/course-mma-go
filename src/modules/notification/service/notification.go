package service

import (
	"fmt"
	"go-mma/shared/common/logger"
)

type NotificationService interface {
	SendEmail(to string, subject string, payload map[string]any) error
}
type notificationService struct {
}

func NewNotificationService() NotificationService {
	return &notificationService{}
}

func (s *notificationService) SendEmail(to string, subject string, payload map[string]any) error {
	// implement email sending logic here
	logger.Log.Info(fmt.Sprintf("Sending email to %s with subject: %s and payload: %v", to, subject, payload))
	return nil
}
