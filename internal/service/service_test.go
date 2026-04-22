package service

import (
	"errors"
	"log/slog"
	"maps"
	"slices"
	"testing"

	"github.com/antongolenev23/tuchka-server/internal/config"
	"github.com/antongolenev23/tuchka-server/internal/entity"
	"github.com/antongolenev23/tuchka-server/internal/mocks"
	"github.com/antongolenev23/tuchka-server/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type storageWant struct {
	errors []error
}

type repositoryWant struct {
	errors []error
}

type got struct {
	files []entity.File
}

type want struct {
	storage      storageWant
	repo         repositoryWant
	removeCalls  []bool
	uploadResult entity.OperationResult
}

func TestUpload(t *testing.T) {
	tests := []struct {
		name string
		got  got
		want want
	}{
		{
			name: "correct files saving",
			got: got{
				files: []entity.File{
					{Name: "file1.txt"},
					{Name: "file2.txt"},
				},
			},
			want: want{
				storage: storageWant{
					errors: []error{nil, nil},
				},
				repo: repositoryWant{
					errors: []error{nil, nil},
				},
				removeCalls: []bool{false, false},
				uploadResult: entity.OperationResult{
					Success: []string{"file1.txt", "file2.txt"},
					Errors:  nil,
				},
			},
		},
		{
			name: "storage and repository errors while file saving",
			got: got{
				files: []entity.File{
					{Name: "file1.txt"},
					{Name: "file2.txt"},
					{Name: "file3.txt"},
				},
			},
			want: want{
				storage: storageWant{
					errors: []error{errors.New("storage error"), nil, nil},
				},
				repo: repositoryWant{
					errors: []error{nil, errors.New("repository error"), repository.ErrMetadataAlreadyExists},
				},
				removeCalls: []bool{false, true, true},
				uploadResult: entity.OperationResult{
					Success: nil,
					Errors: map[string]string{
						"file1.txt": "failed to save file",
						"file2.txt": "failed to save file",
						"file3.txt": "file already exists",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := mocks.NewMockStorage(t)
			for i := 0; i < len(tt.got.files); i++ {
				mockStorage.EXPECT().
					Save(tt.got.files[i].Name, mock.Anything, mock.Anything).
					Return("", 0, tt.want.storage.errors[i]).
					Once()

				if tt.want.removeCalls[i] {
					mockStorage.EXPECT().
						Remove(mock.Anything).
						Return(nil).
						Once()
				}
			}

			mockRepository := mocks.NewMockRepository(t)
			for i := 0; i < len(tt.got.files); i++ {
				if tt.want.storage.errors[i] == nil {
					mockRepository.EXPECT().
						SaveFileMetadata(mock.Anything).
						Return(tt.want.repo.errors[i]).
						Once()
				}
			}

			log := slog.New(slog.DiscardHandler)

			cfg := &config.Config{}
			service := New(mockRepository, mockStorage, cfg)

			mockUserID := uuid.New()

			var result entity.OperationResult
			service.Upload(tt.got.files, &result, mockUserID, log)

			if !equalUploadResult(result, tt.want.uploadResult) {
				t.Fatalf("results not equal. Expected: %v, Got: %v", tt.want.uploadResult, result)
			}
		})
	}
}

func equalUploadResult(a, b entity.OperationResult) bool {
	if !slices.Equal(a.Success, b.Success) {
		return false
	}

	if !maps.Equal(a.Errors, b.Errors) {
		return false
	}

	return true
}
