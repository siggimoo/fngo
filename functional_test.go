package fngo

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParallelMap(t *testing.T) {
	expectedLengths := map[int]any{
		3: true,
		4: true,
		5: true,
		6: true,
		7: true,
	}

	tests := []struct {
		name          string
		mapError      error
		cancelContext bool
		expectedError error
	}{
		{"nominal", nil, false, nil},
		{"mapError", assert.AnError, false, assert.AnError},
		{"masterContextCanceled", nil, true, context.Canceled},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if test.cancelContext {
				cancel()
			}

			names := SliceSource(ctx, []string{"alice", "bob", "charlie", "darren", "erin"})

			mappedLengths := ParallelMap(names, func(_ context.Context, name string) (int, error) {
				return len(name), test.mapError
			})

			actualLengths := make(map[int]any)
			err := Sink(mappedLengths, func(_ context.Context, length int) error {
				actualLengths[length] = true
				return nil
			})

			assert.Equal(t, test.expectedError, err, "wrong error")

			if test.expectedError == nil {
				assert.Equal(t, expectedLengths, actualLengths, "wrong lengths")
			}
		})
	}
}

func TestPipeline(t *testing.T) {
	expectedNames := []string{
		"alice",
		"alice",
		"alice",
		"alice",
		"alice",
		"charlie",
		"charlie",
		"charlie",
		"charlie",
		"charlie",
		"charlie",
		"charlie",
		"david",
		"david",
		"david",
		"david",
		"david",
	}

	tests := []struct {
		name          string
		sourceError   error
		filterError   error
		mapError      error
		sinkError     error
		cancelContext bool
		expectedError error
	}{
		{"nominal", nil, nil, nil, nil, false, nil},
		{"sourceError", assert.AnError, nil, nil, nil, false, assert.AnError},
		{"filterError", nil, assert.AnError, nil, nil, false, assert.AnError},
		{"mapError", nil, nil, assert.AnError, nil, false, assert.AnError},
		{"sinkError", nil, nil, nil, assert.AnError, false, assert.AnError},
		{"masterContextCanceled", nil, nil, nil, nil, true, context.Canceled},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if test.cancelContext {
				cancel()
			}

			names := Source(ctx, func(_ context.Context, emit func(string) error) error {
				for _, name := range []string{"alice", "bob", "charlie", "david", "erin"} {
					if err := emit(name); err != nil {
						return err
					}
				}

				return test.sourceError
			})

			namesWithA := Filter(names, func(_ context.Context, name string) (bool, error) {
				return strings.ContainsRune(name, 'a'), test.filterError
			})

			repeatedNames := Map(namesWithA, func(_ context.Context, name string) ([]string, error) {
				output := make([]string, len(name))

				for i := range output {
					output[i] = name
				}

				return output, test.mapError
			})

			flattenedNames := Flatten(repeatedNames)

			actualNames := make([]string, 0)
			err := Sink(flattenedNames, func(_ context.Context, name string) error {
				actualNames = append(actualNames, name)
				return test.sinkError
			})

			assert.Equal(t, test.expectedError, err, "wrong error")

			if test.expectedError == nil {
				assert.Equal(t, expectedNames, actualNames, "wrong names")
			}
		})
	}
}

func TestSliceSource(t *testing.T) {
	expectedNames := []string{"alice", "bob", "charlie", "david", "erin"}

	tests := []struct {
		name          string
		cancelContext bool
		expectedError error
	}{
		{"nominal", false, nil},
		{"masterContextCanceled", true, context.Canceled},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if test.cancelContext {
				cancel()
			}

			names := SliceSource(ctx, expectedNames)

			actualNames := make([]string, 0)
			err := Sink(names, func(_ context.Context, name string) error {
				actualNames = append(actualNames, name)
				return nil
			})

			assert.Equal(t, test.expectedError, err, "wrong error")

			if test.expectedError == nil {
				assert.Equal(t, expectedNames, actualNames, "wrong names")
			}
		})
	}
}
