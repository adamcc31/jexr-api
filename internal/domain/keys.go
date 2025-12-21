package domain

type CtxKey string

const (
	KeyUserID    CtxKey = "UserID"
	KeyUserEmail CtxKey = "Email"
	KeyUserRole  CtxKey = "Role"
)
