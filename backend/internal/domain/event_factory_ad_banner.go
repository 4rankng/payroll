package domain

import (
	"context"
)

// NewAdBannerCreatedEvent creates an AdBannerCreatedEvent.
func NewAdBannerCreatedEvent(ctx context.Context, banner *AdBanner, actorUserID uint, actorFullName string) AdBannerCreatedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionCreate,
		EntityTypeAdBanner,
		actorFullName,
		banner.Title,
	)

	return AdBannerCreatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "AdBannerCreated", banner.ID, actorUserID, AuditActionCreate, EntityTypeAdBanner, auditMessage),
		Title:     banner.Title,
		StartsAt:  banner.StartsAt,
		EndsAt:    banner.EndsAt,
	}
}

// NewAdBannerUpdatedEvent creates an AdBannerUpdatedEvent. The banner's
// updated_at bump doubles as the campaign version: the portal re-shows the
// sheet once per version.
func NewAdBannerUpdatedEvent(ctx context.Context, banner *AdBanner, actorUserID uint, actorFullName string) AdBannerUpdatedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionUpdate,
		EntityTypeAdBanner,
		actorFullName,
		banner.Title,
	)

	return AdBannerUpdatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "AdBannerUpdated", banner.ID, actorUserID, AuditActionUpdate, EntityTypeAdBanner, auditMessage),
		Title:     banner.Title,
		StartsAt:  banner.StartsAt,
		EndsAt:    banner.EndsAt,
	}
}

// NewAdBannerDeletedEvent creates an AdBannerDeletedEvent.
func NewAdBannerDeletedEvent(ctx context.Context, banner *AdBanner, actorUserID uint, actorFullName string) AdBannerDeletedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionDelete,
		EntityTypeAdBanner,
		actorFullName,
		banner.Title,
	)

	return AdBannerDeletedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "AdBannerDeleted", banner.ID, actorUserID, AuditActionDelete, EntityTypeAdBanner, auditMessage),
		Title:     banner.Title,
	}
}
