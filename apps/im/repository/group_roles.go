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

func (s *MongoStore) ApplyGroupRoleChanged(
	ctx context.Context,
	groupID string,
	userID string,
	role int32,
	version uint64,
) error {
	if role != int32(model.GroupRoleAdmin) && role != int32(model.GroupRoleMember) {
		return fmt.Errorf("invalid group role %d", role)
	}

	session, err := s.client.StartSession()
	if err != nil {
		return err
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
				bson.M{"$set": bson.M{"updatedAt": now}},
				options.FindOneAndUpdate().SetReturnDocument(options.After),
			).Decode(&conversation)
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrConversationNotFound
			}
			if err != nil {
				return nil, err
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
				return nil, err
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
						bson.M{"membershipVersion": bson.M{"$exists": false}},
						bson.M{"membershipVersion": bson.M{"$lt": version}},
					},
				},
				bson.M{
					"$set": bson.M{
						"groupRole":         role,
						"membershipVersion": version,
						"updatedAt":         now,
					},
				},
			)
			if err != nil {
				return nil, err
			}
			if result.MatchedCount == 0 {
				return nil, err
			}

			return nil, nil
		},
	)

	return err
}

func (s *MongoStore) ApplyGroupOwnerTransferred(
	ctx context.Context,
	groupID string,
	PreviousOwnerID string,
	newOwnerID string,
	version uint64,
) error {
	session, err := s.client.StartSession()
	if err != nil {
		return err
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
				bson.M{"$set": bson.M{
					"updatedAt": now,
				}},
				options.FindOneAndUpdate().SetReturnDocument(options.After),
			).Decode(&conversation)
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrConversationNotFound
			}
			if err != nil {
				return nil, err
			}

			updates := []struct {
				userID string
				role   int32
			}{
				{PreviousOwnerID, int32(model.GroupRoleMember)},
				{newOwnerID, int32(model.GroupRoleOwner)},
			}

			for _, update := range updates {
				result, err := s.convMember.UpdateOne(
					ctx,
					bson.M{
						"conversationId": groupID,
						"userId":         update.userID,
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
							"groupRole":         update.role,
							"membershipVersion": version,
							"updatedAt":         now,
						},
					},
				)
				if err != nil {
					return nil, err
				}
				if result.MatchedCount == 0 {
					return nil, ErrMembershipMemberNotFound
				}
			}
			return nil, nil
		},
	)
	return err
}


