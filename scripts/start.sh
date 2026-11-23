go run ./apps/im/rpc \
  -f apps/im/rpc/etc/im.yaml

go run ./apps/social/rpc \
  -f apps/social/rpc/etc/social.yaml

go run ./apps/im/ws \
  -f apps/im/ws/etc/im.yaml

go run ./apps/im/ws \
  -f apps/im/ws/etc/im-node2.yaml

go run ./apps/task/mq \
  -f apps/task/mq/etc/task.yaml

go run ./apps/user/rpc \
  -f apps/user/rpc/etc/user.yaml

go run ./apps/im/api \
  -f apps/im/api/etc/im.yaml
