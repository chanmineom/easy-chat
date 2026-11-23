package repository

import "errors"

var (
	ErrCannotFriendSelf       = errors.New("cannot add yourself as a friend")
	ErrFriendshipNotFound     = errors.New("friendship not found")
	ErrAlreadyFriends         = errors.New("users are already friends")
	ErrFriendRequestExists    = errors.New("a pending friend request already exists")
	ErrFriendRequestNotFound  = errors.New("friend request not found")
	ErrFriendRequestHandled   = errors.New("friend request is no longer pending")
	ErrFriendRequestForbidden = errors.New("user cannot handle this friend request")
)
