package usecase_test

import (
	"context"
	"testing"

	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/internal/usecase"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Repositories
type MockCandidateRepo struct {
	mock.Mock
}

func (m *MockCandidateRepo) GetByUserID(ctx context.Context, userID string) (*domain.CandidateProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CandidateProfile), args.Error(1)
}

func (m *MockCandidateRepo) Create(ctx context.Context, profile *domain.CandidateProfile) error {
	return m.Called(ctx, profile).Error(0)
}

func (m *MockCandidateRepo) Update(ctx context.Context, profile *domain.CandidateProfile) error {
	return m.Called(ctx, profile).Error(0)
}

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) Create(ctx context.Context, user *domain.User) error {
	return m.Called(ctx, user).Error(0)
}
func (m *MockUserRepo) Update(ctx context.Context, user *domain.User) error {
	return m.Called(ctx, user).Error(0)
}
func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func TestCandidateIDOR(t *testing.T) {
	mockRepo := new(MockCandidateRepo)
	validate := validator.New()
	uc := usecase.NewCandidateUsecase(mockRepo, nil, validate)

	t.Run("Should fail when Context UserID does not match Argument UserID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), domain.KeyUserID, "user1")
		_, err := uc.GetProfile(ctx, "user2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only view your own profile")
	})

	t.Run("Should fail safely when Context UserID is nil", func(t *testing.T) {
		ctx := context.Background() // keys missing
		_, err := uc.GetProfile(ctx, "user1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "User not authenticated")
	})
}

func TestAuthPrivilege(t *testing.T) {
	mockRepo := new(MockUserRepo)
	uc := usecase.NewAuthUsecase(mockRepo)

	t.Run("Should fail if role is not admin", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), domain.KeyUserRole, "candidate")
		err := uc.AssignRole(ctx, "target_user", "admin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Only admins can assign roles")
	})

	t.Run("Should fail safe if role is nil", func(t *testing.T) {
		ctx := context.Background()
		err := uc.AssignRole(ctx, "target_user", "admin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Only admins can assign roles")
	})
}

func TestCandidateUpdateValidation(t *testing.T) {
	mockRepo := new(MockCandidateRepo)
	validate := validator.New()
	uc := usecase.NewCandidateUsecase(mockRepo, nil, validate)

	t.Run("Should fail if required fields are missing", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), domain.KeyUserID, "user1")
		profile := &domain.CandidateProfile{
			// Missing Title, Skills, etc.
		}
		err := uc.UpdateProfile(ctx, profile)
		assert.Error(t, err)
		// Expect validation error
	})

	t.Run("Should force UserID from context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), domain.KeyUserID, "user1")
		profile := &domain.CandidateProfile{
			UserID: "hacker_try",
			Title:  "Valid Title",
			Skills: []string{"Go"},
		}

		// We mock Update to check what actual profile is passed
		mockRepo.On("Update", ctx, mock.AnythingOfType("*domain.CandidateProfile")).Return(nil).Run(func(args mock.Arguments) {
			p := args.Get(1).(*domain.CandidateProfile)
			assert.Equal(t, "user1", p.UserID)
		})

		_ = uc.UpdateProfile(ctx, profile)
	})
}
