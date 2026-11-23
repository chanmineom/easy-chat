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

func (s *MongoStore) ApplyGroupDissolved(
	ctx context.Context,
	groupID string,
	version uint64,
) error {
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
				return nil, fmt.Errorf(
					"lock group conversation: %w",
					err,
				)
			}

			if conversation.MembershipVersion >= version {
				return nil, nil
			}

			_, err = s.convs.UpdateOne(
				ctx,
				bson.M{
					"_id": groupID,
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
						"dissolved":         true,
						"membershipVersion": version,
						"updatedAt":         now,
					},
				},
			)
			if err != nil {
				return nil, fmt.Errorf(
					"dissolve conversation: %w",
					err,
				)
			}

			_, err = s.convMember.UpdateMany(
				ctx,
				bson.M{
					"conversationId": groupID,
					"isVisible":      true,
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
						"isVisible":         false,
						"leftSeq":           conversation.LastSeq,
						"membershipVersion": version,
						"updatedAt":         now,
					},
				},
			)
			if err != nil {
				return nil, fmt.Errorf(
					"hide dissolved conversation members: %w",
					err,
				)
			}

			return nil, nil
		},
	)
	if err != nil {
		return err
	}

	return nil
}
