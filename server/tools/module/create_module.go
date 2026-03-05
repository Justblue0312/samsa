package module

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	BaseDir, _    = os.Getwd()
	AppDir        = filepath.Join(BaseDir, "internal", "feature")
	FilesToCreate = []string{"repository", "http_handler", "usecase", "register", "models"}
)

type writeFunc func(file *os.File, name string, caser cases.Caser)

func CreateModule(name string, ignoreFiles []string, override bool) error {
	moduleDir := filepath.Join(AppDir, strings.ToLower(name))

	if !override {
		if _, err := os.Stat(moduleDir); err == nil {
			return fmt.Errorf("module %q already exists, use override=true to overwrite", name)
		}
	}

	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return fmt.Errorf("failed to create module directory: %w", err)
	}

	caser := cases.Title(language.English)

	filenames := make([]string, 0, 5)
	for _, filename := range FilesToCreate {
		if !slices.Contains(ignoreFiles, filename) {
			filenames = append(filenames, filename)
		}
	}

	errCh := make(chan error, len(filenames))
	var wg sync.WaitGroup
	for _, filename := range filenames {
		wg.Add(1)
		go func(filename string) {
			defer wg.Done()

			writer := getWriter(filename)
			if writer == nil {
				errCh <- fmt.Errorf("unknown file type: %s", filename)
				return
			}

			filePath := filepath.Join(moduleDir, filename+".go")
			if err := createFile(filePath, name, caser, writer, override); err != nil {
				errCh <- err
			}
		}(filename)
	}

	// Close the channel once all goroutines finish
	wg.Wait()
	close(errCh)

	var merr error
	for err := range errCh {
		merr = multierror.Append(merr, err)
	}

	return merr
}

func getWriter(filename string) writeFunc {
	switch filename {
	case "repository":
		return writeRepository
	case "http_handler":
		return writeHandler
	case "usecase":
		return writeUsecase
	case "register":
		return writeRegister
	case "models":
		return writeModel
	default:
		return nil
	}
}

func createFile(path, name string, caser cases.Caser, write writeFunc, override bool) error {
	if !override {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file %q already exists, skipping", path)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer file.Close()

	write(file, name, caser)
	return nil
}

func writeRepository(file *os.File, name string, _ cases.Caser) {
	lower := strings.ToLower(name)
	_, _ = file.WriteString("package " + lower + "\n\n")
	_, _ = file.WriteString("//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks\n\n")
	_, _ = file.WriteString("type Repository interface {\n}\n\n")
	_, _ = file.WriteString("type repository struct{}\n\n")
	_, _ = file.WriteString("func NewRepository() Repository {\n\treturn &repository{}\n}\n")
}

func writeHandler(file *os.File, name string, _ cases.Caser) {
	lower := strings.ToLower(name)
	_, _ = file.WriteString("package " + lower + "\n\n")
	// _, _ = file.WriteString("//go:generate mockgen -destination=mocks/mock_http_handler.go -source=http_handler.go -package=mocks\n\n")
	_, _ = file.WriteString("type HTTPHandler struct {\n}\n\n")
	_, _ = file.WriteString("func NewHTTPHandler() *HTTPHandler {\n\treturn &HTTPHandler{}\n}\n")
}

func writeUsecase(file *os.File, name string, _ cases.Caser) {
	lower := strings.ToLower(name)
	_, _ = file.WriteString("package " + lower + "\n\n")
	_, _ = file.WriteString("//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks\n\n")
	_, _ = file.WriteString("type UseCase interface {\n}\n\n")
	_, _ = file.WriteString("type usecase struct{}\n\n")
	_, _ = file.WriteString("func NewUseCase() UseCase {\n\treturn &usecase{}\n}\n")
}

func writeRegister(file *os.File, name string, caser cases.Caser) {
	titled := caser.String(name)
	lower := strings.ToLower(name)
	_, _ = file.WriteString("package " + lower + "\n\n")
	_, _ = file.WriteString("import \"github.com/go-chi/chi/v5\"\n\n")
	_, _ = file.WriteString("func Register" + titled + "Routes(router chi.Router) {\n}\n")
}

func writeModel(file *os.File, name string, _ cases.Caser) {
	_, _ = file.WriteString("package " + strings.ToLower(name) + "\n")
}
