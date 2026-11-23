package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EnsureDirectConversation 是幂等的。即使对同一对用户并发调用，也必须最终只产生一个会话和两条成员记录。
func (s *MongoStore) EnsureDirectConversation(
	ctx context.Context,
	userA string,
	userB string,
) (model.Conversation, error) {
	conversationID, directKey, err := directConversationIdentity(userA, userB)
	if err != nil {
		return model.Conversation{}, err
	}

	session, err := s.client.StartSession()
	if err != nil {
		return model.Conversation{}, fmt.Errorf("start mongo session: %w", err)
	}
	defer session.EndSession(ctx)

	var conversation model.Conversation

	_, err = session.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		now := time.Now().UTC()

		conversationUpdate := bson.M{
			"$setOnInsert": bson.M{
				"directKey": directKey,
				"chatType":  model.ChatTypeSingle,
				"lastSeq":   int64(0),
				"createdAt": now,
			},
			"$set": bson.M{
				"updatedAt": now,
			},
		}

		err := s.convs.FindOneAndUpdate(
			ctx,
			bson.M{"_id": conversationID},
			conversationUpdate,
			options.FindOneAndUpdate().
				SetUpsert(true).
				SetReturnDocument(options.After),
		).Decode(&conversation)
		if err != nil {
			return nil, fmt.Errorf("upsert direct conversation: %w", err)
		}

		for _, userID := range []string{userA, userB} {
			memberUpdate := bson.M{
				"$setOnInsert": bson.M{
					"_id":            bson.NewObjectID(),
					"conversationId": conversationID,
					"userId":         userID,
					"joinedSeq":      int64(0),
					"leftSeq":        int64(0),
					"deliveredSeq":   int64(0),
					"readSeq":        int64(0),
					"isVisible":      true,
					"createdAt":      now,
					"lastActivityAt": now,
				},
				"$set": bson.M{
					"updatedAt": now,
				},
			}

			_, err := s.convMember.UpdateOne(
				ctx, bson.M{
					"conversationId": conversationID,
					"userId":         userID,
				},
				memberUpdate,
				options.UpdateOne().SetUpsert(true),
			)
			if err != nil {
				return nil, fmt.Errorf("upsert conversation member user=%s: %w", userID, err)
			}
		}

		return nil, nil
	})
	if err != nil {
		return model.Conversation{}, fmt.Errorf("ensure direct conversation: %w", err)
	}

	return conversation, nil
}

// func directConversationIdentity(userA, userB string) (string, string, error) {
// 	userA = strings.TrimSpace(userA)
// 	userB = strings.TrimSpace(userB)

// 	if userA == "" || userB == "" {
// 		return "", "", fmt.Errorf("both user ids are required")
// 	}
// 	if userA == userB {
// 		return "", "", fmt.Errorf("a direct conversation requires two different users")
// 	}

// 	users := []string{userA, userB}
// 	sort.Strings(users)

// 	sum := sha256.Sum256([]byte(users[0] + "\x00" + users[1]))
// 	directKey := hex.EncodeToString(sum[:16])

// 	return "dc_" + directKey, directKey, nil
// }

func directConversationIdentity(userA, userB string) (string, string, error) {
	return model.DirectConversationID(userA, userB)
}
