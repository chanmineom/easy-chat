package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrMembershipVersionStale   = errors.New("stale membership version")
	ErrMembershipMemberNotFound = errors.New("membership member not found")
)

func (s *MongoStore) ApplyGroupMemberLeft(ctx context.Context, groupID string, userID string, version uint64) error {
	session, err := s.client.StartSession()
	if err != nil {
		return fmt.Errorf("start mongo session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(
		ctx,
		func(ctx context.Context) (any, error) {
			now := time.Now().UTC()

			var conversation model.Conversation
			err := s.convs.FindOneAndUpdate(
				ctx,
				bson.M{"_id": groupID},
				bson.M{
					"$set": bson.M{"updatedAt": now},
				},
				options.FindOneAndUpdate().SetReturnDocument(options.After),
			).Decode(&conversation)
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrConversationNotFound
			}
			if err != nil {
				return nil, fmt.Errorf(
					"lock conversation: %w",
					err,
				)
			}

			if conversation.Dissolved || conversation.MembershipVersion >= version {
				return nil, nil
			}

			var member model.ConversationMember
			err = s.convMember.FindOne(
				ctx,
				bson.M{
					"conversationId": groupID,
					"userId":         userID,
				},
			).Decode(&member)
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrMembershipMemberNotFound
			}
			if err != nil {
				return nil, fmt.Errorf(
					"find membership member: %w",
					err,
				)
			}

			if member.MembershipVersion >= version {
				return nil, nil
			}

			result, err := s.convMember.UpdateOne(
				ctx,
				bson.M{
					"conversationId": groupID,
					"userId":         userID,
					"$or": bson.A{
						bson.M{
							"membershipVersion": bson.M{"$exists": false},
						},
						bson.M{
							"membershipVersion": bson.M{"$lt": version},
						},
					},
				},
				bson.M{
					"$set": bson.M{
						"isVisible":         false,
						"leftSeq":           conversation.LastSeq,
						"membershipVersion": version,
						"updatedAt":         now,
					},
				},
			)
			if err != nil {
				return nil, fmt.Errorf(
					"update membership member: %w",
					err,
				)
			}
			if result.MatchedCount == 0 {
				return nil, nil
			}

			return nil, nil
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *MongoStore) ApplyGroupMembersAdded(ctx context.Context, groupID string, userIDs []string, version uint64) error {
	session, err := s.client.StartSession()
	if err != nil {
		return fmt.Errorf("start mongo session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(
		ctx,
		func(ctx context.Context) (any, error) {
			now := time.Now().UTC()

			var conversation model.Conversation
			err := s.convs.FindOneAndUpdate(
				ctx,
				bson.M{
					"_id": groupID,
				},
				bson.M{
					"$set": bson.M{
						"updatedAt": now,
					},
				},
				options.FindOneAndUpdate().SetReturnDocument(options.After),
			).Decode(&conversation)
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrConversationNotFound
			}
			if err != nil {
				return nil, err
			}

			if conversation.Dissolved {
				return nil, nil
			}

			for _, userID := range userIDs {
				var member model.ConversationMember
				err := s.convMember.FindOne(
					ctx,
					bson.M{
						"conversationId": groupID,
						"userId":         userID,
					},
				).Decode(&member)

				if errors.Is(err, mongo.ErrNoDocuments) {
					_, err := s.convMember.InsertOne(
						ctx,
						model.ConversationMember{
							ID:                bson.NewObjectID(),
							ConversationID:    groupID,
							JoinedSeq:         conversation.LastSeq,
							DeliveredSeq:      conversation.LastSeq,
							ReadSeq:           conversation.LastSeq,
							IsVisible:         true,
							MembershipVersion: version,
							CreatedAt:         now,
							UpdatedAt:         now,
							LastActivityAt:    now,
						},
					)
					if err != nil {
						return nil, err
					}
					continue
				}
				if err != nil {
					return nil, err
				}

				if member.MembershipVersion >= version {
					continue
				}

				_, err = s.convMember.UpdateOne(
					ctx,
					bson.M{
						"conversationId": groupID,
						"userId":         userID,
						"$or": bson.A{
							bson.M{
								"membershipVersion": bson.M{
									"$exists": false,
								},
							},
							bson.M{
								"membershipVersion": bson.M{
									"$lt": version,
								},
							},
						},
					},
					bson.M{
						"$set": bson.M{
							"isVisible":         true,
							"joinedSeq":         conversation.LastSeq,
							"leftSeq":           int64(0),
							"deliveredSeq":      conversation.LastSeq,
							"readSeq":           conversation.LastSeq,
							"membershipVersion": version,
							"lastActivityAt":    now,
							"updatedAt":         now,
						},
					},
				)
				if err != nil {
					return nil, err
				}
			}
			return nil, nil
		},
	)
	if err != nil {
		return err
	}

	return nil
}
