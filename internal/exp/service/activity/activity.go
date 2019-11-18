// Package activity provides an activity service definition.
package activity

import (
	"github.com/shurcooL/events"
	notificationv2 "github.com/shurcooL/home/internal/exp/service/notification"
)

// Service defines methods of an activity service.
type Service interface {
	events.Service
	notificationv2.Service
}
