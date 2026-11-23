export JWT_ACCESS_SECRET="easy-chat-dev"

USERA_TOKEN=$(go run ./apps/im/ws/cmd/devtoken \
  -uid 'userA' \
  -secret "$JWT_ACCESS_SECRET")

USERB_TOKEN=$(go run ./apps/im/ws/cmd/devtoken \
  -uid 'userB' \
  -secret "$JWT_ACCESS_SECRET")

USERC_TOKEN=$(go run ./apps/im/ws/cmd/devtoken \
  -uid 'userC' \
  -secret "$JWT_ACCESS_SECRET")

echo $USERA_TOKEN
echo $USERB_TOKEN
echo $USERC_TOKEN


TOM_TOKEN=$(go run ./apps/im/ws/cmd/devtoken \
  -uid 'user-tom' \
  -secret "$JWT_ACCESS_SECRET")



go run ./apps/task/mq/cmd/publishcmd \
  -conversation 'dc_0a8de2f604c0715b6e39837fcef4d656' \
  -sender 'userA' \
  -client-message 'e2e-message-1' \
  -content '第一条端到端消息'


curl -sS \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlYXN5LWNoYXQtdWlkIjoidXNlckEiLCJleHAiOjE3OTI5Mjg0NTgsImlhdCI6MTc5MjkyNDg1OH0.yhYCZmsUzgAR452wTXdv6RPv2PTt1-lq59ePMxsG0Ik" \
  'http://127.0.0.1:8890/v1/im/messages?conversation_id=dc_0a8de2f604c0715b6e39837fcef4d656&after_seq=0&limit=20'

cd apps/im/rpc/
goctl rpc protoc im.proto \
  --go_out=. \
  --go-grpc_out=. \
  --zrpc_out=.
cd ../../..
