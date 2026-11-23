package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const MaxGroupMembers = 500

var (
	ErrInvalidGroupConversation = errors.New("invalid group conversation")
	ErrConversationTypeConflict = errors.New("conversation type conflict")
)

type EnsureGroupConversationInput struct {
	GroupID           string
	CreatorID         string
	MemberIDs         []string
	MembershipVersion uint64
}

func (s *MongoStore) EnsureGroupConversation(ctx context.Context, input EnsureGroupConversationInput) (model.Conversation, error) {
	memberIDs, err := normalizeGroupMembers(input.CreatorID, input.MemberIDs)
	if err != nil {
		return model.Conversation{}, err
	}

	groupID := strings.TrimSpace(input.GroupID)
	if groupID == "" {
		return model.Conversation{}, fmt.Errorf("%w: group id is required", ErrInvalidGroupConversation)
	}

	session, err := s.client.StartSession()
	if err != nil {
		return model.Conversation{}, fmt.Errorf("start mongo session: %w", err)
	}
	defer session.EndSession(ctx)

	var conversation model.Conversation

	_, err = session.WithTransaction(
		ctx,
		func(ctx context.Context) (any, error) {
			now := time.Now().UTC()

			update := bson.M{
				"$setOnInsert": bson.M{
					"chatType":  model.ChatTypeGroup,
					"lastSeq":   int64(0),
					"createdAt": now,
					"dissolved":         false,
					"membershipVersion": input.MembershipVersion,
				},
				"$set": bson.M{
					"updatedAt": now,
				},
			}

			err := s.convs.FindOneAndUpdate(
				ctx,
				bson.M{"_id": groupID},
				update,
				options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
			).Decode(&conversation)
			if err != nil {
				return nil, fmt.Errorf(
					"upsert group conversation: %w",
					err,
				)
			}

			if conversation.ChatType != model.ChatTypeGroup {
				return nil, ErrConversationTypeConflict
			}

			for _, userID := range memberIDs {
				role := int32(model.GroupRoleMember)
				if userID == input.CreatorID {
					role = int32(model.GroupRoleAdmin)
				}

				memberUpdate := bson.M{
					"$setOnInsert": bson.M{
						"_id":            bson.NewObjectID(),
						"conversationId": groupID,
						"userId":         userID,
						"joinedSeq":      conversation.LastSeq,
						"leftSeq":        int64(0),
						"deliveredSeq":   int64(0),
						"readSeq":        int64(0),
						"isVisible":      true,
						"createdAt":      now,
						"lastActivityAt": now,
						"membershipVersion": input.MembershipVersion,
						"groupRole":      role,
					},
					"$set": bson.M{
						"updatedAt": now,
					},
				}

				_, err := s.convMember.UpdateOne(
					ctx,
					bson.M{
						"conversationId": groupID,
						"userId":         userID,
					},
					memberUpdate,
					options.UpdateOne().SetUpsert(true),
				)
				if err != nil {
					return nil, fmt.Errorf(
						"upsert group member %s: %w",
						userID,
						err,
					)
				}
			}

			return nil, nil
		},
	)
	if err != nil {
		if errors.Is(err, ErrConversationTypeConflict) {
			return model.Conversation{}, err
		}
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Conversation{}, err
		}
		return model.Conversation{}, fmt.Errorf(
			"ensure group conversation: %w",
			err,
		)
	}

	return conversation, nil
}

func normalizeGroupMembers(creatorID string, memberIDs []string) ([]string, error) {
	creatorID = strings.TrimSpace(creatorID)
	if creatorID == "" {
		return nil, fmt.Errorf("%w: creator is required", ErrInvalidGroupConversation)
	}

	unique := make(map[string]struct{}, len(memberIDs)+1)
	unique[creatorID] = struct{}{}

	for _, memberID := range memberIDs {
		memberID = strings.TrimSpace(memberID)
		if memberID == "" {
			return nil, fmt.Errorf("%w: member id is empty", ErrInvalidGroupConversation)
		}
		unique[memberID] = struct{}{}
	}

	if len(unique) > MaxGroupMembers {
		return nil, fmt.Errorf(
			"%w: member count exceeds %d",
			ErrInvalidGroupConversation,
			MaxGroupMembers,
		)
	}

	result := make([]string, 0, len(unique))
	for userID := range unique {
		result = append(result, userID)
	}
	sort.Strings(result)

	return result, nil
}
