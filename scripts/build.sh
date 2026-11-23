cd ~/easy-chat
docker build -f deploy/docker/user-rpc.Dockerfile \
  -t easy-chat-user-rpc:dev .

docker build -f deploy/docker/user-api.Dockerfile \
  -t easy-chat-user-api:dev .

docker build -f deploy/docker/social-rpc.Dockerfile \
  -t easy-chat-social-rpc:dev .

docker build -f deploy/docker/social-api.Dockerfile \
  -t easy-chat-social-api:dev .

docker build -f deploy/docker/im-rpc.Dockerfile \
  -t easy-chat-im-rpc:dev .

docker build -f deploy/docker/im-api.Dockerfile \
  -t easy-chat-im-api:dev .

docker build -f deploy/docker/im-ws.Dockerfile \
  -t easy-chat-im-ws:dev .

docker build -f deploy/docker/task-mq.Dockerfile \
  -t easy-chat-task-mq:dev .