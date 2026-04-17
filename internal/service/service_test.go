package service

import (
	"errors"
	"log/slog"
	"maps"
	"slices"
	"testing"

	"github.com/antongolenev23/tuchka-server/internal/file"
	"github.com/antongolenev23/tuchka-server/internal/repository"
	"github.com/antongolenev23/tuchka-server/internal/storage"
	"github.com/stretchr/testify/mock"
)

type storageWant struct {
	errors []error
}

type repositoryWant struct {
	errors []error
}

type got struct {
	files []file.File
}

type want struct {
	storage      storageWant
	repo         repositoryWant
	removeCalls  []bool
	uploadResult file.Result
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
				files: []file.File{
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
				uploadResult: file.Result{
					Success: []string{"file1.txt", "file2.txt"},
					Errors:  nil,
				},
			},
		},
		{
			name: "storage and repository errors while file saving",
			got: got{
				files: []file.File{
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
				uploadResult: file.Result{
					Success: nil,
					Errors: map[string]string{
						"file1.txt": "internal server error",
						"file2.txt": "internal server error",
						"file3.txt": "file already exists",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := storage.NewMockStorage(t)
			for i := 0; i < len(tt.got.files); i++ {
				mockStorage.EXPECT().
					Save(tt.got.files[i].Name, mock.Anything).
					Return("", 0, tt.want.storage.errors[i]).
					Once()

				if tt.want.removeCalls[i] {
					mockStorage.EXPECT().
						Remove(mock.Anything).
						Return(nil).
						Once()
				}
			}

			mockRepository := repository.NewMockRepository(t)
			for i := 0; i < len(tt.got.files); i++ {
				if tt.want.storage.errors[i] == nil {
					mockRepository.EXPECT().
						SaveFileMetadata(mock.Anything).
						Return(tt.want.repo.errors[i]).
						Once()
				}
			}

			log := slog.New(slog.DiscardHandler)

			service := New(mockRepository, mockStorage)

			var result file.Result
			service.Upload(tt.got.files, &result, log)

			if !equalUploadResult(result, tt.want.uploadResult) {
				t.Fatalf("results not equal. Expected: %v, Got: %v", tt.want.uploadResult, result)
			}
		})
	}
}

func equalUploadResult(a, b file.Result) bool {
	if !slices.Equal(a.Success, b.Success) {
		return false
	}

	if !maps.Equal(a.Errors, b.Errors) {
		return false
	}

	return true
}
